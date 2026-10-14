package projects

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func requestBody(c *gin.Context) (CreateRequest, bool) {
	var request CreateRequest
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		c.JSON(400, gin.H{"error": "Send a JSON object with name and templateId fields."})
		return request, false
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		c.JSON(400, gin.H{"error": "Send exactly one valid JSON object."})
		return request, false
	}
	request.Name = strings.TrimSpace(request.Name)
	length := utf8.RuneCountInString(request.Name)
	if length < 3 || length > 50 {
		c.JSON(400, gin.H{"error": "Project name must contain between 3 and 50 characters."})
		return request, false
	}
	if !KnownTemplate(request.TemplateID) {
		c.JSON(400, gin.H{"error": "Choose a supported templateId."})
		return request, false
	}
	return request, true
}
func NewRouter(store Store) (*gin.Engine, error) {
	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		if err := store.Ready(c.Request.Context()); err != nil {
			c.JSON(503, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ok", "storage": store.Mode()})
	})
	r.GET("/api/status", func(c *gin.Context) {
		status := "ok"
		code := 200
		if err := store.Ready(c.Request.Context()); err != nil {
			status = "unavailable"
			code = 503
		}
		c.JSON(code, gin.H{"status": status, "storage": store.Mode(), "service": "project-service"})
	})
	r.GET("/api/templates", func(c *gin.Context) { c.JSON(200, gin.H{"templates": Templates()}) })
	r.GET("/api/templates/:id", func(c *gin.Context) {
		for _, t := range Templates() {
			if t.ID == c.Param("id") {
				c.JSON(200, t)
				return
			}
		}
		c.JSON(404, gin.H{"error": "Template not found"})
	})
	create := func(c *gin.Context, download bool) {
		request, ok := requestBody(c)
		if !ok {
			return
		}
		p, err := NewProject(request)
		if err != nil {
			serverError(c, err)
			return
		}
		archive, err := Generate(p)
		if err != nil {
			serverError(c, err)
			return
		}
		if err = store.Put(c.Request.Context(), p); err != nil {
			serverError(c, err)
			return
		}
		if download {
			sendZIP(c, p, archive)
			return
		}
		c.Header("Location", "/api/projects/"+p.ID)
		c.JSON(201, gin.H{"message": "Project created", "id": p.ID, "name": p.Name, "templateId": p.TemplateID, "project": p})
	}
	r.POST("/api/projects", func(c *gin.Context) { create(c, false) })
	r.POST("/api/projects/download", func(c *gin.Context) { create(c, true) })
	r.GET("/api/projects", func(c *gin.Context) {
		limit := 100
		if raw := c.Query("limit"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 || n > 100 {
				c.JSON(400, gin.H{"error": "limit must be between 1 and 100"})
				return
			}
			limit = n
		}
		page, err := store.List(c.Request.Context(), limit, c.Query("cursor"))
		if err != nil {
			if errors.Is(err, ErrCursor) {
				c.JSON(400, gin.H{"error": "Invalid pagination cursor"})
				return
			}
			serverError(c, err)
			return
		}
		c.JSON(200, page)
	})
	r.GET("/api/projects/:id", func(c *gin.Context) {
		p, err := store.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			storeError(c, err)
			return
		}
		c.JSON(200, p)
	})
	r.GET("/api/projects/:id/download", func(c *gin.Context) {
		p, err := store.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			storeError(c, err)
			return
		}
		archive, err := Generate(p)
		if err != nil {
			serverError(c, err)
			return
		}
		sendZIP(c, p, archive)
	})
	return r, nil
}
func sendZIP(c *gin.Context, p Project, data []byte) {
	c.Header("Content-Disposition", `attachment; filename="`+p.ID+`.zip"`)
	c.Header("X-Project-ID", p.ID)
	c.Header("Cache-Control", "private, no-store")
	c.Data(http.StatusOK, "application/zip", data)
}
func storeError(c *gin.Context, err error) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(404, gin.H{"error": "Project not found"})
		return
	}
	serverError(c, err)
}
func serverError(c *gin.Context, err error) {
	log.Printf("project request failed: %v", err)
	c.JSON(500, gin.H{"error": "The project service could not complete the request. Please try again."})
}
func Server(addr string, handler http.Handler) *http.Server {
	return &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
}
