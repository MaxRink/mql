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

	// Fetch a minimal page to read TotalCount without loading all records
	params := url.Values{}
	params.Set("page", "0")
	params.Set("page-size", "1")

	inventory, err := client.GetComputersInventory(params)
	if err != nil {
		return 0, err
	}
	if inventory == nil {
		return 0, nil
	}
	return int64(inventory.TotalCount), nil
}
