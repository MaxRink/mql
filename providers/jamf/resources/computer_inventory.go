// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"net/url"
	"strconv"
	"time"

	"github.com/deploymenttheory/go-api-sdk-jamfpro/sdk/jamfpro"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/jamf/connection"
)

// parseJamfTime parses a Jamf Pro API date string (ISO 8601 / RFC 3339).
// Returns nil if the string is empty or unparseable.
func parseJamfTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

func (r *mqlJamf) computerInventory() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.JamfConnection)
	client := conn.Client

	const pageSize = 100
	var res []interface{}
	page := 0

	for {
		params := url.Values{}
		params.Set("page", strconv.Itoa(page))
		params.Set("page-size", strconv.Itoa(pageSize))

		inventory, err := client.GetComputersInventory(params)
		if err != nil {
			return nil, err
		}
		if inventory == nil || len(inventory.Results) == 0 {
			break
		}

		for _, c := range inventory.Results {
			// Cache local user accounts from the bulk fetch so that
			// localUserAccounts() can return them without an extra API call.
			conn.CacheLocalUserAccounts(c.ID, c.LocalUserAccounts)

			item, err := CreateResource(r.MqlRuntime, "jamf.computer", map[string]*llx.RawData{
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
				"reportDate":                           llx.TimeDataPtr(parseJamfTime(c.General.ReportDate)),
				"lastContactTime":                      llx.TimeDataPtr(parseJamfTime(c.General.LastContactTime)),
				"lastEnrolledDate":                     llx.TimeDataPtr(parseJamfTime(c.General.LastEnrolledDate)),
				"initialEntryDate":                     llx.TimeDataPtr(parseJamfTime(c.General.InitialEntryDate)),
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
			res = append(res, item)
		}

		if len(res) >= inventory.TotalCount {
			break
		}
		page++
	}

	return res, nil
}

func (c *mqlJamfComputer) id() (string, error) {
	return "jamf.computer/" + c.Id.Data, nil
}

func (c *mqlJamfComputer) localUserAccounts() ([]interface{}, error) {
	conn := c.MqlRuntime.Connection.(*connection.JamfConnection)

	// Use cached data from the initial inventory fetch if available.
	if cached, ok := conn.GetCachedLocalUserAccounts(c.Id.Data); ok {
		return createLocalUserAccountResources(c.MqlRuntime, cached)
	}

	// Fallback to individual API call for computers not fetched via bulk inventory.
	client := conn.Client
	inventory, err := client.GetComputerInventoryByID(c.Id.Data)
	if err != nil {
		return nil, err
	}
	if inventory == nil {
		return nil, nil
	}
	return createLocalUserAccountResources(c.MqlRuntime, inventory.LocalUserAccounts)
}

func createLocalUserAccountResources(runtime *plugin.Runtime, accounts []jamfpro.ComputerInventorySubsetLocalUserAccount) ([]interface{}, error) {
	var res []interface{}
	for _, user := range accounts {
		item, err := CreateResource(runtime, "jamf.localUserAccount", map[string]*llx.RawData{
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
		res = append(res, item)
	}
	return res, nil
}
