package data

import (
	"context"
	"fmt"

	"saasbase/platform/tenant"
)

// ConnStringResolver maps the current request context to a database DSN.
type ConnStringResolver interface {
	Resolve(ctx context.Context) (string, error)
}

// SharedConn routes every tenant to a single database.
type SharedConn struct {
	dsn string
}

func NewSharedConn(dsn string) *SharedConn {
	return &SharedConn{dsn: dsn}
}

func (s *SharedConn) Resolve(context.Context) (string, error) {
	return s.dsn, nil
}

// TenantConnStore supplies a dedicated DSN for tenants that have one.
type TenantConnStore interface {
	DSN(ctx context.Context, tenantID string) (dsn string, ok bool, err error)
}

// TenantConn routes tenants with a dedicated database to it and everyone else
// to the default DSN.
type TenantConn struct {
	store      TenantConnStore
	defaultDSN string
}

func NewTenantConn(store TenantConnStore, defaultDSN string) *TenantConn {
	return &TenantConn{store: store, defaultDSN: defaultDSN}
}

func (t *TenantConn) Resolve(ctx context.Context) (string, error) {
	id := tenant.FromContext(ctx)
	if id == "" {
		return t.defaultDSN, nil
	}
	dsn, ok, err := t.store.DSN(ctx, id)
	if err != nil {
		return "", fmt.Errorf("resolve dsn for tenant %q: %w", id, err)
	}
	if !ok {
		return t.defaultDSN, nil
	}
	return dsn, nil
}
