// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// ServicePrincipalDetails contains the Service Principal credentials for the AKS cluster
type ServicePrincipalDetails struct {
	TenantID        string
	SubscriptionID  string
	AadClientID     string
	AadClientSecret string
	Location        string
	ResourceGroup   string
}

var spDetails ServicePrincipalDetails

/*
/etc/kubernetes/azure.json is ...

{
    "cloud":"AzurePublicCloud",
    "tenantId": "XXX",
    "subscriptionId": "XXX",
    "aadClientId": "XXXX",
    "aadClientSecret": "XXXXX",
    "resourceGroup": "MC_akslala_akslala_westeurope",
    "location": "westeurope",
	...
}
*/

// InitializeServicePrincipalDetails reads the /etc/kubernetes/azure.json file on the host (mounted via hostPath on the Pod)
// this files contains the credentials for the AKS cluster's Service Principal
func InitializeServicePrincipalDetails() error {

	if os.Getenv("TENANT_ID") != "" && os.Getenv("SUBSCRIPTION_ID") != "" && os.Getenv("AAD_CLIENT_ID") != "" && os.Getenv("AAD_CLIENT_SECRET") != "" && os.Getenv("LOCATION") != "" && os.Getenv("RESOURCE_GROUP") != "" {
		spDetails = ServicePrincipalDetails{
			TenantID:        os.Getenv("TENANT_ID"),
			SubscriptionID:  os.Getenv("SUBSCRIPTION_ID"),
			AadClientID:     os.Getenv("AAD_CLIENT_ID"),
			AadClientSecret: os.Getenv("AAD_CLIENT_SECRET"),
			Location:        os.Getenv("LOCATION"),
			ResourceGroup:   os.Getenv("RESOURCE_GROUP"),
		}
		return nil
	}

	file, e := ioutil.ReadFile("/akssp/azure.json")
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		return e
	}
	var f interface{}
	err := json.Unmarshal(file, &f)

	if err != nil {
		fmt.Printf("Unmarshaling error: %v\n", err)
		return err
	}

	m := f.(map[string]interface{})

	spDetails = ServicePrincipalDetails{
		TenantID:        m["tenantId"].(string),
		SubscriptionID:  m["subscriptionId"].(string),
		AadClientID:     m["aadClientId"].(string),
		AadClientSecret: m["aadClientSecret"].(string),
		Location:        m["location"].(string),
		ResourceGroup:   m["resourceGroup"].(string),
	}

	return nil
}

// GetPublicIPName returns the name of the Public IP resource, which is based on the Node's name
func GetPublicIPName(vmName string) string {
	return "ipconfig-" + vmName
}
