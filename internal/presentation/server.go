package presentation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"configgen/internal/domain"
	"configgen/internal/usecase"
	"configgen/web"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

var (
	log      = logrus.New()
	validate = validator.New()
)

func init() {
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(logrus.StandardLogger().Out)
	if gin.Mode() == gin.ReleaseMode {
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetLevel(logrus.DebugLevel)
	}
}

type Server struct {
	router    *gin.Engine
	configGen *usecase.ConfigGenerator
	srv       *http.Server
}

func NewServer(configGen *usecase.ConfigGenerator) *Server {
	if gin.Mode() == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}
	g := gin.Default()

	s := &Server{router: g, configGen: configGen}

	g.GET("/", s.home)
	g.GET("/health", s.health)
	g.GET("/web/views/:name", s.serveView)
	g.POST("/api/v1/generate/compose", s.generateCompose)
	g.POST("/api/v1/generate/k8s", s.generateK8s)
	g.POST("/api/v1/generate/:type", s.generateWithType)
	g.POST("/api/v1/generate", s.generate)
	g.GET("/api/v1/configs", s.listConfigs)
	g.GET("/api/v1/configs/:id", s.getConfig)

	return s
}

func (s *Server) Run(addr string) error {
	s.srv = &http.Server{Addr: addr, Handler: s.router}
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) home(c *gin.Context) {
	data, err := web.FS.ReadFile("index.html")
	if err != nil {
		c.String(http.StatusOK, "ConfigGen 已启动，API: /api/v1/generate, /api/v1/configs/:id")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

func (s *Server) serveView(c *gin.Context) {
	name := c.Param("name")
	data, err := web.FS.ReadFile("views/" + name)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

func parseConfigRequest(raw []byte) (domain.ConfigRequest, error) {
	var req domain.ConfigRequest
	if err := json.Unmarshal(raw, &req); err == nil {
		return req, nil
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return req, err
	}

	if v, ok := rawMap["port"]; ok {
		switch t := v.(type) {
		case string:
			if n, err := strconv.Atoi(t); err == nil {
				req.Port = n
			}
		case float64:
			req.Port = int(t)
		}
	}
	if v, ok := rawMap["replicas"]; ok {
		switch t := v.(type) {
		case string:
			if n, err := strconv.Atoi(t); err == nil {
				req.Replicas = n
			}
		case float64:
			req.Replicas = int(t)
		}
	}

	if v, ok := rawMap["type"].(string); ok {
		req.Type = v
	}
	if v, ok := rawMap["app_name"].(string); ok {
		req.AppName = v
	}
	if v, ok := rawMap["image"].(string); ok {
		req.Image = v
	}
	if v, ok := rawMap["tag"].(string); ok {
		req.Tag = v
	}
	if v, ok := rawMap["k8s_resource"]; ok {
		switch t := v.(type) {
		case string:
			if t != "" {
				req.K8sResource = []string{t}
			}
		case []interface{}:
			resources := make([]string, len(t))
			for i, item := range t {
				if s, ok := item.(string); ok {
					resources[i] = s
				}
			}
			req.K8sResource = resources
		}
	}
	if v, ok := rawMap["env"].(map[string]interface{}); ok {
		env := map[string]string{}
		for k, val := range v {
			if vs, ok := val.(string); ok {
				env[k] = vs
			}
		}
		req.Env = env
	}

	return req, nil
}

func (s *Server) generateCompose(c *gin.Context) {
	s.generateWithTypeInternal(c, "compose")
}

func (s *Server) generateK8s(c *gin.Context) {
	s.generateWithTypeInternal(c, "k8s")
}

func (s *Server) generateWithType(c *gin.Context) {
	configType := c.Param("type")
	s.generateWithTypeInternal(c, configType)
}

func (s *Server) generate(c *gin.Context) {
	var req domain.ConfigRequest
	rawBody, err := c.GetRawData()
	if err != nil {
		log.WithError(err).Warn("Failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	req, err = parseConfigRequest(rawBody)
	if err != nil {
		log.WithError(err).Warn("Failed to parse request JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format", "details": err.Error()})
		return
	}

	if req.Type == "" {
		req.Type = "k8s"
	}

	s.processGenerateRequest(c, req)
}

func (s *Server) generateWithTypeInternal(c *gin.Context, configType string) {
	var req domain.ConfigRequest
	rawBody, err := c.GetRawData()
	if err != nil {
		log.WithError(err).Warn("Failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	req, err = parseConfigRequest(rawBody)
	if err != nil {
		log.WithError(err).Warn("Failed to parse request JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format", "details": err.Error()})
		return
	}

	req.Type = configType

	s.processGenerateRequest(c, req)
}

func (s *Server) processGenerateRequest(c *gin.Context, req domain.ConfigRequest) {
	s.configGen.Prepare(&req)

	if err := validate.Struct(req); err != nil {
		log.WithError(err).WithField("request", req).Warn("Validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	log.WithFields(logrus.Fields{
		"type":    req.Type,
		"appName": req.AppName,
	}).Info("Generating config")

	rec, err := s.configGen.GenerateAndSave(req)
	if err != nil {
		log.WithError(err).Error("Failed to generate config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate configuration"})
		return
	}

	log.WithField("id", rec.ID).Info("Config generated and saved")
	c.JSON(http.StatusCreated, rec)
}

func (s *Server) getConfig(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be integer"})
		return
	}

	rec, err := s.configGen.GetRecord(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		log.WithError(err).Error("Failed to get config record")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration"})
		return
	}

	c.JSON(http.StatusOK, rec)
}

func (s *Server) listConfigs(c *gin.Context) {
	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "20")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	records, err := s.configGen.ListRecords(offset, limit)
	if err != nil {
		log.WithError(err).Error("Failed to list configs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list configurations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"records": records, "offset": offset, "limit": limit})
}
