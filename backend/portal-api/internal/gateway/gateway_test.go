package gateway

import (
	"context"
	"internal-developer-portal/portal-api/internal/projects"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGatewayServesUIAndProxiesAPI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service, _ := projects.NewRouter(projects.NewMemoryStore())
	upstream := httptest.NewServer(service)
	defer upstream.Close()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>portal</h1>"), 0600); err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(upstream.URL, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		path     string
		code     int
		contains string
	}{{"/", 200, "portal"}, {"/projects", 200, "portal"}, {"/api/templates", 200, "go-gin-api"}, {"/readyz", 200, "ok"}, {"/api/projects/missing", 404, "not found"}, {"/assets/missing.js", 404, ""}} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", check.path, nil).WithContext(ctx))
		if response.Code != check.code || !strings.Contains(strings.ToLower(response.Body.String()), strings.ToLower(check.contains)) {
			t.Fatalf("%s: %d %s", check.path, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", "/api/projects", strings.NewReader(`{"name":"gateway-test","templateId":"go-gin-api"}`)).WithContext(ctx))
	if response.Code != http.StatusCreated {
		t.Fatalf("create through gateway: %d %s", response.Code, response.Body.String())
	}
	upstream.Close()
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/api/templates", nil).WithContext(ctx))
	if response.Code != 502 {
		t.Fatalf("outage: %d", response.Code)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/readyz", nil))
	if response.Code != 503 {
		t.Fatalf("outage readiness: %d", response.Code)
	}
}
