// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/jamf/connection"
)

func (r *mqlJamf) users() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.JamfConnection)
	client := conn.Client

	users, err := client.GetUsers()
	if err != nil {
		return nil, err
	}

	var res []interface{}
	for _, u := range users.Users {
		item, err := CreateResource(r.MqlRuntime, "jamfUsers", map[string]*llx.RawData{
			"id":   llx.IntData(u.ID),
			"name": llx.StringData(u.Name),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, item.(*mqlJamfUsers))
	}
	return res, nil
}

func (u *mqlJamfUsers) id() (string, error) {
	if u == nil {
		return "", errors.New("no id")
	}
	return u.Name.Data, nil
}
