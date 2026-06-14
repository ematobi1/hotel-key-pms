package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hotel-key/pms-core/internal/model"
)

type HotelRepository interface {
	List(ctx context.Context, tenantID string, limit, offset int) ([]model.Hotel, int, error)
	Get(ctx context.Context, tenantID, id string) (*model.Hotel, error)
	Create(ctx context.Context, tenantID string, req model.CreateHotelRequest) (*model.Hotel, error)
	Update(ctx context.Context, tenantID, id string, req model.UpdateHotelRequest) (*model.Hotel, error)
	Delete(ctx context.Context, tenantID, id string) error
}

type hotelRepository struct {
	pool *pgxpool.Pool
}

func NewHotelRepository(pool *pgxpool.Pool) HotelRepository {
	return &hotelRepository{pool: pool}
}

// setTenantContext applies RLS by setting app.current_tenant_id on the connection.
// Every query must call this first so PostgreSQL row-level security policies apply.
func setTenantContext(ctx context.Context, tx pgx.Tx, tenantID string) error {
	_, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)
	return err
}

func (r *hotelRepository) List(ctx context.Context, tenantID string, limit, offset int) ([]model.Hotel, int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return nil, 0, fmt.Errorf("set tenant context: %w", err)
	}

	var total int
	if err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM hotels WHERE tenant_id = $1`,
		tenantID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count hotels: %w", err)
	}

	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, name, code, country, city, address, phone, email,
		        star_rating, active, created_at, updated_at
		 FROM hotels
		 WHERE tenant_id = $1
		 ORDER BY name
		 LIMIT $2 OFFSET $3`,
		tenantID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("query hotels: %w", err)
	}
	defer rows.Close()

	var hotels []model.Hotel
	for rows.Next() {
		var h model.Hotel
		if err := rows.Scan(
			&h.ID, &h.TenantID, &h.Name, &h.Code, &h.Country, &h.City,
			&h.Address, &h.Phone, &h.Email, &h.StarRating, &h.Active,
			&h.CreatedAt, &h.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan hotel: %w", err)
		}
		hotels = append(hotels, h)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("commit: %w", err)
	}
	return hotels, total, nil
}

func (r *hotelRepository) Get(ctx context.Context, tenantID, id string) (*model.Hotel, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	var h model.Hotel
	err = tx.QueryRow(ctx,
		`SELECT id, tenant_id, name, code, country, city, address, phone, email,
		        star_rating, active, created_at, updated_at
		 FROM hotels
		 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(
		&h.ID, &h.TenantID, &h.Name, &h.Code, &h.Country, &h.City,
		&h.Address, &h.Phone, &h.Email, &h.StarRating, &h.Active,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query hotel: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &h, nil
}

func (r *hotelRepository) Create(ctx context.Context, tenantID string, req model.CreateHotelRequest) (*model.Hotel, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	var h model.Hotel
	err = tx.QueryRow(ctx,
		`INSERT INTO hotels (tenant_id, name, code, country, city, address, phone, email, star_rating)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, tenant_id, name, code, country, city, address, phone, email,
		           star_rating, active, created_at, updated_at`,
		tenantID, req.Name, req.Code, req.Country, req.City,
		req.Address, req.Phone, req.Email, req.StarRating,
	).Scan(
		&h.ID, &h.TenantID, &h.Name, &h.Code, &h.Country, &h.City,
		&h.Address, &h.Phone, &h.Email, &h.StarRating, &h.Active,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert hotel: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &h, nil
}

func (r *hotelRepository) Update(ctx context.Context, tenantID, id string, req model.UpdateHotelRequest) (*model.Hotel, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	var h model.Hotel
	err = tx.QueryRow(ctx,
		`UPDATE hotels
		 SET name        = COALESCE($3, name),
		     city        = COALESCE($4, city),
		     address     = COALESCE($5, address),
		     phone       = COALESCE($6, phone),
		     email       = COALESCE($7, email),
		     star_rating = COALESCE($8, star_rating),
		     active      = COALESCE($9, active),
		     updated_at  = now()
		 WHERE id = $1 AND tenant_id = $2
		 RETURNING id, tenant_id, name, code, country, city, address, phone, email,
		           star_rating, active, created_at, updated_at`,
		id, tenantID, req.Name, req.City, req.Address, req.Phone,
		req.Email, req.StarRating, req.Active,
	).Scan(
		&h.ID, &h.TenantID, &h.Name, &h.Code, &h.Country, &h.City,
		&h.Address, &h.Phone, &h.Email, &h.StarRating, &h.Active,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update hotel: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &h, nil
}

func (r *hotelRepository) Delete(ctx context.Context, tenantID, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}

	tag, err := tx.Exec(ctx,
		`DELETE FROM hotels WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("delete hotel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return tx.Commit(ctx)
}

var ErrNotFound = fmt.Errorf("not found")
