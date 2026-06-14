package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/hotel-key/reservation/internal/repository"
	"github.com/hotel-key/reservation/internal/service"
)

type ReservationHandler struct {
	svc service.ReservationService
	log *zap.Logger
}

func NewReservationHandler(svc service.ReservationService, log *zap.Logger) *ReservationHandler {
	return &ReservationHandler{svc: svc, log: log}
}

func tenantID(c *gin.Context) string {
	return c.GetHeader("X-Tenant-ID")
}

func errResp(code, msg string) gin.H {
	return gin.H{"error": gin.H{"code": code, "message": msg}}
}

func (h *ReservationHandler) List(c *gin.Context) {
	tid := tenantID(c)
	if tid == "" {
		c.JSON(http.StatusUnauthorized, errResp("UNAUTHORIZED", "missing tenant"))
		return
	}
	hotelID := c.Query("hotel_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	data, total, err := h.svc.List(c.Request.Context(), tid, hotelID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", "list failed"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{"total": total, "limit": limit, "offset": offset},
	})
}

func (h *ReservationHandler) Get(c *gin.Context) {
	tid := tenantID(c)
	if tid == "" {
		c.JSON(http.StatusUnauthorized, errResp("UNAUTHORIZED", "missing tenant"))
		return
	}
	rv, err := h.svc.Get(c.Request.Context(), tid, c.Param("id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "reservation not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", "get failed"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rv})
}

func (h *ReservationHandler) Create(c *gin.Context) {
	tid := tenantID(c)
	if tid == "" {
		c.JSON(http.StatusUnauthorized, errResp("UNAUTHORIZED", "missing tenant"))
		return
	}
	var req service.CreateReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_FAILED", err.Error()))
		return
	}
	rv, err := h.svc.Create(c.Request.Context(), tid, req)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			c.JSON(http.StatusConflict, errResp("DOUBLE_BOOKING", "room type unavailable for requested dates"))
			return
		}
		c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": rv})
}

func (h *ReservationHandler) UpdateStatus(c *gin.Context) {
	tid := tenantID(c)
	if tid == "" {
		c.JSON(http.StatusUnauthorized, errResp("UNAUTHORIZED", "missing tenant"))
		return
	}
	var body struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errResp("VALIDATION_FAILED", err.Error()))
		return
	}
	rv, err := h.svc.UpdateStatus(c.Request.Context(), tid, c.Param("id"), body.Status)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "reservation not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", "update failed"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rv})
}

func (h *ReservationHandler) CheckIn(c *gin.Context) {
	tid := tenantID(c)
	var body struct{ RoomID string `json:"room_id"` }
	c.ShouldBindJSON(&body)
	rv, err := h.svc.CheckIn(c.Request.Context(), tid, c.Param("id"), body.RoomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rv})
}

func (h *ReservationHandler) CheckOut(c *gin.Context) {
	tid := tenantID(c)
	rv, err := h.svc.CheckOut(c.Request.Context(), tid, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rv})
}

func (h *ReservationHandler) Cancel(c *gin.Context) {
	tid := tenantID(c)
	if err := h.svc.Cancel(c.Request.Context(), tid, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}
