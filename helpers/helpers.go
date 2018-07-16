// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/utils"
)

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

func InitializeServicePrincipalDetails() error {
	file, e := ioutil.ReadFile("/aks/azure.json")
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

	fmt.Printf("%s\n", m["tenantId"])
	fmt.Printf("%s\n", m["subscriptionId"])
	fmt.Printf("%s\n", m["aadClientId"])
	//fmt.Printf("%s\n", m["aadClientSecret"])
	fmt.Printf("%s\n", m["resourceGroup"])
	fmt.Printf("%s\n", m["location"])

	spDetails = ServicePrincipalDetails{
		TenantID:        m["tenantId"].(string),
		SubscriptionID:  m["subscriptionId"].(string),
		AadClientID:     m["aadClientId"].(string),
		AadClientSecret: m["aadClientSecret"].(string),
		Location:        m["location"].(string),
		ResourceGroup:   m["resourceGroup"].(string),
	}

	oauthConfig, err = adal.NewOAuthConfig(Environment().ActiveDirectoryEndpoint, spDetails.TenantID)
	return err

	return nil
}

// PrintAndLog writes to stdout and to a logger.
func PrintAndLog(message string) {
	log.Println(message)
	fmt.Println(message)
}

func contains(array []string, element string) bool {
	for _, e := range array {
		if e == element {
			return true
		}
	}
	return false
}

// UserAgent return the string to be appended to user agent header
func UserAgent() string {
	return "samples " + utils.GetCommit()
}

// ReadJSON reads a json file, and unmashals it.
// Very useful for template deployments.
func ReadJSON(path string) (*map[string]interface{}, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read template file: %v\n", err)
	}
	contents := make(map[string]interface{})
	json.Unmarshal(data, &contents)
	return &contents, nil
}
