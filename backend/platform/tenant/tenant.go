package tenant

import (
	"context"
	"reflect"

	"gorm.io/gorm"
)

const Header = "X-Tenant-ID"

type ctxKey struct{}
type skipKey struct{}

// NewContext enters tenantID. It also clears any inherited SkipFilter: being
// scoped to one tenant and reading across all tenants are mutually exclusive,
// and an inherited skip would otherwise silently widen every query made from
// the derived context. Call SkipFilter after NewContext when both are wanted.
func NewContext(ctx context.Context, tenantID string) context.Context {
	ctx = context.WithValue(ctx, ctxKey{}, tenantID)
	return context.WithValue(ctx, skipKey{}, false)
}

// FromContext returns the current tenant. An empty string means host scope.
func FromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKey{}).(string)
	return v
}

// SkipFilter disables tenant filtering for host-level queries.
func SkipFilter(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipKey{}, true)
}

func filterSkipped(ctx context.Context) bool {
	v, _ := ctx.Value(skipKey{}).(bool)
	return v
}

// Model is embedded by every entity that belongs to a tenant.
type Model struct {
	TenantID string `gorm:"index;size:64;not null"`
}

type scoped interface{ tenantField() *string }

func (m *Model) tenantField() *string { return &m.TenantID }

func Register(db *gorm.DB) error {
	if err := db.Callback().Create().Before("gorm:create").
		Register("tenant:assign", assign); err != nil {
		return err
	}
	_ = db.Callback().Create().Before("gorm:create").Register("tenant:rls_create", rlsCallback)

	if err := db.Callback().Query().Before("gorm:query").
		Register("tenant:filter", filter); err != nil {
		return err
	}
	_ = db.Callback().Query().Before("gorm:query").Register("tenant:rls_query", rlsCallback)

	if err := db.Callback().Update().Before("gorm:update").
		Register("tenant:filter_update", filter); err != nil {
		return err
	}
	_ = db.Callback().Update().Before("gorm:update").Register("tenant:rls_update", rlsCallback)

	_ = db.Callback().Delete().Before("gorm:delete").Register("tenant:rls_delete", rlsCallback)

	return db.Callback().Delete().Before("gorm:delete").
		Register("tenant:filter_delete", filter)
}

func rlsCallback(db *gorm.DB) {
	if db.Statement.Context == nil {
		return
	}
	_ = SetDBTenantConfig(db, db.Statement.Context)
}

// assign stamps the current tenant onto new rows. SkipFilter deliberately has no
// effect here: it suppresses reads only. Letting it suppress writes as well is
// how rows end up with an empty tenant and silently escape every later filter.
func assign(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}
	field := db.Statement.Schema.LookUpField("TenantID")
	if field == nil {
		return
	}
	id := FromContext(db.Statement.Context)
	if id == "" {
		return
	}

	setIfEmpty := func(rv reflect.Value) {
		val, isZero := field.ValueOf(db.Statement.Context, rv)
		if isZero || val == "" {
			_ = field.Set(db.Statement.Context, rv, id)
		}
	}

	rv := db.Statement.ReflectValue
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			setIfEmpty(rv.Index(i))
		}
	case reflect.Struct:
		setIfEmpty(rv)
	}
}

func filter(db *gorm.DB) {
	if db.Statement.Schema == nil || filterSkipped(db.Statement.Context) {
		return
	}
	if db.Statement.Schema.LookUpField("TenantID") == nil {
		return
	}
	db.Where(db.Statement.Table+".tenant_id = ?", FromContext(db.Statement.Context))
}
