package appendonly

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EntryType string

const (
	EntryCharge   EntryType = "charge"
	EntryPayment  EntryType = "payment"
	EntryTax      EntryType = "tax"
	EntryDiscount EntryType = "discount"
	EntryRefund   EntryType = "refund"
	EntryVoid     EntryType = "void"
)

type FolioEntry struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	ReservationID string    `json:"reservation_id"`
	EntryType     EntryType `json:"entry_type"`
	Description   string    `json:"description"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	PostedAt      time.Time `json:"posted_at"`
	PostedBy      string    `json:"posted_by"`
}

type PostEntryInput struct {
	TenantID      string
	ReservationID string
	EntryType     EntryType
	Description   string
	Amount        float64
	Currency      string
	PostedBy      string
}

// FolioLedger is the append-only folio writer. Immutability is enforced at the
// database level via triggers (see migrations/001_initial_schema.sql). This layer
// must never attempt UPDATE or DELETE on folio_entries.
type FolioLedger struct {
	pool *pgxpool.Pool
}

func NewFolioLedger(pool *pgxpool.Pool) *FolioLedger {
	return &FolioLedger{pool: pool}
}

// Post appends a single entry to the folio. The DB trigger will reject any
// mutation of existing rows — this is the only valid write path.
func (f *FolioLedger) Post(ctx context.Context, in PostEntryInput) (*FolioEntry, error) {
	if in.Amount == 0 {
		return nil, fmt.Errorf("folio entry amount must be non-zero")
	}

	var entry FolioEntry
	err := f.pool.QueryRow(ctx,
		`INSERT INTO folio_entries
		   (tenant_id, reservation_id, entry_type, description, amount, currency, posted_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, tenant_id, reservation_id, entry_type, description, amount, currency, posted_at, posted_by`,
		in.TenantID, in.ReservationID, string(in.EntryType),
		in.Description, in.Amount, in.Currency, in.PostedBy,
	).Scan(
		&entry.ID, &entry.TenantID, &entry.ReservationID,
		&entry.EntryType, &entry.Description, &entry.Amount,
		&entry.Currency, &entry.PostedAt, &entry.PostedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("post folio entry: %w", err)
	}
	return &entry, nil
}

// ListByReservation returns all entries for a reservation in chronological order.
func (f *FolioLedger) ListByReservation(ctx context.Context, tenantID, reservationID string) ([]FolioEntry, error) {
	rows, err := f.pool.Query(ctx,
		`SELECT id, tenant_id, reservation_id, entry_type, description, amount, currency, posted_at, posted_by
		 FROM folio_entries
		 WHERE reservation_id = $1 AND tenant_id = $2
		 ORDER BY posted_at`,
		reservationID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("query folio entries: %w", err)
	}
	defer rows.Close()

	var entries []FolioEntry
	for rows.Next() {
		var e FolioEntry
		if err := rows.Scan(&e.ID, &e.TenantID, &e.ReservationID, &e.EntryType,
			&e.Description, &e.Amount, &e.Currency, &e.PostedAt, &e.PostedBy); err != nil {
			return nil, fmt.Errorf("scan folio entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Balance returns the current balance (sum of all entries) for a reservation.
func (f *FolioLedger) Balance(ctx context.Context, tenantID, reservationID string) (float64, string, error) {
	var balance float64
	var currency string
	err := f.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0), COALESCE(MAX(currency), 'JPY')
		 FROM folio_entries
		 WHERE reservation_id = $1 AND tenant_id = $2`,
		reservationID, tenantID,
	).Scan(&balance, &currency)
	if err != nil {
		return 0, "", fmt.Errorf("balance query: %w", err)
	}
	return balance, currency, nil
}

// Void posts a compensating void entry for the given original entry ID.
// This is the correct pattern for correcting errors — never update/delete rows.
func (f *FolioLedger) Void(ctx context.Context, tenantID, originalEntryID, postedBy, reason string) (*FolioEntry, error) {
	var orig FolioEntry
	err := f.pool.QueryRow(ctx,
		`SELECT id, reservation_id, amount, currency FROM folio_entries
		 WHERE id = $1 AND tenant_id = $2`,
		originalEntryID, tenantID,
	).Scan(&orig.ID, &orig.ReservationID, &orig.Amount, &orig.Currency)
	if err != nil {
		return nil, fmt.Errorf("find original entry: %w", err)
	}

	return f.Post(ctx, PostEntryInput{
		TenantID:      tenantID,
		ReservationID: orig.ReservationID,
		EntryType:     EntryVoid,
		Description:   fmt.Sprintf("VOID of %s: %s", originalEntryID, reason),
		Amount:        -orig.Amount,
		Currency:      orig.Currency,
		PostedBy:      postedBy,
	})
}
