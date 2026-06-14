package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/hotel-key/pms-core/internal/model"
	"github.com/hotel-key/pms-core/internal/repository"
)

type HotelService interface {
	List(ctx context.Context, tenantID string, limit, offset int) (*model.ListHotelsResponse, error)
	Get(ctx context.Context, tenantID, id string) (*model.Hotel, error)
	Create(ctx context.Context, tenantID string, req model.CreateHotelRequest) (*model.Hotel, error)
	Update(ctx context.Context, tenantID, id string, req model.UpdateHotelRequest) (*model.Hotel, error)
	Delete(ctx context.Context, tenantID, id string) error
}

type hotelService struct {
	repo repository.HotelRepository
	log  *zap.Logger
}

func NewHotelService(repo repository.HotelRepository, log *zap.Logger) HotelService {
	return &hotelService{repo: repo, log: log}
}

func (s *hotelService) List(ctx context.Context, tenantID string, limit, offset int) (*model.ListHotelsResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	hotels, total, err := s.repo.List(ctx, tenantID, limit, offset)
	if err != nil {
		s.log.Error("list hotels failed", zap.String("tenant_id", tenantID), zap.Error(err))
		return nil, fmt.Errorf("list hotels: %w", err)
	}

	return &model.ListHotelsResponse{
		Data: hotels,
		Meta: model.Meta{Total: total, Limit: limit, Offset: offset},
	}, nil
}

func (s *hotelService) Get(ctx context.Context, tenantID, id string) (*model.Hotel, error) {
	hotel, err := s.repo.Get(ctx, tenantID, id)
	if err != nil {
		s.log.Error("get hotel failed", zap.String("tenant_id", tenantID), zap.String("hotel_id", id), zap.Error(err))
		return nil, err
	}
	return hotel, nil
}

func (s *hotelService) Create(ctx context.Context, tenantID string, req model.CreateHotelRequest) (*model.Hotel, error) {
	hotel, err := s.repo.Create(ctx, tenantID, req)
	if err != nil {
		s.log.Error("create hotel failed", zap.String("tenant_id", tenantID), zap.Error(err))
		return nil, fmt.Errorf("create hotel: %w", err)
	}
	s.log.Info("hotel created", zap.String("tenant_id", tenantID), zap.String("hotel_id", hotel.ID))
	return hotel, nil
}

func (s *hotelService) Update(ctx context.Context, tenantID, id string, req model.UpdateHotelRequest) (*model.Hotel, error) {
	hotel, err := s.repo.Update(ctx, tenantID, id, req)
	if err != nil {
		s.log.Error("update hotel failed", zap.String("tenant_id", tenantID), zap.String("hotel_id", id), zap.Error(err))
		return nil, err
	}
	return hotel, nil
}

func (s *hotelService) Delete(ctx context.Context, tenantID, id string) error {
	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		s.log.Error("delete hotel failed", zap.String("tenant_id", tenantID), zap.String("hotel_id", id), zap.Error(err))
		return err
	}
	s.log.Info("hotel deleted", zap.String("tenant_id", tenantID), zap.String("hotel_id", id))
	return nil
}
