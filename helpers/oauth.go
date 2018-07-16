// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"log"
	"net/url"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

var (
	// for service principal and device
	oauthConfig        *adal.OAuthConfig
	armAuthorizer      autorest.Authorizer
	batchAuthorizer    autorest.Authorizer
	graphAuthorizer    autorest.Authorizer
	keyvaultAuthorizer autorest.Authorizer

	// for device
	nativeClientID string
)

// OAuthGrantType specifies which grant type to use.
type OAuthGrantType int

const (
	// OAuthGrantTypeServicePrincipal for client credentials flow
	OAuthGrantTypeServicePrincipal OAuthGrantType = iota
	// OAuthGrantTypeDeviceFlow for device-auth flow
	OAuthGrantTypeDeviceFlow
)

// AuthGrantType returns what kind of authentication is going to be used: device flow or service principal
func AuthGrantType() OAuthGrantType {
	if DeviceFlow() {
		return OAuthGrantTypeDeviceFlow
	}
	return OAuthGrantTypeServicePrincipal
}

// GetResourceManagementAuthorizer gets an OAuth token for managing resources using the specified grant type.
func GetResourceManagementAuthorizer(grantType OAuthGrantType) (a autorest.Authorizer, err error) {
	if armAuthorizer != nil {
		return armAuthorizer, nil
	}

	switch grantType {
	case OAuthGrantTypeServicePrincipal:
		a, err = auth.NewAuthorizerFromEnvironment()
	case OAuthGrantTypeDeviceFlow:
		config := auth.NewDeviceFlowConfig(nativeClientID, spDetails.TenantID)
		a, err = config.Authorizer()
	default:
		log.Fatalln("invalid token type specified")
	}

	if err == nil {
		armAuthorizer = a
	}
	return
}

// GetBatchAuthorizer gets an authorizer for Azure batch using the specified grant type.
func GetBatchAuthorizer(grantType OAuthGrantType) (a autorest.Authorizer, err error) {
	if batchAuthorizer != nil {
		return batchAuthorizer, nil
	}

	a, err = getAuthorizer(grantType, Environment().BatchManagementEndpoint)
	if err == nil {
		batchAuthorizer = a
	}

	return
}

// GetGraphAuthorizer gets an authorizer for the graphrbac API using the specified grant type.
func GetGraphAuthorizer(grantType OAuthGrantType) (a autorest.Authorizer, err error) {
	if graphAuthorizer != nil {
		return graphAuthorizer, nil
	}

	a, err = getAuthorizer(grantType, Environment().GraphEndpoint)
	if err == nil {
		graphAuthorizer = a
	}

	return
}

// GetResourceManagementTokenHybrid retrieves auth token for hybrid environment
func GetResourceManagementTokenHybrid(activeDirectoryEndpoint, tokenAudience string) (adal.OAuthTokenProvider, error) {
	var token adal.OAuthTokenProvider
	oauthConfig, err := adal.NewOAuthConfig(activeDirectoryEndpoint, spDetails.TenantID)
	token, err = adal.NewServicePrincipalToken(
		*oauthConfig,
		spDetails.AadClientID,
		spDetails.AadClientSecret,
		tokenAudience)

	return token, err
}

func getAuthorizer(grantType OAuthGrantType, endpoint string) (a autorest.Authorizer, err error) {
	switch grantType {
	case OAuthGrantTypeServicePrincipal:
		token, err := adal.NewServicePrincipalToken(*oauthConfig, spDetails.AadClientID, spDetails.AadClientSecret, endpoint)
		if err != nil {
			return a, err
		}
		a = autorest.NewBearerAuthorizer(token)
	case OAuthGrantTypeDeviceFlow:
		config := auth.NewDeviceFlowConfig(nativeClientID, spDetails.TenantID)
		config.Resource = endpoint
		a, err = config.Authorizer()
	default:
		log.Fatalln("invalid token type specified")
	}
	return
}

// GetKeyvaultAuthorizer gets an authorizer for the keyvault dataplane
func GetKeyvaultAuthorizer(grantType OAuthGrantType) (a autorest.Authorizer, err error) {
	if keyvaultAuthorizer != nil {
		return keyvaultAuthorizer, nil
	}

	vaultEndpoint := strings.TrimSuffix(Environment().KeyVaultEndpoint, "/")
	config, err := adal.NewOAuthConfig(Environment().ActiveDirectoryEndpoint, spDetails.TenantID)
	updatedAuthorizeEndpoint, err := url.Parse("https://login.windows.net/" + spDetails.TenantID + "/oauth2/token")
	config.AuthorizeEndpoint = *updatedAuthorizeEndpoint
	if err != nil {
		return
	}

	switch grantType {
	case OAuthGrantTypeServicePrincipal:
		token, err := adal.NewServicePrincipalToken(*config, spDetails.AadClientID, spDetails.AadClientSecret, vaultEndpoint)
		if err != nil {
			return a, err
		}
		a = autorest.NewBearerAuthorizer(token)
	case OAuthGrantTypeDeviceFlow:
		deviceConfig := auth.NewDeviceFlowConfig(nativeClientID, spDetails.TenantID)
		deviceConfig.Resource = vaultEndpoint
		deviceConfig.AADEndpoint = updatedAuthorizeEndpoint.String()
		a, err = deviceConfig.Authorizer()
	default:
		log.Fatalln("invalid token type specified")
	}

	if err == nil {
		keyvaultAuthorizer = a
	}

	return
}
