// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	// fmt.Printf("%s\n", m["tenantId"])
	// fmt.Printf("%s\n", m["subscriptionId"])
	// fmt.Printf("%s\n", m["aadClientId"])
	// fmt.Printf("%s\n", m["aadClientSecret"])
	// fmt.Printf("%s\n", m["location"])
	// fmt.Printf("%s\n", m["resourceGroup"])

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
