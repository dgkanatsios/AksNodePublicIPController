// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

var (
	// for service principal and device
	//oauthConfig        *adal.OAuthConfig
	armAuthorizer      autorest.Authorizer
	batchAuthorizer    autorest.Authorizer
	graphAuthorizer    autorest.Authorizer
	keyvaultAuthorizer autorest.Authorizer

	// for device
	nativeClientID string
)

// GetResourceManagementAuthorizer gets an OAuth token for managing resources using Service Principal credentials
func GetResourceManagementAuthorizer() (a autorest.Authorizer, err error) {
	if armAuthorizer != nil {
		return armAuthorizer, nil
	}

	config := auth.NewClientCredentialsConfig(spDetails.AadClientID, spDetails.AadClientSecret, spDetails.TenantID)
	a, err = config.Authorizer()

	if err == nil {
		armAuthorizer = a
	}
	return
}
