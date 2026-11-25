package presentation

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"configgen/internal/domain"
	"configgen/web"
)

// UseCase 是 presentation 依赖的用例接口（定义在消费方）。
type UseCase interface {
	Plan(sel domain.Selection) (domain.Plan, error)
	Render(sel domain.Selection) (domain.Record, error)
	GetRecord(id int64) (domain.Record, error)
	ListRecords(offset, limit int) ([]domain.RecordSummary, int, error)
	Health() error
}

type Server struct {
	router *gin.Engine
	uc     UseCase
}

var distFS = func() fs.FS {
	sub, err := fs.Sub(web.FS, "dist")
	if err != nil {
		panic("web: embedded dist directory missing: " + err.Error())
	}
	return sub
}()

func NewServer(uc UseCase) *Server {
	router := gin.New()
	router.Use(gin.Recovery(), requestLogger())
	s := &Server{router: router, uc: uc}

	router.GET("/health", s.health)
	router.GET("/api/v1/catalog", s.catalog)
	router.POST("/api/v1/plan", s.plan)
	router.POST("/api/v1/render", s.render)
	router.GET("/api/v1/records", s.listRecords)
	router.GET("/api/v1/records/:id", s.getRecord)
	router.GET("/", s.index)
	router.NoRoute(s.spaFallback)
	return s
}

func (s *Server) Handler() http.Handler { return s.router }

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("http_request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
}

func (s *Server) health(c *gin.Context) {
	if err := s.uc.Health(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "error": "database unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) catalog(c *gin.Context) {
	c.JSON(http.StatusOK, domain.CatalogView())
}

func decodeSelection(c *gin.Context) (domain.Selection, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var sel domain.Selection
	if err := decoder.Decode(&sel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体解析失败", "details": err.Error()})
		return domain.Selection{}, false
	}
	return sel, true
}

func (s *Server) plan(c *gin.Context) {
	sel, ok := decodeSelection(c)
	if !ok {
		return
	}
	plan, err := s.uc.Plan(sel)
	if err != nil {
		writeError(c, plan, err)
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (s *Server) render(c *gin.Context) {
	sel, ok := decodeSelection(c)
	if !ok {
		return
	}
	rec, err := s.uc.Render(sel)
	if err != nil {
		writeError(c, rec.Plan, err)
		return
	}
	c.JSON(http.StatusCreated, rec)
}

func writeError(c *gin.Context, plan domain.Plan, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		status := http.StatusBadRequest
		if appErr.Kind == domain.ErrUnprocessable {
			status = http.StatusUnprocessableEntity
		}
		payload := gin.H{"error": appErr.Message, "diagnostics": appErr.Diagnostics}
		if len(plan.Nodes) > 0 {
			payload["plan"] = plan
		}
		c.JSON(status, payload)
		return
	}
	slog.Error("request failed", "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "内部错误"})
}

func (s *Server) listRecords(c *gin.Context) {
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offset 必须是非负整数"})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}
	records, total, err := s.uc.ListRecords(offset, limit)
	if err != nil {
		slog.Error("list records failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询历史记录失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"records": records, "total": total, "offset": offset, "limit": limit})
}

func (s *Server) getRecord(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id 必须是整数"})
		return
	}
	rec, err := s.uc.GetRecord(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		slog.Error("get record failed", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询记录失败"})
		return
	}
	c.JSON(http.StatusOK, rec)
}

func (s *Server) index(c *gin.Context) { s.serveIndex(c) }

func (s *Server) serveIndex(c *gin.Context) {
	data, err := web.FS.ReadFile("dist/index.html")
	if err != nil {
		c.String(http.StatusOK, "ConfigGen 已启动，API: /api/v1/catalog")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

func (s *Server) spaFallback(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	name := strings.TrimPrefix(c.Request.URL.Path, "/")
	if name != "" && fs.ValidPath(name) {
		if f, err := distFS.Open(name); err == nil {
			f.Close()
			c.FileFromFS(name, http.FS(distFS))
			return
		}
	}
	s.serveIndex(c)
}
