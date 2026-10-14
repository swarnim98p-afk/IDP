package gateway

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func NewRouter(upstream, staticDir string) (*gin.Engine, error) {
	target, err := url.Parse(upstream)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") {
		return nil, fmt.Errorf("invalid PROJECT_SERVICE_URL")
	}
	r := gin.Default()
	if err = r.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	proxy.Director = func(req *http.Request) { director(req); req.Host = target.Host }
	proxy.Transport = &http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext, ResponseHeaderTimeout: 15 * time.Second, IdleConnTimeout: 60 * time.Second}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(502)
		w.Write([]byte(`{"error":"The project service is unavailable. Please try again shortly."}`))
	}
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		req, err := http.NewRequestWithContext(c.Request.Context(), "GET", strings.TrimRight(upstream, "/")+"/readyz", nil)
		if err != nil {
			c.JSON(503, gin.H{"status": "unavailable"})
			return
		}
		client := http.Client{Timeout: 3 * time.Second}
		res, err := client.Do(req)
		if err != nil {
			c.JSON(503, gin.H{"status": "unavailable"})
			return
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			c.JSON(503, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.Any("/api/*path", func(c *gin.Context) { proxy.ServeHTTP(c.Writer, c.Request) })
	if staticDir != "" {
		absolute, err := filepath.Abs(staticDir)
		if err != nil {
			return nil, err
		}
		files := http.FileServer(http.Dir(absolute))
		r.NoRoute(func(c *gin.Context) {
			if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
				c.JSON(404, gin.H{"error": "Not found"})
				return
			}
			path := filepath.Join(absolute, filepath.FromSlash(strings.TrimPrefix(c.Request.URL.Path, "/")))
			relative, err := filepath.Rel(absolute, path)
			if err != nil || strings.HasPrefix(relative, "..") {
				c.AbortWithStatus(404)
				return
			}
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				if strings.HasPrefix(relative, "assets"+string(filepath.Separator)) {
					c.Header("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(c.Writer, c.Request)
				return
			}
			if strings.HasPrefix(c.Request.URL.Path, "/assets/") {
				c.AbortWithStatus(404)
				return
			}
			c.Header("Cache-Control", "no-cache")
			c.File(filepath.Join(absolute, "index.html"))
		})
	}
	return r, nil
}
