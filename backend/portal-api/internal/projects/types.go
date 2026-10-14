package projects

import (
	"context"
	"errors"
)

const TemplateVersion = "1.0.0"

var ErrNotFound = errors.New("project not found")
var ErrConflict = errors.New("project ID already exists")

type CreateRequest struct {
	Name       string `json:"name"`
	TemplateID string `json:"templateId"`
}

type Project struct {
	ID              string `json:"id" dynamodbav:"id"`
	Name            string `json:"name" dynamodbav:"name"`
	TemplateID      string `json:"templateId" dynamodbav:"templateId"`
	TemplateVersion string `json:"templateVersion" dynamodbav:"templateVersion"`
	ModuleName      string `json:"moduleName,omitempty" dynamodbav:"moduleName,omitempty"`
	CreatedAt       string `json:"createdAt" dynamodbav:"createdAt"`
	Status          string `json:"status" dynamodbav:"status"`
	DownloadURL     string `json:"downloadUrl" dynamodbav:"downloadUrl"`
}

type Page struct {
	Projects   []Project `json:"projects"`
	NextCursor string    `json:"nextCursor,omitempty"`
}

type Store interface {
	Put(context.Context, Project) error
	Get(context.Context, string) (Project, error)
	List(context.Context, int, string) (Page, error)
	Ready(context.Context) error
	Mode() string
}

type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Files       int    `json:"files"`
	Version     string `json:"version"`
}

func Templates() []Template {
	return []Template{
		{ID: "go-gin-api", Name: "Go + Gin API", Description: "A ready-to-run Go API with a health endpoint and Dockerfile.", Language: "Go", Files: 4, Version: TemplateVersion},
		{ID: "react-frontend", Name: "React Frontend", Description: "A React application with Vite, responsive styling, and Docker support.", Language: "JavaScript", Files: 6, Version: TemplateVersion},
	}
}

func KnownTemplate(id string) bool {
	for _, t := range Templates() {
		if t.ID == id {
			return true
		}
	}
	return false
}
