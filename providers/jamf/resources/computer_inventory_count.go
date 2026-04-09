// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"net/url"

	"go.mondoo.com/mql/v13/providers/jamf/connection"
)

func (r *mqlJamf) computerInventoryCount() (int64, error) {
	conn := r.MqlRuntime.Connection.(*connection.JamfConnection)
	client := conn.Client

	res, err := client.GetComputersInventory(url.Values{})
	if err != nil {
		return 0, err
	}
	return int64(res.TotalCount), nil
}
