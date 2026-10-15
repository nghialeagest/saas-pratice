package module

import (
	"context"
	"fmt"

	"saasbase/platform/data"
)

type Permission struct {
	Namespace string
	Action    string
	HostOnly  bool
}

type Deps struct {
	DB *data.Provider
}

type Module interface {
	Name() string
	Tables() []any
	Permissions() []Permission
	Init(Deps) error
}

// Wirer is implemented by modules that consume another module's contract.
type Wirer interface {
	Wire() error
}

// Seeder is implemented by modules that need bootstrap data per tenant.
type Seeder interface {
	Seed(ctx context.Context, tenantID string) error
}

var (
	registered []Module
	provided   []any
)

func Register(m Module) { registered = append(registered, m) }

func All() []Module { return registered }

func Provide(v any) { provided = append(provided, v) }

func Resolve[T any]() (T, error) {
	var zero T
	for _, v := range provided {
		if t, ok := v.(T); ok {
			return t, nil
		}
	}
	return zero, fmt.Errorf("no module provides %T", zero)
}

func InitAll(deps Deps) error {
	provided = nil
	for _, m := range registered {
		if err := m.Init(deps); err != nil {
			return fmt.Errorf("init module %s: %w", m.Name(), err)
		}
	}
	for _, m := range registered {
		w, ok := m.(Wirer)
		if !ok {
			continue
		}
		if err := w.Wire(); err != nil {
			return fmt.Errorf("wire module %s: %w", m.Name(), err)
		}
	}
	return nil
}

func Tables() []any {
	var out []any
	for _, m := range registered {
		out = append(out, m.Tables()...)
	}
	return out
}

func Permissions() []Permission {
	var out []Permission
	for _, m := range registered {
		out = append(out, m.Permissions()...)
	}
	return out
}

func SeedAll(ctx context.Context, tenantID string) error {
	for _, m := range registered {
		s, ok := m.(Seeder)
		if !ok {
			continue
		}
		if err := s.Seed(ctx, tenantID); err != nil {
			return fmt.Errorf("seed module %s: %w", m.Name(), err)
		}
	}
	return nil
}
