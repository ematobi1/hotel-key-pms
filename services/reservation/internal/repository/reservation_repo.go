package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Reservation struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	HotelID       string    `json:"hotel_id"`
	RoomID        *string   `json:"room_id,omitempty"`
	RoomTypeID    string    `json:"room_type_id"`
	GuestID       string    `json:"guest_id"`
	Status        string    `json:"status"`
	ArrivalDate   time.Time `json:"arrival_date"`
	DepartureDate time.Time `json:"departure_date"`
	NumAdults     int       `json:"num_adults"`
	NumChildren   int       `json:"num_children"`
	RatePlan      string    `json:"rate_plan"`
	TotalAmount   *float64  `json:"total_amount,omitempty"`
	Currency      string    `json:"currency"`
	Channel       string    `json:"channel"`
	ExternalRef   *string   `json:"external_ref,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateReservationInput struct {
	TenantID      string
	HotelID       string
	RoomTypeID    string
	GuestID       string
	ArrivalDate   time.Time
	DepartureDate time.Time
	NumAdults     int
	NumChildren   int
	RatePlan      string
	Currency      string
	Channel       string
}

type ReservationRepository interface {
	List(ctx context.Context, tenantID, hotelID string, limit, offset int) ([]Reservation, int, error)
	Get(ctx context.Context, tenantID, id string) (*Reservation, error)
	Create(ctx context.Context, in CreateReservationInput) (*Reservation, error)
	UpdateStatus(ctx context.Context, tenantID, id, status string) (*Reservation, error)
	CheckIn(ctx context.Context, tenantID, id, roomID string) (*Reservation, error)
	CheckOut(ctx context.Context, tenantID, id string) (*Reservation, error)
}

var ErrNotFound = fmt.Errorf("not found")
var ErrConflict = fmt.Errorf("reservation conflict")
var ErrRoomNotFound = fmt.Errorf("room not found")
var ErrRoomTypeMismatch = fmt.Errorf("room type does not match reservation")
var ErrRoomOccupied = fmt.Errorf("room occupied")

type reservationRepository struct{ pool *pgxpool.Pool }

func NewReservationRepository(pool *pgxpool.Pool) ReservationRepository {
	return &reservationRepository{pool: pool}
}

func setTenant(ctx context.Context, tx pgx.Tx, tenantID string) error {
	_, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)
	return err
}

func (r *reservationRepository) List(ctx context.Context, tenantID, hotelID string, limit, offset int) ([]Reservation, int, error) {
	// TODO: add date-range and status filters from query params
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback(ctx)
	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, 0, err
	}

	var total int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM reservations WHERE tenant_id=$1 AND hotel_id=$2`, tenantID, hotelID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count reservations: %w", err)
	}

	rows, err := tx.Query(ctx,
		`SELECT id, tenant_id, hotel_id, room_id, room_type_id, guest_id,
		        status, arrival_date, departure_date, num_adults, num_children,
		        rate_plan, total_amount, currency, channel, external_ref, created_at, updated_at
		 FROM reservations WHERE tenant_id=$1 AND hotel_id=$2
		 ORDER BY arrival_date DESC LIMIT $3 OFFSET $4`,
		tenantID, hotelID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []Reservation
	for rows.Next() {
		var rv Reservation
		if err := rows.Scan(
			&rv.ID, &rv.TenantID, &rv.HotelID, &rv.RoomID, &rv.RoomTypeID, &rv.GuestID,
			&rv.Status, &rv.ArrivalDate, &rv.DepartureDate, &rv.NumAdults, &rv.NumChildren,
			&rv.RatePlan, &rv.TotalAmount, &rv.Currency, &rv.Channel, &rv.ExternalRef,
			&rv.CreatedAt, &rv.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, rv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *reservationRepository) Get(ctx context.Context, tenantID, id string) (*Reservation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	var rv Reservation
	err = tx.QueryRow(ctx,
		`SELECT id, tenant_id, hotel_id, room_id, room_type_id, guest_id,
		        status, arrival_date, departure_date, num_adults, num_children,
		        rate_plan, total_amount, currency, channel, external_ref, created_at, updated_at
		 FROM reservations WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
	).Scan(
		&rv.ID, &rv.TenantID, &rv.HotelID, &rv.RoomID, &rv.RoomTypeID, &rv.GuestID,
		&rv.Status, &rv.ArrivalDate, &rv.DepartureDate, &rv.NumAdults, &rv.NumChildren,
		&rv.RatePlan, &rv.TotalAmount, &rv.Currency, &rv.Channel, &rv.ExternalRef,
		&rv.CreatedAt, &rv.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	tx.Commit(ctx)
	return &rv, nil
}

func (r *reservationRepository) Create(ctx context.Context, in CreateReservationInput) (*Reservation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err := setTenant(ctx, tx, in.TenantID); err != nil {
		return nil, err
	}

	// Conflict check: any confirmed/checked_in reservation for the same room type overlapping these dates
	var conflictCount int
	tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM reservations
		 WHERE hotel_id=$1 AND room_type_id=$2
		   AND status IN ('confirmed','checked_in')
		   AND arrival_date < $4 AND departure_date > $3`,
		in.HotelID, in.RoomTypeID, in.ArrivalDate, in.DepartureDate,
	).Scan(&conflictCount)

	var rv Reservation
	err = tx.QueryRow(ctx,
		`INSERT INTO reservations
		   (tenant_id, hotel_id, room_type_id, guest_id, arrival_date, departure_date,
		    num_adults, num_children, rate_plan, currency, channel)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 RETURNING id, tenant_id, hotel_id, room_id, room_type_id, guest_id,
		           status, arrival_date, departure_date, num_adults, num_children,
		           rate_plan, total_amount, currency, channel, external_ref, created_at, updated_at`,
		in.TenantID, in.HotelID, in.RoomTypeID, in.GuestID,
		in.ArrivalDate, in.DepartureDate, in.NumAdults, in.NumChildren,
		in.RatePlan, in.Currency, in.Channel,
	).Scan(
		&rv.ID, &rv.TenantID, &rv.HotelID, &rv.RoomID, &rv.RoomTypeID, &rv.GuestID,
		&rv.Status, &rv.ArrivalDate, &rv.DepartureDate, &rv.NumAdults, &rv.NumChildren,
		&rv.RatePlan, &rv.TotalAmount, &rv.Currency, &rv.Channel, &rv.ExternalRef,
		&rv.CreatedAt, &rv.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert reservation: %w", err)
	}
	tx.Commit(ctx)
	return &rv, nil
}

func (r *reservationRepository) UpdateStatus(ctx context.Context, tenantID, id, status string) (*Reservation, error) {
	// TODO: validate allowed status transitions before updating
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	var rv Reservation
	err = tx.QueryRow(ctx,
		`UPDATE reservations SET status=$3, updated_at=now()
		 WHERE id=$1 AND tenant_id=$2
		 RETURNING id, tenant_id, hotel_id, room_id, room_type_id, guest_id,
		           status, arrival_date, departure_date, num_adults, num_children,
		           rate_plan, total_amount, currency, channel, external_ref, created_at, updated_at`,
		id, tenantID, status,
	).Scan(
		&rv.ID, &rv.TenantID, &rv.HotelID, &rv.RoomID, &rv.RoomTypeID, &rv.GuestID,
		&rv.Status, &rv.ArrivalDate, &rv.DepartureDate, &rv.NumAdults, &rv.NumChildren,
		&rv.RatePlan, &rv.TotalAmount, &rv.Currency, &rv.Channel, &rv.ExternalRef,
		&rv.CreatedAt, &rv.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	tx.Commit(ctx)
	return &rv, nil
}

// CheckIn assigns roomID and marks the reservation checked_in in one transaction.
// The room row is locked FOR UPDATE so concurrent check-ins to the same room
// serialize on the occupancy check, whatever their dates.
func (r *reservationRepository) CheckIn(ctx context.Context, tenantID, id, roomID string) (*Reservation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	var hotelID, roomTypeID string
	var arrival, departure time.Time
	err = tx.QueryRow(ctx,
		`SELECT hotel_id, room_type_id, arrival_date, departure_date
		 FROM reservations WHERE id=$1 AND tenant_id=$2
		 FOR UPDATE`,
		id, tenantID,
	).Scan(&hotelID, &roomTypeID, &arrival, &departure)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load reservation: %w", err)
	}

	var roomRoomTypeID string
	err = tx.QueryRow(ctx,
		`SELECT room_type_id FROM rooms
		 WHERE id=$1 AND tenant_id=$2 AND hotel_id=$3
		 FOR UPDATE`,
		roomID, tenantID, hotelID,
	).Scan(&roomRoomTypeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoomNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load room: %w", err)
	}
	if roomRoomTypeID != roomTypeID {
		return nil, ErrRoomTypeMismatch
	}

	var occupied bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS (
		   SELECT 1 FROM reservations
		   WHERE room_id=$1 AND tenant_id=$2 AND id<>$3
		     AND status='checked_in'
		     AND arrival_date < $5 AND departure_date > $4
		 )`,
		roomID, tenantID, id, arrival, departure,
	).Scan(&occupied)
	if err != nil {
		return nil, fmt.Errorf("check room occupancy: %w", err)
	}
	if occupied {
		return nil, ErrRoomOccupied
	}

	var rv Reservation
	err = tx.QueryRow(ctx,
		`UPDATE reservations SET status='checked_in', room_id=$3, updated_at=now()
		 WHERE id=$1 AND tenant_id=$2
		 RETURNING id, tenant_id, hotel_id, room_id, room_type_id, guest_id,
		           status, arrival_date, departure_date, num_adults, num_children,
		           rate_plan, total_amount, currency, channel, external_ref, created_at, updated_at`,
		id, tenantID, roomID,
	).Scan(
		&rv.ID, &rv.TenantID, &rv.HotelID, &rv.RoomID, &rv.RoomTypeID, &rv.GuestID,
		&rv.Status, &rv.ArrivalDate, &rv.DepartureDate, &rv.NumAdults, &rv.NumChildren,
		&rv.RatePlan, &rv.TotalAmount, &rv.Currency, &rv.Channel, &rv.ExternalRef,
		&rv.CreatedAt, &rv.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update reservation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit check-in: %w", err)
	}
	return &rv, nil
}

func (r *reservationRepository) CheckOut(ctx context.Context, tenantID, id string) (*Reservation, error) {
	return r.UpdateStatus(ctx, tenantID, id, "checked_out")
}
