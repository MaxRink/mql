// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"net/url"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/jamf/connection"
)

func (r *mqlJamf) computerInventory() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.JamfConnection)
	client := conn.Client

	inventory, err := client.GetComputersInventory(url.Values{})
	if err != nil {
		return nil, err
	}

	var res []interface{}
	for _, c := range inventory.Results {
		item, err := CreateResource(r.MqlRuntime, "jamfComputer", map[string]*llx.RawData{
			"id":                                   llx.StringData(c.ID),
			"name":                                 llx.StringData(c.General.Name),
			"make":                                 llx.StringData(c.Hardware.Make),
			"model":                                llx.StringData(c.Hardware.Model),
			"operatingSystemName":                  llx.StringData(c.OperatingSystem.Name),
			"operatingSystemVersion":               llx.StringData(c.OperatingSystem.Version),
			"macAddress":                           llx.StringData(c.Hardware.MacAddress),
			"serialNumber":                         llx.StringData(c.Hardware.SerialNumber),
			"processorType":                        llx.StringData(c.Hardware.ProcessorType),
			"processorCount":                       llx.IntData(c.Hardware.ProcessorCount),
			"coreCount":                            llx.IntData(c.Hardware.CoreCount),
			"totalRamMegabytes":                    llx.IntData(c.Hardware.TotalRamMegabytes),
			"lastIpAddress":                        llx.StringData(c.General.LastIpAddress),
			"lastReportedIp":                       llx.StringData(c.General.LastReportedIp),
			"jamfBinaryVersion":                    llx.StringData(c.General.JamfBinaryVersion),
			"platform":                             llx.StringData(c.General.Platform),
			"reportDate":                           llx.StringData(c.General.ReportDate),
			"lastContactTime":                      llx.StringData(c.General.LastContactTime),
			"lastEnrolledDate":                     llx.StringData(c.General.LastEnrolledDate),
			"initialEntryDate":                     llx.StringData(c.General.InitialEntryDate),
			"itunesStoreAccountActive":             llx.BoolData(c.General.ItunesStoreAccountActive),
			"enrolledViaAutomatedDeviceEnrollment": llx.BoolData(c.General.EnrolledViaAutomatedDeviceEnrollment),
			"fileVault2Enabled":                    llx.StringData(c.OperatingSystem.FileVault2Status),
			"autoLoginDisabled":                    llx.BoolData(c.Security.AutoLoginDisabled),
			"activationLockEnabled":                llx.BoolData(c.Security.ActivationLockEnabled),
			"firewallEnabled":                      llx.BoolData(c.Security.FirewallEnabled),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, item.(*mqlJamfComputer))
	}

	return res, nil
}

func (c *mqlJamfComputer) id() (string, error) {
	if c == nil {
		return "", errors.New("no id")
	}
	return c.Name.Data, nil
}

func (c *mqlJamfComputer) localUserAccounts() ([]interface{}, error) {
	conn := c.MqlRuntime.Connection.(*connection.JamfConnection)
	client := conn.Client

	inventory, err := client.GetComputerInventoryByID(c.Id.Data)
	if err != nil {
		return nil, err
	}

	var users []interface{}
	for _, user := range inventory.LocalUserAccounts {
		userItem, err := CreateResource(c.MqlRuntime, "jamfLocalUserAccount", map[string]*llx.RawData{
			"uid":                          llx.StringData(user.UID),
			"username":                     llx.StringData(user.Username),
			"fullName":                     llx.StringData(user.FullName),
			"admin":                        llx.BoolData(user.Admin),
			"fileVault2Enabled":            llx.BoolData(user.FileVault2Enabled),
			"userAccountType":              llx.StringData(user.UserAccountType),
			"passwordMaxAge":               llx.IntData(int64(user.PasswordMaxAge)),
			"homeDirectory":                llx.StringData(user.HomeDirectory),
			"passwordMinLength":            llx.IntData(int64(user.PasswordMinLength)),
			"passwordMinComplexCharacters": llx.IntData(int64(user.PasswordMinComplexCharacters)),
		})
		if err != nil {
			return nil, err
		}
		users = append(users, userItem)
	}
	return users, nil
}
