package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type SeedService struct {
	db *sql.DB
}

func NewSeedService(db *sql.DB) *SeedService {
	return &SeedService{db: db}
}

func (s *SeedService) Run(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	tenantID, err := s.seedTenant(ctx, tx)
	if err != nil {
		return err
	}

	if err = s.seedAdminUser(ctx, tx, tenantID); err != nil {
		return err
	}

	categoryIDs, err := s.seedCategories(ctx, tx, tenantID)
	if err != nil {
		return err
	}

	if err = s.seedProducts(ctx, tx, tenantID, categoryIDs); err != nil {
		return err
	}

	if err = s.seedWarehouses(ctx, tx, tenantID); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (s *SeedService) Rollback(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(
		ctx,
		`DELETE FROM tenants WHERE slug = $1 OR name = $2`,
		demoTenantSlug, demoTenantName,
	)
	if err != nil {
		return fmt.Errorf("delete demo tenant: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (s *SeedService) seedTenant(ctx context.Context, tx *sql.Tx) (string, error) {
	var tenantID string

	err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM tenants WHERE slug = $1 OR name = $2 LIMIT 1`,
		demoTenantSlug, demoTenantName,
	).Scan(&tenantID)
	if err == nil {
		return tenantID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("select tenant: %w", err)
	}

	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO tenants (name, slug, is_active, settings) VALUES ($1, $2, true, '{}'::jsonb) RETURNING id`,
		demoTenantName, demoTenantSlug,
	).Scan(&tenantID)
	if err != nil {
		return "", fmt.Errorf("insert tenant: %w", err)
	}

	return tenantID, nil
}

func (s *SeedService) seedAdminUser(ctx context.Context, tx *sql.Tx, tenantID string) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO users (tenant_id, email, password_hash, role, full_name)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (tenant_id, email)
		 DO UPDATE SET
		   password_hash = EXCLUDED.password_hash,
		   role = EXCLUDED.role,
		   full_name = EXCLUDED.full_name,
		   updated_at = NOW()`,
		tenantID, adminEmail, string(passwordHash), adminRole, adminFullName,
	)
	if err != nil {
		return fmt.Errorf("upsert admin user: %w", err)
	}

	return nil
}

func (s *SeedService) seedCategories(ctx context.Context, tx *sql.Tx, tenantID string) (map[string]string, error) {
	categoryIDs := make(map[string]string, len(demoCategories))

	for _, c := range demoCategories {
		var categoryID string
		err := tx.QueryRowContext(
			ctx,
			`INSERT INTO categories (tenant_id, name, description, sort_order)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (tenant_id, name)
			 DO UPDATE SET
			   description = EXCLUDED.description,
			   sort_order = EXCLUDED.sort_order,
			   updated_at = NOW()
			 RETURNING id`,
			tenantID, c.Name, c.Description, c.SortOrder,
		).Scan(&categoryID)
		if err != nil {
			return nil, fmt.Errorf("upsert category %s: %w", c.Name, err)
		}
		categoryIDs[c.Name] = categoryID
	}

	return categoryIDs, nil
}

func (s *SeedService) seedProducts(ctx context.Context, tx *sql.Tx, tenantID string, categoryIDs map[string]string) error {
	for _, p := range demoProducts {
		categoryID, ok := categoryIDs[p.CategoryName]
		if !ok {
			return fmt.Errorf("category not found for product %s: %s", p.SKU, p.CategoryName)
		}

		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO products (
			    tenant_id, category_id, sku, name, description, unit, price, reorder_level, metadata
			  ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)
			  ON CONFLICT (tenant_id, sku)
			  DO UPDATE SET
			    category_id = EXCLUDED.category_id,
			    name = EXCLUDED.name,
			    description = EXCLUDED.description,
			    unit = EXCLUDED.unit,
			    price = EXCLUDED.price,
			    reorder_level = EXCLUDED.reorder_level,
			    metadata = EXCLUDED.metadata,
			    updated_at = NOW()`,
			tenantID,
			categoryID,
			p.SKU,
			p.Name,
			p.Description,
			p.Unit,
			p.Price,
			p.ReorderLevel,
			defaultReqState,
		)
		if err != nil {
			return fmt.Errorf("upsert product %s: %w", p.SKU, err)
		}
	}

	return nil
}

func (s *SeedService) seedWarehouses(ctx context.Context, tx *sql.Tx, tenantID string) error {
	for _, w := range demoWarehouses {
		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO warehouses (tenant_id, code, name, address, is_active)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (tenant_id, code)
			 DO UPDATE SET
			   name = EXCLUDED.name,
			   address = EXCLUDED.address,
			   is_active = EXCLUDED.is_active,
			   updated_at = NOW()`,
			tenantID, w.Code, w.Name, w.Address, w.IsActive,
		)
		if err != nil {
			return fmt.Errorf("upsert warehouse %s: %w", w.Code, err)
		}
	}

	return nil
}
