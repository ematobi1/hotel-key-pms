package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/hotel-key/reservation/internal/lock"
	"github.com/hotel-key/reservation/internal/repository"
)

type CreateReservationRequest struct {
	HotelID       string    `json:"hotel_id" binding:"required"`
	RoomTypeID    string    `json:"room_type_id" binding:"required"`
	GuestID       string    `json:"guest_id" binding:"required"`
	ArrivalDate   time.Time `json:"arrival_date" binding:"required"`
	DepartureDate time.Time `json:"departure_date" binding:"required"`
	NumAdults     int       `json:"num_adults" binding:"min=1"`
	NumChildren   int       `json:"num_children"`
	RatePlan      string    `json:"rate_plan"`
	Currency      string    `json:"currency"`
	Channel       string    `json:"channel"`
}

var ErrCheckInInProgress = errors.New("concurrent check-in in progress for room")

type ReservationService interface {
	List(ctx context.Context, tenantID, hotelID string, limit, offset int) ([]repository.Reservation, int, error)
	Get(ctx context.Context, tenantID, id string) (*repository.Reservation, error)
	Create(ctx context.Context, tenantID string, req CreateReservationRequest) (*repository.Reservation, error)
	UpdateStatus(ctx context.Context, tenantID, id, status string) (*repository.Reservation, error)
	CheckIn(ctx context.Context, tenantID, id, roomID string) (*repository.Reservation, error)
	CheckOut(ctx context.Context, tenantID, id string) (*repository.Reservation, error)
	Cancel(ctx context.Context, tenantID, id string) error
}

type reservationService struct {
	repo   repository.ReservationRepository
	locker lock.Locker
	log    *zap.Logger
}

func NewReservationService(repo repository.ReservationRepository, locker lock.Locker, log *zap.Logger) ReservationService {
	return &reservationService{repo: repo, locker: locker, log: log}
}

func (s *reservationService) List(ctx context.Context, tenantID, hotelID string, limit, offset int) ([]repository.Reservation, int, error) {
	return s.repo.List(ctx, tenantID, hotelID, limit, offset)
}

func (s *reservationService) Get(ctx context.Context, tenantID, id string) (*repository.Reservation, error) {
	return s.repo.Get(ctx, tenantID, id)
}

func (s *reservationService) Create(ctx context.Context, tenantID string, req CreateReservationRequest) (*repository.Reservation, error) {
	if !req.DepartureDate.After(req.ArrivalDate) {
		return nil, fmt.Errorf("departure must be after arrival")
	}

	// Acquire per-room-type mutex to prevent double-booking race conditions
	release, err := s.locker.Acquire(ctx, req.RoomTypeID, req.ArrivalDate, req.DepartureDate)
	if err != nil {
		if errors.Is(err, lock.ErrLockHeld) {
			return nil, fmt.Errorf("room type unavailable: concurrent booking in progress")
		}
		return nil, fmt.Errorf("acquire lock: %w", err)
	}
	defer release(ctx)

	in := repository.CreateReservationInput{
		TenantID:      tenantID,
		HotelID:       req.HotelID,
		RoomTypeID:    req.RoomTypeID,
		GuestID:       req.GuestID,
		ArrivalDate:   req.ArrivalDate,
		DepartureDate: req.DepartureDate,
		NumAdults:     req.NumAdults,
		NumChildren:   req.NumChildren,
		RatePlan:      req.RatePlan,
		Currency:      req.Currency,
		Channel:       req.Channel,
	}

	rv, err := s.repo.Create(ctx, in)
	if err != nil {
		s.log.Error("create reservation failed", zap.Error(err))
		return nil, err
	}
	s.log.Info("reservation created",
		zap.String("id", rv.ID),
		zap.String("tenant_id", tenantID),
		zap.String("hotel_id", req.HotelID),
	)
	return rv, nil
}

func (s *reservationService) UpdateStatus(ctx context.Context, tenantID, id, status string) (*repository.Reservation, error) {
	return s.repo.UpdateStatus(ctx, tenantID, id, status)
}

func (s *reservationService) CheckIn(ctx context.Context, tenantID, id, roomID string) (*repository.Reservation, error) {
	rv, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	release, err := s.locker.Acquire(ctx, roomID, rv.ArrivalDate, rv.DepartureDate)
	if err != nil {
		if errors.Is(err, lock.ErrLockHeld) {
			return nil, ErrCheckInInProgress
		}
		return nil, fmt.Errorf("acquire lock: %w", err)
	}
	defer release(ctx)

	return s.repo.CheckIn(ctx, tenantID, id, roomID)
}

func (s *reservationService) CheckOut(ctx context.Context, tenantID, id string) (*repository.Reservation, error) {
	return s.repo.CheckOut(ctx, tenantID, id)
}

func (s *reservationService) Cancel(ctx context.Context, tenantID, id string) error {
	_, err := s.repo.UpdateStatus(ctx, tenantID, id, "cancelled")
	return err
}
