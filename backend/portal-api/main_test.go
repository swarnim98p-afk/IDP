package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	router, err := setupRouter()
	if err != nil {
		t.Fatalf("set up router: %v", err)
	}
	return router
}

func projectRequestJSON(t *testing.T, name, templateID string) string {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"name":       name,
		"templateId": templateID,
	})
	if err != nil {
		t.Fatalf("encode project request: %v", err)
	}
	return string(body)
}

func TestProjectRequests(t *testing.T) {
	router := newTestRouter(t)
	validBody := projectRequestJSON(t, "my-project", "go-gin-api")
	tests := []struct {
		name           string
		body           string
		wantStatus     int
		wantName       string
		wantTemplateID string
	}{
		{
			name:       "Go template",
			body:       validBody,
			wantStatus: http.StatusCreated, wantName: "my-project", wantTemplateID: "go-gin-api",
		},
		{
			name:       "React template",
			body:       projectRequestJSON(t, "my-frontend", "react-frontend"),
			wantStatus: http.StatusCreated, wantName: "my-frontend", wantTemplateID: "react-frontend",
		},
		{
			name:       "trim surrounding whitespace before checking minimum length",
			body:       projectRequestJSON(t, " \tapi\n ", "go-gin-api"),
			wantStatus: http.StatusCreated, wantName: "api", wantTemplateID: "go-gin-api",
		},
		{
			name:       "trim surrounding whitespace before checking maximum length",
			body:       projectRequestJSON(t, " "+strings.Repeat("a", 50)+" ", "go-gin-api"),
			wantStatus: http.StatusCreated, wantName: strings.Repeat("a", 50), wantTemplateID: "go-gin-api",
		},
		{
			name:       "Unicode minimum length",
			body:       projectRequestJSON(t, strings.Repeat("界", 3), "go-gin-api"),
			wantStatus: http.StatusCreated, wantName: strings.Repeat("界", 3), wantTemplateID: "go-gin-api",
		},
		{
			name:       "Unicode maximum length counts characters rather than bytes",
			body:       projectRequestJSON(t, strings.Repeat("界", 50), "go-gin-api"),
			wantStatus: http.StatusCreated, wantName: strings.Repeat("界", 50), wantTemplateID: "go-gin-api",
		},
		{
			name:       "trailing JSON whitespace is allowed",
			body:       validBody + " \n\t",
			wantStatus: http.StatusCreated, wantName: "my-project", wantTemplateID: "go-gin-api",
		},
		{name: "name too short", body: projectRequestJSON(t, "ab", "go-gin-api"), wantStatus: http.StatusBadRequest},
		{name: "trimmed name too short", body: projectRequestJSON(t, " ab ", "go-gin-api"), wantStatus: http.StatusBadRequest},
		{name: "name too long", body: projectRequestJSON(t, strings.Repeat("a", 51), "go-gin-api"), wantStatus: http.StatusBadRequest},
		{name: "Unicode name too short", body: projectRequestJSON(t, "界界", "go-gin-api"), wantStatus: http.StatusBadRequest},
		{name: "Unicode name too long", body: projectRequestJSON(t, strings.Repeat("界", 51), "go-gin-api"), wantStatus: http.StatusBadRequest},
		{name: "whitespace-only name", body: projectRequestJSON(t, " \t\n ", "go-gin-api"), wantStatus: http.StatusBadRequest},
		{name: "empty name", body: projectRequestJSON(t, "", "go-gin-api"), wantStatus: http.StatusBadRequest},
		{name: "empty template ID", body: projectRequestJSON(t, "my-project", ""), wantStatus: http.StatusBadRequest},
		{name: "unknown template", body: projectRequestJSON(t, "my-project", "unknown"), wantStatus: http.StatusBadRequest},
		{name: "template ID must match exactly", body: projectRequestJSON(t, "my-project", " go-gin-api "), wantStatus: http.StatusBadRequest},
		{name: "missing name", body: `{"templateId":"go-gin-api"}`, wantStatus: http.StatusBadRequest},
		{name: "missing template ID", body: `{"name":"my-project"}`, wantStatus: http.StatusBadRequest},
		{name: "empty object", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "null name", body: `{"name":null,"templateId":"go-gin-api"}`, wantStatus: http.StatusBadRequest},
		{name: "null template ID", body: `{"name":"my-project","templateId":null}`, wantStatus: http.StatusBadRequest},
		{name: "numeric name", body: `{"name":123,"templateId":"go-gin-api"}`, wantStatus: http.StatusBadRequest},
		{name: "array name", body: `{"name":["my-project"],"templateId":"go-gin-api"}`, wantStatus: http.StatusBadRequest},
		{name: "numeric template ID", body: `{"name":"my-project","templateId":123}`, wantStatus: http.StatusBadRequest},
		{name: "object template ID", body: `{"name":"my-project","templateId":{}}`, wantStatus: http.StatusBadRequest},
		{name: "empty body", body: "", wantStatus: http.StatusBadRequest},
		{name: "malformed JSON", body: `{"name":"my-project",`, wantStatus: http.StatusBadRequest},
		{name: "null body", body: `null`, wantStatus: http.StatusBadRequest},
		{name: "array body", body: `[]`, wantStatus: http.StatusBadRequest},
		{name: "string body", body: `"my-project"`, wantStatus: http.StatusBadRequest},
		{name: "second JSON object", body: validBody + ` {"name":"other","templateId":"react-frontend"}`, wantStatus: http.StatusBadRequest},
		{name: "trailing null", body: validBody + ` null`, wantStatus: http.StatusBadRequest},
		{name: "trailing garbage", body: validBody + ` invalid`, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, tt.wantStatus, response.Body.String())
			}

			var result struct {
				Message    string `json:"message"`
				Name       string `json:"name"`
				TemplateID string `json:"templateId"`
				Error      string `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatalf("decode response JSON: %v", err)
			}
			if tt.wantStatus == http.StatusBadRequest {
				if result.Error == "" {
					t.Error("invalid request must return an error message")
				}
				return
			}
			if result.Message != "Project created" {
				t.Errorf("message = %q, want %q", result.Message, "Project created")
			}
			if result.Name != tt.wantName {
				t.Errorf("name = %q, want %q", result.Name, tt.wantName)
			}
			if result.TemplateID != tt.wantTemplateID {
				t.Errorf("templateId = %q, want %q", result.TemplateID, tt.wantTemplateID)
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	response := httptest.NewRecorder()
	newTestRouter(t).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if result.Status != "ok" {
		t.Errorf("health status = %q, want ok", result.Status)
	}
}

func TestTemplateEndpoints(t *testing.T) {
	router := newTestRouter(t)
	t.Run("list available templates", func(t *testing.T) {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/templates", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
		}
		var result struct {
			Templates []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"templates"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode template list: %v", err)
		}
		if len(result.Templates) != 2 {
			t.Fatalf("template count = %d, want 2", len(result.Templates))
		}
		ids := make(map[string]bool)
		for _, template := range result.Templates {
			ids[template.ID] = true
			if template.Name == "" || template.Description == "" {
				t.Errorf("template %q must include its name and description", template.ID)
			}
		}
		for _, id := range []string{"go-gin-api", "react-frontend"} {
			if !ids[id] {
				t.Errorf("template %q missing from list", id)
			}
		}
	})

	for _, id := range []string{"go-gin-api", "react-frontend"} {
		t.Run("get "+id, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/templates/"+id, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			var result struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatalf("decode template response: %v", err)
			}
			if result.ID != id || result.Name == "" || result.Description == "" {
				t.Errorf("unexpected template response: %+v", result)
			}
		})
	}

	t.Run("unknown template", func(t *testing.T) {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/templates/unknown", nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
		var result struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode error response: %v", err)
		}
		if result.Error == "" {
			t.Error("unknown template must return an error message")
		}
	})
}
