// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package mqlx provides a high-level API for embedding MQL in Go programs.
//
// It covers two modes of use:
//
// Expression mode evaluates MQL over values you pass in. It needs no
// providers, no asset, and no subprocesses — a bare environment is fully
// usable:
//
//	env, err := mqlx.NewEnv()
//	q, err := env.Compile("props.name == /admin-.*/ && props.count > 3",
//	    mqlx.WithProps(map[string]any{"name": "", "count": 0}))
//	res, err := q.Eval(ctx,
//	    mqlx.WithPropValues(map[string]any{"name": "admin-x", "count": 5}))
//	fmt.Println(res.Value()) // true
//
// Asset mode runs queries against connected infrastructure (a host, a cloud
// account, a cluster). Connect once, compile once, evaluate against as many
// assets as needed:
//
//	conn, err := env.ConnectLocal(ctx)
//	res, err := conn.Query(ctx, "asset { name platform }")
//
//	var info struct {
//	    Name     string `mql:"name"`
//	    Platform string `mql:"platform"`
//	}
//	err = res.Decode(&info)
//
// Compiled queries (Query) and environments (Env) are safe for concurrent
// use; compile once and evaluate from many goroutines.
//
// Schema availability: a query can only reference resources whose provider
// schema is loaded. Expression-mode queries (operators, string/array/map
// methods, and core resources such as regex and time) always compile.
// Resources of other providers require a prior Connect or
// NewEnv(WithProviders("aws", ...)).
package mqlx

import (
	"sync"

	"github.com/cockroachdb/errors"
	mql "go.mondoo.com/mql/v13"
	"go.mondoo.com/mql/v13/providers"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
)

// Env is the environment for compiling and evaluating MQL. It carries the
// feature set and the provider schemas, and lazily maintains an internal
// runtime for expression-mode evaluation. Create one Env per process and
// share it; it is safe for concurrent use.
type Env struct {
	features mql.Features

	mu     sync.Mutex
	conns  []*Conn
	closed bool

	exprOnce sync.Once
	exprRT   *providers.Runtime
	exprErr  error
}

// EnvOption configures an Env during NewEnv.
type EnvOption func(*Env) error

// WithFeatures sets the MQL feature flags. Default: mql.DefaultFeatures.
func WithFeatures(features mql.Features) EnvOption {
	return func(e *Env) error {
		e.features = features
		return nil
	}
}

// WithProviders eagerly loads the schemas of the named providers (e.g. "aws",
// "os") so that their resources compile before any connection is made. The
// providers must be installed; they are not started by this option.
func WithProviders(names ...string) EnvOption {
	return func(e *Env) error {
		for _, name := range names {
			schema, err := providers.Coordinator.LoadSchema(name)
			if err != nil {
				return errors.Wrap(err, "failed to load schema for provider "+name)
			}
			if ext, ok := providers.Coordinator.Schema().(providers.ExtensibleSchema); ok {
				ext.Add(name, schema)
			}
		}
		return nil
	}
}

// NewEnv creates a new environment. With no options it is immediately usable
// for expression-mode queries.
func NewEnv(opts ...EnvOption) (*Env, error) {
	e := &Env{
		features: mql.DefaultFeatures,
	}
	for _, opt := range opts {
		if err := opt(e); err != nil {
			return nil, err
		}
	}
	return e, nil
}

// Features returns the feature flags this environment was created with.
func (e *Env) Features() mql.Features {
	return e.features
}

// schema returns the current provider schema. It grows as provider schemas
// are loaded (via Connect or WithProviders).
func (e *Env) schema() resources.ResourcesSchema {
	return providers.Coordinator.Schema()
}

// exprRuntime returns the internal runtime used for expression-mode
// evaluation. It is backed solely by the builtin core provider, which runs
// in-process: creating it never spawns a provider subprocess and never
// touches any infrastructure.
func (e *Env) exprRuntime() (*providers.Runtime, error) {
	e.exprOnce.Do(func() {
		rt := providers.Coordinator.NewRuntime()
		rt.AutoUpdate = providers.UpdateProvidersConfig{Enabled: false}
		if err := rt.UseProvider(providers.BuiltinCoreID); err != nil {
			e.exprErr = err
			return
		}

		// The core provider requires an asset with one connection entry; it
		// carries no real connectivity.
		asset := &inventory.Asset{
			Connections: []*inventory.Config{{Type: "core"}},
		}
		conn, err := rt.Provider.Instance.Plugin.Connect(&plugin.ConnectReq{
			Asset:    asset,
			Features: e.features,
		}, nil)
		if err != nil {
			e.exprErr = err
			return
		}
		rt.Provider.Connection = conn
		e.exprRT = rt
	})
	return e.exprRT, e.exprErr
}

func (e *Env) trackConn(c *Conn) {
	e.mu.Lock()
	e.conns = append(e.conns, c)
	e.mu.Unlock()
}

// Close closes all connections opened through this Env and the internal
// expression runtime. It does not shut down the provider coordinator; other
// components in the process may still be using it. Use Shutdown at process
// exit.
func (e *Env) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	conns := e.conns
	e.conns = nil
	e.mu.Unlock()

	for _, c := range conns {
		c.Close()
	}
	if e.exprRT != nil {
		e.exprRT.Close()
	}
	return nil
}

// Shutdown closes the Env and additionally shuts down the provider
// coordinator, stopping all provider subprocesses. Call it once at process
// exit; in a process that embeds other MQL consumers, prefer Close.
func (e *Env) Shutdown() error {
	if err := e.Close(); err != nil {
		return err
	}
	providers.Coordinator.Shutdown()
	return nil
}
