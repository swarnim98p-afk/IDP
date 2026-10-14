package projects

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateDownloadAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")
	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(store)
	if err != nil {
		t.Fatal(err)
	}
	for _, templateID := range []string{"go-gin-api", "react-frontend"} {
		body, _ := json.Marshal(CreateRequest{Name: `../payments "team"`, TemplateID: templateID})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("POST", "/api/projects/download", bytes.NewReader(body)))
		if response.Code != 200 || response.Header().Get("Content-Type") != "application/zip" {
			t.Fatalf("download: %d %s", response.Code, response.Body.String())
		}
		id := response.Header().Get("X-Project-ID")
		reloaded, err := NewFileStore(path)
		if err != nil {
			t.Fatal(err)
		}
		saved, err := reloaded.Get(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if saved.Name != `../payments "team"` || saved.TemplateID != templateID {
			t.Fatalf("wrong saved project: %+v", saved)
		}
		archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
		if err != nil {
			t.Fatal(err)
		}
		wantFiles := 4
		if templateID == "react-frontend" {
			wantFiles = 6
		}
		if len(archive.File) != wantFiles {
			t.Fatalf("got %d files", len(archive.File))
		}
		for _, file := range archive.File {
			if strings.Contains(file.Name, "..") || filepath.IsAbs(file.Name) {
				t.Fatalf("unsafe archive path %s", file.Name)
			}
			reader, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			content, err := io.ReadAll(reader)
			reader.Close()
			if err != nil {
				t.Fatal(err)
			}
			if file.Name == "go.mod" && !strings.Contains(string(content), saved.ModuleName) {
				t.Fatal("unique module name missing")
			}
			if file.Name == "src/main.jsx" && !strings.Contains(string(content), `\"team\"`) {
				t.Fatal("project name was not escaped for JavaScript")
			}
		}
		router2, _ := NewRouter(reloaded)
		download := httptest.NewRecorder()
		router2.ServeHTTP(download, httptest.NewRequest("GET", saved.DownloadURL, nil))
		if download.Code != 200 || !bytes.Equal(download.Body.Bytes(), response.Body.Bytes()) {
			t.Fatal("saved download changed after reload")
		}
	}
	reloaded, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	page, err := reloaded.List(context.Background(), 1, "")
	if err != nil || len(page.Projects) != 1 || page.NextCursor == "" {
		t.Fatalf("first page: %+v %v", page, err)
	}
	second, err := reloaded.List(context.Background(), 1, page.NextCursor)
	if err != nil || len(second.Projects) != 1 || second.Projects[0].ID == page.Projects[0].ID || second.NextCursor != "" {
		t.Fatalf("second page: %+v %v", second, err)
	}
	if err := reloaded.Put(context.Background(), second.Projects[0]); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := reloaded.Get(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}

func TestInvalidRequestsDoNotSave(t *testing.T) {
	store := NewMemoryStore()
	router, _ := NewRouter(store)
	for _, body := range []string{`{"name":"payments","templateId":"go-gin-api","unexpected":true}`, strings.Repeat(" ", 65<<10), `{"name":"bad","templateId":"unknown"}`} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("POST", "/api/projects", strings.NewReader(body)))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("got %d", response.Code)
		}
	}
	page, _ := store.List(context.Background(), 100, "")
	if len(page.Projects) != 0 {
		t.Fatal("invalid request saved a project")
	}
	for _, url := range []string{"/api/projects?limit=0", "/api/projects?limit=101", "/api/projects?cursor=***"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", url, nil))
		if response.Code != 400 {
			t.Fatalf("%s: %d", url, response.Code)
		}
	}
}

func TestPersistenceFailureDoesNotRetainProject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(path, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	store, _ := NewFileStore(filepath.Join(path, "projects.json"))
	if store == nil {
		store = NewMemoryStore()
		store.file = filepath.Join(path, "projects.json")
	}
	p, _ := NewProject(CreateRequest{Name: "payments", TemplateID: "go-gin-api"})
	if err := store.Put(context.Background(), p); err == nil {
		t.Fatal("expected persistence failure")
	}
	if _, err := store.Get(context.Background(), p.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("failed write left a phantom project")
	}
}

func TestDynamoDBLocal(t *testing.T) {
	endpoint := os.Getenv("TEST_DYNAMODB_ENDPOINT")
	if endpoint == "" {
		t.Skip("set TEST_DYNAMODB_ENDPOINT to run against DynamoDB Local")
	}
	ctx := context.Background()
	store, err := NewDynamoStore(ctx, "portal-integration-tests", "ap-south-1", endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.EnsureLocalTable(ctx); err != nil {
		t.Fatal(err)
	}
	p, _ := NewProject(CreateRequest{Name: "dynamo-roundtrip", TemplateID: "go-gin-api"})
	if err = store.Put(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, p.ID)
	if err != nil || got != p {
		t.Fatalf("round trip %+v %v", got, err)
	}
	if err = store.Put(ctx, p); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err = store.Get(ctx, "does-not-exist"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	seen := false
	cursor := ""
	for {
		page, err := store.List(ctx, 1, cursor)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range page.Projects {
			if item.ID == p.ID {
				seen = true
			}
		}
		cursor = page.NextCursor
		if cursor == "" {
			break
		}
	}
	if !seen {
		t.Fatal("saved project missing from paginated scan")
	}
}
