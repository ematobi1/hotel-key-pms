package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/hotel-key/pms-core/internal/model"
	"github.com/hotel-key/pms-core/internal/repository"
	"github.com/hotel-key/pms-core/internal/service"
)

type HotelHandler struct {
	svc service.HotelService
	log *zap.Logger
}

func NewHotelHandler(svc service.HotelService, log *zap.Logger) *HotelHandler {
	return &HotelHandler{svc: svc, log: log}
}

// tenantID extracts the tenant from the JWT claim injected by the Istio/Keycloak auth layer.
func tenantID(c *gin.Context) string {
	if id := c.GetHeader("X-Tenant-ID"); id != "" {
		return id
	}
	// TODO: extract from validated JWT sub claim
	return c.GetHeader("X-Forwarded-User")
}

func (h *HotelHandler) List(c *gin.Context) {
	tid := tenantID(c)
	if tid == "" {
		c.JSON(http.StatusUnauthorized, errorResponse("UNAUTHORIZED", "missing tenant identity"))
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	resp, err := h.svc.List(c.Request.Context(), tid, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "failed to list hotels"))
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HotelHandler) Get(c *gin.Context) {
	tid := tenantID(c)
	if tid == "" {
		c.JSON(http.StatusUnauthorized, errorResponse("UNAUTHORIZED", "missing tenant identity"))
		return
	}

	hotel, err := h.svc.Get(c.Request.Context(), tid, c.Param("id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "hotel not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "failed to get hotel"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": hotel})
}

func (h *HotelHandler) Create(c *gin.Context) {
	tid := tenantID(c)
	if tid == "" {
		c.JSON(http.StatusUnauthorized, errorResponse("UNAUTHORIZED", "missing tenant identity"))
		return
	}

	var req model.CreateHotelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse("VALIDATION_FAILED", err.Error()))
		return
	}

	hotel, err := h.svc.Create(c.Request.Context(), tid, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "failed to create hotel"))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": hotel})
}

func (h *HotelHandler) Update(c *gin.Context) {
	tid := tenantID(c)
	if tid == "" {
		c.JSON(http.StatusUnauthorized, errorResponse("UNAUTHORIZED", "missing tenant identity"))
		return
	}

	var req model.UpdateHotelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, errorResponse("VALIDATION_FAILED", err.Error()))
		return
	}

	hotel, err := h.svc.Update(c.Request.Context(), tid, c.Param("id"), req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "hotel not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "failed to update hotel"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": hotel})
}

func (h *HotelHandler) Delete(c *gin.Context) {
	tid := tenantID(c)
	if tid == "" {
		c.JSON(http.StatusUnauthorized, errorResponse("UNAUTHORIZED", "missing tenant identity"))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), tid, c.Param("id")); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "hotel not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", "failed to delete hotel"))
		return
	}
	c.Status(http.StatusNoContent)
}

func errorResponse(code, message string) gin.H {
	return gin.H{"error": gin.H{"code": code, "message": message}}
}
