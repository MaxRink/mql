// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/providers/jamf/connection"
)

func (r *mqlJamf) version() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.JamfConnection)
	client := conn.Client

	info, err := client.GetJamfProVersion()
	if err != nil {
		return "", err
	}
	return *info.Version, nil
}
