package helpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-03-30/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"

	"github.com/Azure/go-autorest/autorest/to"

	log "github.com/Sirupsen/logrus"
)

func getIPClient() (*network.PublicIPAddressesClient, error) {
	ipClient := network.NewPublicIPAddressesClient(spDetails.SubscriptionID)
	auth, err := GetResourceManagementAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("error in getIPClient %s", err.Error())
	}
	ipClient.Authorizer = auth
	return &ipClient, nil
}

func getVMClient() (*compute.VirtualMachinesClient, error) {
	vmClient := compute.NewVirtualMachinesClient(spDetails.SubscriptionID)
	auth, err := GetResourceManagementAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("error in getVMClient %s", err.Error())
	}
	vmClient.Authorizer = auth
	return &vmClient, nil
}

func getNicClient() (*network.InterfacesClient, error) {
	nicClient := network.NewInterfacesClient(spDetails.SubscriptionID)
	auth, err := GetResourceManagementAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("error in getNicClient %s", err.Error())
	}
	nicClient.Authorizer = auth
	return &nicClient, nil
}

func createPublicIP(ctx context.Context, ipName string) (*network.PublicIPAddress, error) {
	ipClient, err := getIPClient()
	if err != nil {
		return nil, err
	}
	future, err := ipClient.CreateOrUpdate(
		ctx,
		spDetails.ResourceGroup,
		ipName,
		network.PublicIPAddress{
			Name:     to.StringPtr(ipName),
			Location: &spDetails.Location,
			PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
				PublicIPAddressVersion:   network.IPv4,
				PublicIPAllocationMethod: network.Dynamic, // IPv4 address created is a dynamic one
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("cannot create Public IP address: %v", err)
	}

	err = future.WaitForCompletion(ctx, ipClient.Client)
	if err != nil {
		return nil, fmt.Errorf("cannot get Public IP address CreateOrUpdate method response: %v", err)
	}

	ipAddr, err := future.Result(*ipClient)
	if err != nil {
		return nil, err
	}
	return &ipAddr, nil
}

func getVM(ctx context.Context, vmName string) (*compute.VirtualMachine, error) {
	vmClient, err := getVMClient()
	if err != nil {
		return nil, err
	}
	vm, err := vmClient.Get(ctx, spDetails.ResourceGroup, vmName, compute.InstanceView)
	if err != nil {
		return nil, err
	}
	return &vm, nil
}

func getNetworkInterface(ctx context.Context, vmName string) (*network.Interface, error) {
	vm, err := getVM(ctx, vmName)
	if err != nil {
		return nil, err
	}

	if vm.NetworkProfile == nil || len(*vm.NetworkProfile.NetworkInterfaces) == 0 {
		return nil, fmt.Errorf("Error. Network profile for VM %s is %v and len(vm.NetworkInterfaces)=%d", vmName, vm.NetworkProfile, len(*vm.NetworkProfile.NetworkInterfaces))
	}

	//this will be something like /subscriptions/6bd0e514-c783-4dac-92d2-6788744eee7a/resourceGroups/MC_akslala_akslala_westeurope/providers/Microsoft.Network/networkInterfaces/aks-nodepool1-26427378-nic-0
	nicFullName := &(*vm.NetworkProfile.NetworkInterfaces)[0].ID

	nicName := getResourceName(**nicFullName)

	nicClient, err := getNicClient()
	if err != nil {
		return nil, err
	}

	networkInterface, err := nicClient.Get(ctx, spDetails.ResourceGroup, nicName, "")
	return &networkInterface, err
}

type IPUpdater interface {
	CreateOrUpdateVMPulicIP(ctx context.Context, vmName string, ipName string) error
	DeletePublicIP(ctx context.Context, ipName string) error
	DisassociatePublicIPForNode(ctx context.Context, nodeName string) error
}
type IPUpdate struct{}

// CreateOrUpdateVMPulicIP will create a new Public IP and assign it to the Virtual Machine
func (*IPUpdate) CreateOrUpdateVMPulicIP(ctx context.Context, vmName string, ipName string) error {

	log.Infof("Trying to get NIC from the VM %s", vmName)

	nic, err := getNetworkInterface(ctx, vmName)
	if err != nil {
		return fmt.Errorf("cannot get network interface: %v", err)
	}

	log.Infof("Trying to create the Public IP for Node %s", vmName)

	ip, err := createPublicIP(ctx, ipName)
	if err != nil {
		return fmt.Errorf("Cannot create Public IP for Node %s: %v", vmName, err)
	}

	log.Infof("Public IP for Node %s created", vmName)

	// set this IP Address to NIC's IP configuration
	(*nic.IPConfigurations)[0].PublicIPAddress = ip

	nicClient, err := getNicClient()
	if err != nil {
		return err
	}

	log.Infof("Trying to assign the Public IP to the NIC for Node %s", vmName)

	future, err := nicClient.CreateOrUpdate(ctx, spDetails.ResourceGroup, getResourceName(*nic.ID), *nic)

	if err != nil {
		return fmt.Errorf("cannot update NIC for Node %s: %v", vmName, err)
	}

	err = future.WaitForCompletion(ctx, nicClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get NIC CreateOrUpdate response for Node %s: %v", vmName, err)
	}

	log.Infof("NIC for Node %s successfully updated", vmName)

	return nil
}

// DeletePublicIP deletes the designated Public IP
func (*IPUpdate) DeletePublicIP(ctx context.Context, ipName string) error {
	ipClient, err := getIPClient()
	if err != nil {
		return err
	}
	future, err := ipClient.Delete(ctx, spDetails.ResourceGroup, ipName)
	if err != nil {
		return fmt.Errorf("cannot delete Public IP address %s: %v", ipName, err)
	}

	err = future.WaitForCompletion(ctx, ipClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get public ip address %s CreateOrUpdate method's response: %v", ipName, err)
	}

	log.Infof("IP %s successfully deleted", ipName)

	return nil
}

// DisassociatePublicIPForNode will remove the Public IP address association from the VM's NIC
func (*IPUpdate) DisassociatePublicIPForNode(ctx context.Context, nodeName string) error {
	ipClient, err := getIPClient()
	if err != nil {
		return err
	}
	ipAddress, err := ipClient.Get(ctx, spDetails.ResourceGroup, GetPublicIPName(nodeName), "")
	if err != nil {
		return fmt.Errorf("cannot get IP Address: %v for Node %s", err, nodeName)
	}

	var nicName string
	if ipAddress.IPConfiguration != nil {
		ipConfiguration := *ipAddress.IPConfiguration.ID
		//ipConfiguration has a value similar to:
		///subscriptions/X/resourceGroups/Y/providers/Microsoft.Network/networkInterfaces/aks-nodepool1-26427378-nic-X/ipConfigurations/ipconfig1

		nicName = getNICNameFromIPConfiguration(ipConfiguration)
	} else {
		// IPConfiguration is nil => this IP address is already disassociated
		return nil
	}

	nicClient, err := getNicClient()
	if err != nil {
		return err
	}

	// get the NIC
	nic, err := nicClient.Get(ctx, spDetails.ResourceGroup, nicName, "")

	if err != nil {
		return fmt.Errorf("cannot get NIC for Node %s, error: %v", nodeName, err)
	}

	// set its Public IP to nil
	(*nic.IPConfigurations)[0].PublicIPAddress = nil

	// update the NIC so it has a nil Public IP
	future, err := nicClient.CreateOrUpdate(ctx, spDetails.ResourceGroup, getResourceName(*nic.ID), nic)

	if err != nil {
		return fmt.Errorf("cannot update NIC for Node %s, error: %v", nodeName, err)
	}

	err = future.WaitForCompletion(ctx, nicClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get NIC CreateOrUpdate response for Node %s, error: %v", nodeName, err)
	}

	// there is a chance that after the scale-in operation completes, the NIC will still be alive
	// This may happen due to a race condition between AKS calling Delete on the NIC and our code that
	// calls CreateOrUpdate
	// to make sure NIC gets removed, we'll just call delete on its instance
	futureDelete, err := nicClient.Delete(ctx, spDetails.ResourceGroup, getResourceName(*nic.ID))
	if err != nil {
		return fmt.Errorf("cannot delete NIC for Node %s, error: %v. NIC may have already been deleted", nodeName, err)
	}

	err = futureDelete.WaitForCompletion(ctx, nicClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get NIC Delete response for Node %s:, error: %v. NIC may have already been deleted", nodeName, err)
	}

	return nil
}

// getResourceName accepts a string of type
// /subscriptions/A/resourceGroups/B/providers/Microsoft.Network/publicIPAddresses/ipconfig-aks-nodepool1-X
// will return just the ID, i.e. ipconfig-aks-nodepool1-X
func getResourceName(fullID string) string {
	parts := strings.Split(fullID, "/")
	return parts[len(parts)-1]
}

func getNICNameFromIPConfiguration(ipConfig string) string {
	///subscriptions/X/resourceGroups/Y/providers/Microsoft.Network/networkInterfaces/aks-nodepool1-26427378-nic-X/ipConfigurations/ipconfig1
	parts := strings.Split(ipConfig, "/")
	return parts[len(parts)-3]
}
