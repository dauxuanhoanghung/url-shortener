package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── test router helper ────────────────────────────────────────────────────────

func newURLRouter(svc service.URLService) *gin.Engine {
	r := gin.New()
	h := NewURLHandler(svc, "https://short.ly")
	// Inject a fixed userID so auth middleware is not needed in tests.
	authed := r.Group("")
	authed.Use(func(c *gin.Context) {
		c.Set("userID", uuid.New().String())
		c.Next()
	})
	authed.POST("/urls", h.Create)
	authed.GET("/urls", h.List)
	authed.DELETE("/urls/:id", h.Delete)
	return r
}

func decodeError(t *testing.T, body []byte) string {
	t.Helper()
	var resp dto.ErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode error response: %v — body: %s", err, body)
	}
	return resp.Error.Code
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestURLHandler_Create_201(t *testing.T) {
	svc := &fakeURLService{
		createResp: &dto.URLResponse{
			ID:        uuid.NewString(),
			ShortCode: "abc123",
			ShortURL:  "https://short.ly/r/abc123",
		},
	}
	r := newURLRouter(svc)

	body, _ := json.Marshal(dto.CreateURLRequest{OriginalURL: "https://example.com"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status: got %d want 201 — body: %s", w.Code, w.Body)
	}
}

func TestURLHandler_Create_InvalidURL_400(t *testing.T) {
	svc := &fakeURLService{createErr: service.ErrInvalidURL}
	r := newURLRouter(svc)

	body, _ := json.Marshal(dto.CreateURLRequest{OriginalURL: "javascript:alert(1)"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d want 400", w.Code)
	}
	if code := decodeError(t, w.Body.Bytes()); code != "INVALID_URL" {
		t.Errorf("error code: got %q want INVALID_URL", code)
	}
}

func TestURLHandler_Create_PlanLimit_403(t *testing.T) {
	svc := &fakeURLService{createErr: service.ErrPlanLimitReached}
	r := newURLRouter(svc)

	body, _ := json.Marshal(dto.CreateURLRequest{OriginalURL: "https://example.com"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d want 403", w.Code)
	}
	if code := decodeError(t, w.Body.Bytes()); code != "PLAN_LIMIT_REACHED" {
		t.Errorf("error code: got %q want PLAN_LIMIT_REACHED", code)
	}
}

func TestURLHandler_Create_MissingBody_400(t *testing.T) {
	r := newURLRouter(&fakeURLService{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/urls", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d want 400", w.Code)
	}
}

func TestURLHandler_Create_ShortCodeRetry_500(t *testing.T) {
	svc := &fakeURLService{createErr: service.ErrShortCodeRetry}
	r := newURLRouter(svc)

	body, _ := json.Marshal(dto.CreateURLRequest{OriginalURL: "https://example.com"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d want 500", w.Code)
	}
	if code := decodeError(t, w.Body.Bytes()); code != "SHORT_CODE_UNAVAILABLE" {
		t.Errorf("error code: got %q want SHORT_CODE_UNAVAILABLE", code)
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestURLHandler_List_200(t *testing.T) {
	svc := &fakeURLService{
		listResp: &dto.ListURLResponse{
			URLs: []dto.URLResponse{
				{
					ID:        uuid.NewString(),
					ShortCode: "abc",
					ShortURL:  "https://short.ly/r/abc",
					Metadata: &dto.URLMetadataResponse{
						FetchStatus: "ok",
						Title:       ptrStr("Example"),
					},
				},
			},
			Total: 1,
		},
	}
	r := newURLRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/urls", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d want 200 — body: %s", w.Code, w.Body)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			URLs []struct {
				Metadata *struct {
					FetchStatus string  `json:"fetch_status"`
					Title       *string `json:"title"`
				} `json:"metadata"`
			} `json:"urls"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.Total != 1 {
		t.Errorf("total: got %d want 1", resp.Data.Total)
	}
	if len(resp.Data.URLs) != 1 {
		t.Fatalf("urls length: got %d want 1", len(resp.Data.URLs))
	}
	m := resp.Data.URLs[0].Metadata
	if m == nil {
		t.Fatal("expected metadata in list response")
	}
	if m.FetchStatus != "ok" {
		t.Errorf("fetch_status: got %q want ok", m.FetchStatus)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestURLHandler_Delete_200(t *testing.T) {
	r := newURLRouter(&fakeURLService{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/urls/"+uuid.NewString(), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d want 200", w.Code)
	}
}

func TestURLHandler_Delete_NotFound_404(t *testing.T) {
	svc := &fakeURLService{deleteErr: service.ErrURLNotFound}
	r := newURLRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/urls/"+uuid.NewString(), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d want 404", w.Code)
	}
	if code := decodeError(t, w.Body.Bytes()); code != "SHORT_CODE_NOT_FOUND" {
		t.Errorf("error code: got %q want SHORT_CODE_NOT_FOUND", code)
	}
}

func TestURLHandler_Delete_Forbidden_403(t *testing.T) {
	svc := &fakeURLService{deleteErr: service.ErrURLForbidden}
	r := newURLRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/urls/"+uuid.NewString(), nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d want 403", w.Code)
	}
	if code := decodeError(t, w.Body.Bytes()); code != "FORBIDDEN" {
		t.Errorf("error code: got %q want FORBIDDEN", code)
	}
}

func TestURLHandler_Delete_InvalidID_400(t *testing.T) {
	r := newURLRouter(&fakeURLService{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/urls/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d want 400", w.Code)
	}
}

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeURLService struct {
	createResp *dto.URLResponse
	createErr  error
	listResp   *dto.ListURLResponse
	listErr    error
	deleteErr  error
}

func (f *fakeURLService) Create(_ context.Context, _ uuid.UUID, _ dto.CreateURLRequest, _ string) (*dto.URLResponse, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResp != nil {
		return f.createResp, nil
	}
	return &dto.URLResponse{
		ID:        uuid.NewString(),
		ShortCode: "abc123",
		ShortURL:  "https://short.ly/r/abc123",
		CreatedAt: time.Now(),
	}, nil
}

func (f *fakeURLService) List(_ context.Context, _ uuid.UUID, _, _ int, _ string) (*dto.ListURLResponse, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &dto.ListURLResponse{URLs: []dto.URLResponse{}, Total: 0}, nil
}

func (f *fakeURLService) Delete(_ context.Context, _, _ uuid.UUID) error {
	return f.deleteErr
}

func ptrStr(s string) *string { return &s }

// ── ensure fake satisfies interface at compile time ───────────────────────────
var _ service.URLService = (*fakeURLService)(nil)

// ensure no unused import on errors
var _ = errors.New
