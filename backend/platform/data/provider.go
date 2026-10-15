package data

import (
	"context"
	"fmt"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"saasbase/platform/tenant"
)

type Opener func(dsn string) (*gorm.DB, error)

// Provider hands out a *gorm.DB for the current context, caching one pool per DSN.
type Provider struct {
	resolver ConnStringResolver
	open     Opener

	mu    sync.RWMutex
	pools map[string]*gorm.DB
}

func NewProvider(resolver ConnStringResolver, open Opener) *Provider {
	if open == nil {
		open = OpenPostgres
	}
	return &Provider{
		resolver: resolver,
		open:     open,
		pools:    make(map[string]*gorm.DB),
	}
}

func (p *Provider) DB(ctx context.Context) (*gorm.DB, error) {
	dsn, err := p.resolver.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	p.mu.RLock()
	db, ok := p.pools[dsn]
	p.mu.RUnlock()
	if ok {
		return db.WithContext(ctx), nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if db, ok := p.pools[dsn]; ok {
		return db.WithContext(ctx), nil
	}
	db, err = p.open(dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := tenant.Register(db); err != nil {
		return nil, fmt.Errorf("register tenant callbacks: %w", err)
	}
	p.pools[dsn] = db
	return db.WithContext(ctx), nil
}

// Pools returns every connection opened so far, for migrations and shutdown.
func (p *Provider) Pools() []*gorm.DB {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*gorm.DB, 0, len(p.pools))
	for _, db := range p.pools {
		out = append(out, db)
	}
	return out
}

func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var firstErr error
	for dsn, db := range p.pools {
		sqlDB, err := db.DB()
		if err == nil {
			err = sqlDB.Close()
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
		delete(p.pools, dsn)
	}
	return firstErr
}

func OpenPostgres(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Warn),
		SkipDefaultTransaction: true,
	})
}
