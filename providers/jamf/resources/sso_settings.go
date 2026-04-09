// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/jamf/connection"
)

func (r *mqlJamf) sso() (*mqlSsoSettings, error) {
	conn := r.MqlRuntime.Connection.(*connection.JamfConnection)
	client := conn.Client

	info, err := client.GetSsoSettings()
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(r.MqlRuntime, "ssoSettings", map[string]*llx.RawData{
		"ssoEnabled":                                     llx.BoolData(info.SsoEnabled),
		"ssoForEnrollmentEnabled":                        llx.BoolData(info.SsoForEnrollmentEnabled),
		"ssoBypassAllowed":                               llx.BoolData(info.SsoBypassAllowed),
		"sessionTimeout":                                 llx.IntData(info.SessionTimeout),
		"ssoForMacOsSelfServiceEnabled":                  llx.BoolData(info.SsoForMacOsSelfServiceEnabled),
		"tokenExpirationDisabled":                        llx.BoolData(info.TokenExpirationDisabled),
		"userAttributeEnabled":                           llx.BoolData(info.UserAttributeEnabled),
		"userAttributeName":                              llx.StringData(info.UserAttributeName),
		"userMapping":                                    llx.StringData(info.UserMapping),
		"enrollmentSsoForAccountDrivenEnrollmentEnabled": llx.BoolData(info.EnrollmentSsoForAccountDrivenEnrollmentEnabled),
		"idpUrl":                       llx.StringData(info.IdpUrl),
		"idpProviderType":              llx.StringData(info.IdpProviderType),
		"groupEnrollmentAccessEnabled": llx.BoolData(info.GroupEnrollmentAccessEnabled),
		"groupAttributeName":           llx.StringData(info.GroupAttributeName),
		"entityId":                     llx.StringData(info.EntityId),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlSsoSettings), nil
}
