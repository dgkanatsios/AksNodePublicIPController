package helpers

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-03-30/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	log "github.com/Sirupsen/logrus"
)

func getIPClient() network.PublicIPAddressesClient {
	ipClient := network.NewPublicIPAddressesClient(spDetails.SubscriptionID)
	auth, _ := GetResourceManagementAuthorizer(AuthGrantType())
	ipClient.Authorizer = auth
	ipClient.AddToUserAgent(UserAgent())
	return ipClient
}

func getVMClient() compute.VirtualMachinesClient {
	vmClient := compute.NewVirtualMachinesClient(spDetails.SubscriptionID)
	auth, _ := GetResourceManagementAuthorizer(AuthGrantType())
	vmClient.Authorizer = auth
	vmClient.AddToUserAgent(UserAgent())
	return vmClient
}

func getNicClient() network.InterfacesClient {
	nicClient := network.NewInterfacesClient(spDetails.SubscriptionID)
	auth, _ := GetResourceManagementAuthorizer(AuthGrantType())
	nicClient.Authorizer = auth
	nicClient.AddToUserAgent(UserAgent())
	return nicClient
}

func CreatePublicIP(ctx context.Context, ipName string) (ip network.PublicIPAddress, err error) {
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
				PublicIPAllocationMethod: network.Dynamic,
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

// GetVM gets the specified VM info
func GetVM(ctx context.Context, vmName string) (compute.VirtualMachine, error) {
	vmClient := getVMClient()
	return vmClient.Get(ctx, spDetails.ResourceGroup, vmName, compute.InstanceView)
}

func GetNetworkInterface(ctx context.Context, vmName string) (*network.Interface, error) {
	vm, err := GetVM(ctx, vmName)
	if err != nil {
		return nil, err
	}

	nicID := &(*vm.NetworkProfile.NetworkInterfaces)[0].ID

	nicClient := getNicClient()

	networkInterface, err := nicClient.Get(ctx, spDetails.ResourceGroup, **nicID, "")
	return &networkInterface, err
}

func UpdateVMNIC(ctx context.Context, vmName string, ipName string) error {

	log.Info("Trying to get NIC from the VM")

	nic, err := GetNetworkInterface(ctx, vmName)
	if err != nil {
		return fmt.Errorf("cannot get network interface: %v", err)
	}

	log.Info("Trying to create the Public IP")

	ip, err := CreatePublicIP(ctx, ipName)
	if err != nil {
		return fmt.Errorf("cannot create public IP: %v", err)
	}

	(*nic.IPConfigurations)[0].PublicIPAddress = &ip

	nicClient := getNicClient()

	log.Info("Trying to assign the Public IP to the NIC")

	future, err := nicClient.CreateOrUpdate(ctx, spDetails.ResourceGroup, *nic.ID, *nic)

	if err != nil {
		return fmt.Errorf("cannot create nic: %v", err)
	}

	err = future.WaitForCompletion(ctx, nicClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get nic create or update future response: %v", err)
	}

	return nil
}
