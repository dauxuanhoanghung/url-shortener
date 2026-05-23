package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dauxuanhoanghung/url-shortener/pkg/utils"
	"github.com/gin-gonic/gin"
)

const testJWTSecret = "middleware-test-secret"

func init() { gin.SetMode(gin.TestMode) }

// newAuthRouter wires AuthRequired in front of a tiny handler that echoes
// the userID/role context keys, so tests can assert middleware side-effects.
func newAuthRouter() *gin.Engine {
	r := gin.New()
	r.Use(AuthRequired(testJWTSecret))
	r.GET("/whoami", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"userID": c.GetString("userID"),
			"role":   c.GetString("role"),
		})
	})
	return r
}

func TestAuthRequired_MissingHeader(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	newAuthRouter().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "UNAUTHORIZED") {
		t.Errorf("body: got %q", w.Body.String())
	}
}

func TestAuthRequired_InvalidToken(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-jwt")
	newAuthRouter().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "INVALID_TOKEN") {
		t.Errorf("expected INVALID_TOKEN, got %q", w.Body.String())
	}
}

func TestAuthRequired_WrongSecret(t *testing.T) {
	// Token signed with a different secret must be rejected.
	tok, err := utils.GenerateAccessToken(
		utils.TokenInput{UserID: "u1", Email: "u@e.com", Role: "user"},
		"different-secret",
	)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	newAuthRouter().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d want 401", w.Code)
	}
}

func TestAuthRequired_ValidToken_PopulatesContext(t *testing.T) {
	tok, err := utils.GenerateAccessToken(
		utils.TokenInput{UserID: "user-42", Email: "u@e.com", Role: "user"},
		testJWTSecret,
	)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	newAuthRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200, body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"userID":"user-42"`) {
		t.Errorf("userID missing from response: %s", body)
	}
	if !strings.Contains(body, `"role":"user"`) {
		t.Errorf("role missing from response: %s", body)
	}
}

func TestAuthRequired_QueryTokenForSSE(t *testing.T) {
	// SSE connections fall back to ?token= because EventSource cannot set
	// custom headers. Verify both paths populate the context identically.
	tok, err := utils.GenerateAccessToken(
		utils.TokenInput{UserID: "sse-user", Email: "u@e.com", Role: "user"},
		testJWTSecret,
	)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/whoami?token="+tok, nil)
	newAuthRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"userID":"sse-user"`) {
		t.Errorf("query-param token path did not populate context: %s", w.Body.String())
	}
}

func TestAuthRequired_NonBearerScheme(t *testing.T) {
	// Authorization header without the "Bearer " prefix must be rejected
	// rather than silently treated as no auth.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	newAuthRouter().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d want 401", w.Code)
	}
}
