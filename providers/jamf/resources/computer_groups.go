// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/jamf/connection"
)

func (r *mqlJamf) smartGroups() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.JamfConnection)
	client := conn.Client

	groups, err := client.GetComputerGroups()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for _, g := range groups.Results {
		item, err := CreateResource(r.MqlRuntime, "computerGroups", map[string]*llx.RawData{
			"id":         llx.IntData(g.ID),
			"name":       llx.StringData(g.Name),
			"smartGroup": llx.BoolData(g.IsSmart),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, item.(*mqlComputerGroups))
	}

	return res, nil
}

func (u *mqlComputerGroups) id() (string, error) {
	if u == nil {
		return "", errors.New("no id")
	}
	return u.Name.Data, nil
}
