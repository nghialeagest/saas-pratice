package authn

import "context"

type identityCtxKey struct{}

// Identity represents the authenticated subject across the request lifecycle.
type Identity struct {
	UserID    string
	TenantID  string
	SessionID string
}

// NewContext returns a new context containing the given Identity.
func NewContext(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityCtxKey{}, id)
}

// FromContext extracts the Identity from the context, if present.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityCtxKey{}).(Identity)
	return id, ok
}
