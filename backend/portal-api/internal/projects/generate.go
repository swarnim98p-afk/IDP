package projects

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	assets "internal-developer-portal/portal-api/templates"
)

func NewProject(request CreateRequest) (Project, error) {
	var randomID [16]byte
	if _, err := rand.Read(randomID[:]); err != nil {
		return Project{}, err
	}
	id := "project-" + hex.EncodeToString(randomID[:])
	return Project{ID: id, Name: request.Name, TemplateID: request.TemplateID, TemplateVersion: TemplateVersion, ModuleName: "internal-developer-portal/generated/" + id, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), Status: "ready", DownloadURL: "/api/projects/" + id + "/download"}, nil
}

func Generate(p Project) ([]byte, error) {
	if p.TemplateVersion != TemplateVersion {
		return nil, fmt.Errorf("unsupported template version %q", p.TemplateVersion)
	}
	if !KnownTemplate(p.TemplateID) {
		return nil, fmt.Errorf("unknown template")
	}
	nameJSON, err := json.Marshal(p.Name)
	if err != nil {
		return nil, err
	}
	values := map[string]string{"ModuleName": p.ModuleName, "ProjectID": p.ID, "ProjectNameJSON": string(nameJSON)}
	files := []struct{ source, target string }{{"main.go.tmpl", "main.go"}, {"go.mod.tmpl", "go.mod"}, {"README.md.tmpl", "README.md"}, {"Dockerfile.tmpl", "Dockerfile"}}
	if p.TemplateID == "react-frontend" {
		files = []struct{ source, target string }{{"package.json.tmpl", "package.json"}, {"index.html.tmpl", "index.html"}, {"main.jsx.tmpl", "src/main.jsx"}, {"style.css.tmpl", "src/style.css"}, {"README.md.tmpl", "README.md"}, {"Dockerfile.tmpl", "Dockerfile"}}
	}
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for _, file := range files {
		content, err := assets.Render(p.TemplateID, file.source, values)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", file.source, err)
		}
		entry, err := archive.Create(file.target)
		if err != nil {
			return nil, err
		}
		if _, err = entry.Write([]byte(content)); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
