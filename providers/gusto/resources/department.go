// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/gusto/connection"
)

type mqlGustoDepartmentInternal struct {
	employeeUUIDs   []string
	contractorUUIDs []string
}

func (d *mqlGustoDepartment) id() (string, error) {
	return "gusto.department/" + d.Uuid.Data, d.Uuid.Error
}

func initGustoDepartment(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	uuidArg, ok := args["uuid"]
	if !ok || uuidArg == nil || uuidArg.Value == nil {
		return args, nil, nil
	}
	uuid, ok := uuidArg.Value.(string)
	if !ok || uuid == "" {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GustoConnection)
	companies, err := conn.ListCompanies(context.Background())
	if err != nil {
		return nil, nil, err
	}
	for _, company := range companies {
		departments, err := conn.ListDepartments(context.Background(), company.UUID)
		if err != nil {
			return nil, nil, err
		}
		for i := range departments {
			if departments[i].UUID != uuid {
				continue
			}
			// Build the resource directly so the Internal struct's
			// employeeUUIDs are populated — otherwise employees() would
			// fall back to re-listing the company's employees.
			dept, err := newMqlGustoDepartment(runtime, &departments[i])
			if err != nil {
				return nil, nil, err
			}
			return args, dept, nil
		}
	}
	return nil, nil, errors.New("gusto.department with uuid " + uuid + " not accessible with the configured token")
}

func newMqlGustoDepartment(runtime *plugin.Runtime, d *connection.Department) (*mqlGustoDepartment, error) {
	r, err := CreateResource(runtime, "gusto.department", map[string]*llx.RawData{
		"uuid":        llx.StringData(d.UUID),
		"companyUuid": llx.StringData(d.CompanyUUID),
		"name":        llx.StringData(d.Title),
	})
	if err != nil {
		return nil, err
	}
	dept := r.(*mqlGustoDepartment)
	dept.employeeUUIDs = make([]string, 0, len(d.EmployeeRefs))
	for _, ref := range d.EmployeeRefs {
		dept.employeeUUIDs = append(dept.employeeUUIDs, ref.UUID)
	}
	dept.contractorUUIDs = make([]string, 0, len(d.ContractorRef))
	for _, ref := range d.ContractorRef {
		dept.contractorUUIDs = append(dept.contractorUUIDs, ref.UUID)
	}
	return dept, nil
}

func (d *mqlGustoDepartment) employees() ([]any, error) {
	// If the parent department list populated employee UUIDs, resolve each
	// one lazily. Otherwise fall back to listing the company's employees and
	// filtering by department_uuid — needed when the department is selected
	// directly via `gusto.department(uuid: ...)`.
	if len(d.employeeUUIDs) > 0 {
		out := make([]any, 0, len(d.employeeUUIDs))
		for _, uuid := range d.employeeUUIDs {
			r, err := NewResource(d.MqlRuntime, "gusto.employee", map[string]*llx.RawData{
				"uuid": llx.StringData(uuid),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, r)
		}
		return out, nil
	}

	conn := d.MqlRuntime.Connection.(*connection.GustoConnection)
	employees, err := conn.ListEmployees(context.Background(), d.CompanyUuid.Data)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for i := range employees {
		if employees[i].DepartmentUUID != d.Uuid.Data {
			continue
		}
		r, err := newMqlGustoEmployee(d.MqlRuntime, &employees[i])
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (d *mqlGustoDepartment) contractors() ([]any, error) {
	// When the parent department list populated contractor UUIDs, resolve
	// each one lazily via the cached contractor list.
	if len(d.contractorUUIDs) > 0 {
		out := make([]any, 0, len(d.contractorUUIDs))
		for _, uuid := range d.contractorUUIDs {
			r, err := NewResource(d.MqlRuntime, "gusto.contractor", map[string]*llx.RawData{
				"uuid": llx.StringData(uuid),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, r)
		}
		return out, nil
	}

	// Selected directly via `gusto.department(uuid: ...)` — the contractor
	// list isn't keyed by department on Gusto's side, so an empty list is
	// the right answer when we have no parent context.
	return []any{}, nil
}
