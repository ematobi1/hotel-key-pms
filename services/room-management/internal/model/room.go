package model

import "time"

type Room struct {
	ID         string    `json:"id"`
	HotelID    string    `json:"hotel_id"`
	TenantID   string    `json:"tenant_id"`
	RoomTypeID string    `json:"room_type_id"`
	Number     string    `json:"number"`
	Floor      *int      `json:"floor,omitempty"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type RoomType struct {
	ID                    string    `json:"id"`
	HotelID               string    `json:"hotel_id"`
	TenantID              string    `json:"tenant_id"`
	Name                  string    `json:"name"`
	Code                  string    `json:"code"`
	MaxOccupancy          int       `json:"max_occupancy"`
	BaseRate              float64   `json:"base_rate"`
	Currency              string    `json:"currency"`
	PerPersonPricingEnabled bool    `json:"per_person_pricing_enabled"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// UpdateRoomStatusRequest carries a validated status change.
type UpdateRoomStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=clean dirty inspected out_of_order out_of_service"`
}

// TODO: Add per-person pricing engine for Japan/China markets.
// See services/room-management/pricing/per_person_engine.go
// Per-person pricing calculates occupancy-based rates:
//   rate = base_rate + (num_persons - 1) * extra_person_charge
// Required for Ryokan (旅館) style properties in Japan where rates are quoted per person.
