package model

import "time"

type Hotel struct {
	ID         string    `json:"id" db:"id"`
	TenantID   string    `json:"tenant_id" db:"tenant_id"`
	Name       string    `json:"name" db:"name"`
	Code       string    `json:"code" db:"code"`
	Country    string    `json:"country" db:"country"`
	City       string    `json:"city" db:"city"`
	Address    string    `json:"address" db:"address"`
	Phone      string    `json:"phone" db:"phone"`
	Email      string    `json:"email" db:"email"`
	StarRating int       `json:"star_rating" db:"star_rating"`
	Active     bool      `json:"active" db:"active"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type CreateHotelRequest struct {
	Name       string `json:"name" binding:"required,min=2,max=255"`
	Code       string `json:"code" binding:"required,min=2,max=20"`
	Country    string `json:"country" binding:"required,len=2"`
	City       string `json:"city" binding:"required"`
	Address    string `json:"address" binding:"required"`
	Phone      string `json:"phone" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	StarRating int    `json:"star_rating" binding:"min=1,max=5"`
}

type UpdateHotelRequest struct {
	Name       *string `json:"name" binding:"omitempty,min=2,max=255"`
	City       *string `json:"city"`
	Address    *string `json:"address"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email" binding:"omitempty,email"`
	StarRating *int    `json:"star_rating" binding:"omitempty,min=1,max=5"`
	Active     *bool   `json:"active"`
}

type ListHotelsResponse struct {
	Data []Hotel `json:"data"`
	Meta Meta    `json:"meta"`
}

type Meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
