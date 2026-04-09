// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/jamf/connection"
)

func (r *mqlJamf) packages() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.JamfConnection)
	client := conn.Client

	inventory, err := client.GetPackages("id:asc", "")
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for _, c := range inventory.Results {
		item, err := CreateResource(r.MqlRuntime, "jamfPackages", map[string]*llx.RawData{
			"name":                llx.StringData(c.PackageName),
			"fileName":            llx.StringData(c.FileName),
			"oSInstall":           llx.BoolDataPtr(c.OSInstall),
			"categoryId":          llx.StringData(c.CategoryID),
			"priority":            llx.IntData(c.Priority),
			"suppressUpdates":     llx.BoolData(*c.SuppressUpdates),
			"supressRegistration": llx.BoolData(*c.SuppressRegistration),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, item.(*mqlJamfPackages))
	}

	return res, nil
}

func (c *mqlJamfPackages) id() (string, error) {
	if c == nil {
		return "", errors.New("no id")
	}
	return c.Name.Data, nil
}
