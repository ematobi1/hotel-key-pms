package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/hotel-key/reservation/internal/repository"
	"github.com/hotel-key/reservation/internal/service"
)

const (
	testTenant      = "00000000-0000-0000-0000-000000000001"
	testHotel       = "00000000-0000-0000-0001-000000000001"
	testReservation = "11111111-1111-1111-1111-111111111111"
	testRoom        = "22222222-2222-2222-2222-222222222222"
)

var errDB = errors.New("pq: connection reset by peer")

type fakeService struct {
	listErr    error
	checkInErr error

	listCalls    int
	checkInCalls int
	gotHotelID   string
	gotRoomID    string
}

func (f *fakeService) List(_ context.Context, _, hotelID string, _, _ int) ([]repository.Reservation, int, error) {
	f.listCalls++
	f.gotHotelID = hotelID
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return []repository.Reservation{{ID: testReservation, HotelID: hotelID}}, 1, nil
}

func (f *fakeService) CheckIn(_ context.Context, _, id, roomID string) (*repository.Reservation, error) {
	f.checkInCalls++
	f.gotRoomID = roomID
	if f.checkInErr != nil {
		return nil, f.checkInErr
	}
	return &repository.Reservation{ID: id, RoomID: &roomID, Status: "checked_in"}, nil
}

func (f *fakeService) Get(context.Context, string, string) (*repository.Reservation, error) {
	return nil, errors.New("not used")
}

func (f *fakeService) Create(context.Context, string, service.CreateReservationRequest) (*repository.Reservation, error) {
	return nil, errors.New("not used")
}

func (f *fakeService) UpdateStatus(context.Context, string, string, string) (*repository.Reservation, error) {
	return nil, errors.New("not used")
}

func (f *fakeService) CheckOut(context.Context, string, string) (*repository.Reservation, error) {
	return nil, errors.New("not used")
}

func (f *fakeService) Cancel(context.Context, string, string) error {
	return errors.New("not used")
}

func newRouter(svc service.ReservationService) (*gin.Engine, *observer.ObservedLogs) {
	gin.SetMode(gin.TestMode)
	core, logs := observer.New(zapcore.ErrorLevel)
	h := NewReservationHandler(svc, zap.New(core))
	r := gin.New()
	r.GET("/v1/reservations", h.List)
	r.POST("/v1/reservations/:id/check-in", h.CheckIn)
	return r, logs
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) errorBody {
	t.Helper()
	var body errorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v; body=%s", err, w.Body.String())
	}
	return body
}

func assertLoggedInternalError(t *testing.T, logs *observer.ObservedLogs) {
	t.Helper()
	entries := logs.FilterLevelExact(zapcore.ErrorLevel).All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 error log before 500, got %d", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["error"] != errDB.Error() {
		t.Fatalf("logged error = %v, want %q", fields["error"], errDB.Error())
	}
}

func TestList(t *testing.T) {
	tests := []struct {
		name         string
		tenant       string
		query        string
		svcErr       error
		wantStatus   int
		wantCode     string
		wantSvcCalls int
	}{
		{name: "missing tenant", tenant: "", query: "?hotel_id=" + testHotel, wantStatus: http.StatusUnauthorized, wantCode: "UNAUTHORIZED"},
		{name: "missing hotel_id", tenant: testTenant, query: "", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "empty hotel_id", tenant: testTenant, query: "?hotel_id=", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "non-uuid hotel_id", tenant: testTenant, query: "?hotel_id=tokyo", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "sql injection hotel_id", tenant: testTenant, query: "?hotel_id=1%27%20OR%201%3D1", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "uuid missing a group", tenant: testTenant, query: "?hotel_id=00000000-0000-0000-000000000001", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "valid unversioned seed uuid", tenant: testTenant, query: "?hotel_id=" + testHotel, wantStatus: http.StatusOK, wantSvcCalls: 1},
		{name: "valid uppercase uuid", tenant: testTenant, query: "?hotel_id=ABCDEF00-0000-4000-8000-000000000001", wantStatus: http.StatusOK, wantSvcCalls: 1},
		{name: "service error", tenant: testTenant, query: "?hotel_id=" + testHotel, svcErr: errDB, wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR", wantSvcCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{listErr: tt.svcErr}
			r, logs := newRouter(svc)
			req := httptest.NewRequest(http.MethodGet, "/v1/reservations"+tt.query, nil)
			if tt.tenant != "" {
				req.Header.Set("X-Tenant-ID", tt.tenant)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
			if svc.listCalls != tt.wantSvcCalls {
				t.Fatalf("service called %d times, want %d", svc.listCalls, tt.wantSvcCalls)
			}
			if tt.wantCode != "" {
				body := decodeError(t, w)
				if body.Error.Code != tt.wantCode {
					t.Fatalf("error code = %q, want %q", body.Error.Code, tt.wantCode)
				}
				if tt.wantCode == "VALIDATION_ERROR" && body.Error.Message != "hotel_id is required and must be a UUID" {
					t.Fatalf("message = %q", body.Error.Message)
				}
			}
			if tt.wantStatus == http.StatusInternalServerError {
				assertLoggedInternalError(t, logs)
				if strings.Contains(w.Body.String(), errDB.Error()) {
					t.Fatalf("500 body leaks internal error: %s", w.Body.String())
				}
			} else if logs.Len() != 0 {
				t.Fatalf("unexpected error logs: %v", logs.All())
			}
		})
	}
}

func TestCheckIn(t *testing.T) {
	validBody := `{"room_id":"` + testRoom + `"}`
	tests := []struct {
		name          string
		tenant        string
		reservationID string
		body          string
		svcErr        error
		wantStatus    int
		wantCode      string
		wantSvcCalls  int
	}{
		{name: "missing tenant", tenant: "", reservationID: testReservation, body: validBody, wantStatus: http.StatusUnauthorized, wantCode: "UNAUTHORIZED"},
		{name: "non-uuid reservation id", tenant: testTenant, reservationID: "abc", body: validBody, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "missing body", tenant: testTenant, reservationID: testReservation, body: "", wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "malformed json", tenant: testTenant, reservationID: testReservation, body: `{"room_id":`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "missing room_id", tenant: testTenant, reservationID: testReservation, body: `{}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "empty room_id", tenant: testTenant, reservationID: testReservation, body: `{"room_id":""}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "non-uuid room_id", tenant: testTenant, reservationID: testReservation, body: `{"room_id":"101"}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "room_id wrong type", tenant: testTenant, reservationID: testReservation, body: `{"room_id":101}`, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "reservation not found", tenant: testTenant, reservationID: testReservation, body: validBody, svcErr: repository.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "NOT_FOUND", wantSvcCalls: 1},
		{name: "room not found", tenant: testTenant, reservationID: testReservation, body: validBody, svcErr: repository.ErrRoomNotFound, wantStatus: http.StatusNotFound, wantCode: "ROOM_NOT_FOUND", wantSvcCalls: 1},
		{name: "room type mismatch", tenant: testTenant, reservationID: testReservation, body: validBody, svcErr: repository.ErrRoomTypeMismatch, wantStatus: http.StatusBadRequest, wantCode: "ROOM_TYPE_MISMATCH", wantSvcCalls: 1},
		{name: "room occupied", tenant: testTenant, reservationID: testReservation, body: validBody, svcErr: repository.ErrRoomOccupied, wantStatus: http.StatusConflict, wantCode: "ROOM_OCCUPIED", wantSvcCalls: 1},
		{name: "concurrent check-in holds lock", tenant: testTenant, reservationID: testReservation, body: validBody, svcErr: service.ErrCheckInInProgress, wantStatus: http.StatusConflict, wantCode: "CHECK_IN_IN_PROGRESS", wantSvcCalls: 1},
		{name: "wrapped room occupied", tenant: testTenant, reservationID: testReservation, body: validBody, svcErr: errors.Join(errors.New("ctx"), repository.ErrRoomOccupied), wantStatus: http.StatusConflict, wantCode: "ROOM_OCCUPIED", wantSvcCalls: 1},
		{name: "service error", tenant: testTenant, reservationID: testReservation, body: validBody, svcErr: errDB, wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR", wantSvcCalls: 1},
		{name: "success", tenant: testTenant, reservationID: testReservation, body: validBody, wantStatus: http.StatusOK, wantSvcCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{checkInErr: tt.svcErr}
			r, logs := newRouter(svc)
			req := httptest.NewRequest(http.MethodPost, "/v1/reservations/"+tt.reservationID+"/check-in", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.tenant != "" {
				req.Header.Set("X-Tenant-ID", tt.tenant)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
			if svc.checkInCalls != tt.wantSvcCalls {
				t.Fatalf("service called %d times, want %d", svc.checkInCalls, tt.wantSvcCalls)
			}
			if tt.wantSvcCalls > 0 && svc.gotRoomID != testRoom {
				t.Fatalf("service got room_id %q, want %q", svc.gotRoomID, testRoom)
			}
			if tt.wantCode != "" {
				if body := decodeError(t, w); body.Error.Code != tt.wantCode {
					t.Fatalf("error code = %q, want %q", body.Error.Code, tt.wantCode)
				}
			}
			if tt.wantStatus == http.StatusInternalServerError {
				assertLoggedInternalError(t, logs)
				if strings.Contains(w.Body.String(), errDB.Error()) {
					t.Fatalf("500 body leaks internal error: %s", w.Body.String())
				}
			} else if logs.Len() != 0 {
				t.Fatalf("unexpected error logs: %v", logs.All())
			}
		})
	}
}
