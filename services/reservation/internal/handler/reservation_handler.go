package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/hotel-key/reservation/internal/repository"
	"github.com/hotel-key/reservation/internal/service"
)

// uuidPattern accepts any canonical 8-4-4-4-12 hex UUID. Version bits are not
// checked because seeded IDs such as 00000000-0000-0000-0001-000000000001 are
// not RFC 4122 versioned.
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isUUID(s string) bool { return uuidPattern.MatchString(s) }

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

func (h *ReservationHandler) internalError(c *gin.Context, msg string, err error) {
	h.log.Error(msg,
		zap.Error(err),
		zap.String("method", c.Request.Method),
		zap.String("path", c.FullPath()),
		zap.String("tenant_id", tenantID(c)),
	)
	c.JSON(http.StatusInternalServerError, errResp("INTERNAL_ERROR", msg))
}

func (h *ReservationHandler) List(c *gin.Context) {
	tid := tenantID(c)
	if tid == "" {
		c.JSON(http.StatusUnauthorized, errResp("UNAUTHORIZED", "missing tenant"))
		return
	}
	hotelID := c.Query("hotel_id")
	if !isUUID(hotelID) {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "hotel_id is required and must be a UUID"))
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	data, total, err := h.svc.List(c.Request.Context(), tid, hotelID, limit, offset)
	if err != nil {
		h.internalError(c, "list failed", err)
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
		h.internalError(c, "get failed", err)
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
		h.internalError(c, "create failed", err)
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
		h.internalError(c, "update failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rv})
}

func (h *ReservationHandler) CheckIn(c *gin.Context) {
	tid := tenantID(c)
	if tid == "" {
		c.JSON(http.StatusUnauthorized, errResp("UNAUTHORIZED", "missing tenant"))
		return
	}
	id := c.Param("id")
	if !isUUID(id) {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "reservation id must be a UUID"))
		return
	}
	var body struct {
		RoomID string `json:"room_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || !isUUID(body.RoomID) {
		c.JSON(http.StatusBadRequest, errResp("VALIDATION_ERROR", "room_id is required and must be a UUID"))
		return
	}

	rv, err := h.svc.CheckIn(c.Request.Context(), tid, id, body.RoomID)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"data": rv})
	case errors.Is(err, repository.ErrNotFound):
		c.JSON(http.StatusNotFound, errResp("NOT_FOUND", "reservation not found"))
	case errors.Is(err, repository.ErrRoomNotFound):
		c.JSON(http.StatusNotFound, errResp("ROOM_NOT_FOUND", "room not found in this hotel"))
	case errors.Is(err, repository.ErrRoomTypeMismatch):
		c.JSON(http.StatusBadRequest, errResp("ROOM_TYPE_MISMATCH", "room does not match the reservation's room type"))
	case errors.Is(err, repository.ErrRoomOccupied):
		c.JSON(http.StatusConflict, errResp("ROOM_OCCUPIED", "room is occupied by another checked-in reservation for overlapping dates"))
	case errors.Is(err, service.ErrCheckInInProgress):
		c.JSON(http.StatusConflict, errResp("CHECK_IN_IN_PROGRESS", "another check-in for this room is in progress"))
	default:
		h.internalError(c, "check-in failed", err)
	}
}

func (h *ReservationHandler) CheckOut(c *gin.Context) {
	tid := tenantID(c)
	rv, err := h.svc.CheckOut(c.Request.Context(), tid, c.Param("id"))
	if err != nil {
		h.internalError(c, "check-out failed", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rv})
}

func (h *ReservationHandler) Cancel(c *gin.Context) {
	tid := tenantID(c)
	if err := h.svc.Cancel(c.Request.Context(), tid, c.Param("id")); err != nil {
		h.internalError(c, "cancel failed", err)
		return
	}
	c.Status(http.StatusNoContent)
}
