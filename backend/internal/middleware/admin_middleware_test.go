package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/gin-gonic/gin"
)

// newAdminRouter pretends AuthRequired has already run by injecting the role
// directly into the context — that's exactly the contract RequireAdmin
// relies on, and isolating it keeps these tests focused on the role check.
func newAdminRouter(setRole string) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if setRole != "" {
			c.Set("role", setRole)
		}
		c.Next()
	})
	r.Use(RequireAdmin())
	r.GET("/admin/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRequireAdmin_MissingRoleContext(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	newAdminRouter("").ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d want 403", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ADMIN_REQUIRED") {
		t.Errorf("expected ADMIN_REQUIRED code, body=%s", w.Body.String())
	}
}

func TestRequireAdmin_UserRoleRejected(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	newAdminRouter(model.RoleUser).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d want 403", w.Code)
	}
}

func TestRequireAdmin_AdminRoleAccepted(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	newAdminRouter(model.RoleAdmin).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d want 200, body=%s", w.Code, w.Body.String())
	}
}
