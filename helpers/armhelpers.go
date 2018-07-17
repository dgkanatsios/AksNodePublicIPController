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

func getIPClient() network.PublicIPAddressesClient {
	ipClient := network.NewPublicIPAddressesClient(spDetails.SubscriptionID)
	auth, _ := GetResourceManagementAuthorizer()
	ipClient.Authorizer = auth
	return ipClient
}

func getVMClient() compute.VirtualMachinesClient {
	vmClient := compute.NewVirtualMachinesClient(spDetails.SubscriptionID)
	auth, _ := GetResourceManagementAuthorizer()
	vmClient.Authorizer = auth
	return vmClient
}

func getNicClient() network.InterfacesClient {
	nicClient := network.NewInterfacesClient(spDetails.SubscriptionID)
	auth, _ := GetResourceManagementAuthorizer()
	nicClient.Authorizer = auth
	return nicClient
}

func createPublicIP(ctx context.Context, ipName string) (ip network.PublicIPAddress, err error) {
	ipClient := getIPClient()
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
		return ip, fmt.Errorf("cannot create public ip address: %v", err)
	}

	err = future.WaitForCompletion(ctx, ipClient.Client)
	if err != nil {
		return ip, fmt.Errorf("cannot get public ip address create or update future response: %v", err)
	}

	return future.Result(ipClient)
}

func getVM(ctx context.Context, vmName string) (compute.VirtualMachine, error) {
	vmClient := getVMClient()
	return vmClient.Get(ctx, spDetails.ResourceGroup, vmName, compute.InstanceView)
}

func getNetworkInterface(ctx context.Context, vmName string) (*network.Interface, error) {
	vm, err := getVM(ctx, vmName)
	if err != nil {
		return nil, err
	}

	//this will be something like /subscriptions/6bd0e514-c783-4dac-92d2-6788744eee7a/resourceGroups/MC_akslala_akslala_westeurope/providers/Microsoft.Network/networkInterfaces/aks-nodepool1-26427378-nic-0
	nicIDFullName := &(*vm.NetworkProfile.NetworkInterfaces)[0].ID

	nicID := getResourceID(**nicIDFullName)

	nicClient := getNicClient()

	networkInterface, err := nicClient.Get(ctx, spDetails.ResourceGroup, nicID, "")
	return &networkInterface, err
}

// CreateOrUpdateVMPulicIP will create a new Public IP and assign it to the Virtual Machine
func CreateOrUpdateVMPulicIP(ctx context.Context, vmName string, ipName string) error {

	log.Infof("Trying to get NIC from the VM %s", vmName)

	nic, err := getNetworkInterface(ctx, vmName)
	if err != nil {
		return fmt.Errorf("cannot get network interface: %v", err)
	}

	log.Infof("Trying to create the Public IP for Node %s", vmName)

	ip, err := createPublicIP(ctx, ipName)
	if err != nil {
		return fmt.Errorf("cannot create public IP for Node %s: %v", vmName, err)
	}

	log.Infof("Public IP for Node %s created", vmName)

	(*nic.IPConfigurations)[0].PublicIPAddress = &ip

	nicClient := getNicClient()

	log.Infof("Trying to assign the Public IP to the NIC for Node %s", vmName)

	future, err := nicClient.CreateOrUpdate(ctx, spDetails.ResourceGroup, getResourceID(*nic.ID), *nic)

	if err != nil {
		return fmt.Errorf("cannot update NIC for Node %s: %v", vmName, err)
	}

	err = future.WaitForCompletion(ctx, nicClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get NIC create or update future response for Node %s: %v", vmName, err)
	}

	log.Infof("NIC for Node %s successfully updated", vmName)

	return nil
}

// DeletePublicIP deletes the designated Public IP
func DeletePublicIP(ctx context.Context, ipName string) error {
	ipClient := getIPClient()
	future, err := ipClient.Delete(ctx, spDetails.ResourceGroup, ipName)
	if err != nil {
		return fmt.Errorf("cannot delete public ip address %s: %v", ipName, err)
	}

	err = future.WaitForCompletion(ctx, ipClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get public ip address %s create or update future response: %v", ipName, err)
	}

	log.Infof("IP %s successfully deleted", ipName)

	return nil
}

func getResourceID(fullID string) string {
	parts := strings.Split(fullID, "/")
	return parts[len(parts)-1]
}
