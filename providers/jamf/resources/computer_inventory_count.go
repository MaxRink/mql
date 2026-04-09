// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

func (r *mqlJamf) computerInventoryCount() (int64, error) {
	inv := r.GetComputerInventory()
	if inv.Error != nil {
		return 0, inv.Error
	}
	return int64(len(inv.Data)), nil
}
