package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	kiterrors "github.com/tiamxu/kit/errors"
	"github.com/tiamxu/kit/log"
)

type ServerConfig struct {
	Address         string        `yaml:"address" json:"address"`
	KeepAlive       bool          `yaml:"keep_alive" json:"keep_alive"`
	ReadTimeout     time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout" json:"write_timeout"`
	AccessLogFormat string        `yaml:"access_log_format" json:"access_log_format"`
	StaticPrefix    string        `yaml:"static_prefix" json:"static_prefix"`
	StaticDir       string        `yaml:"static_dir" json:"static_dir"`
	MultipartMemory int64         `yaml:"multipart_memory" json:"multipart_memory"` // 文件上传最大内存，默认32MB
	BodyLimit       int64         `yaml:"body_limit" json:"body_limit"`             // 请求body最大限制
	CORSConfig      *CORSConfig   `yaml:"cors" json:"cors"`
}

type CORSConfig struct {
	AllowOrigins     []string      `yaml:"allow_origins" json:"allow_origins"`
	AllowMethods     []string      `yaml:"allow_methods" json:"allow_methods"`
	AllowHeaders     []string      `yaml:"allow_headers" json:"allow_headers"`
	ExposeHeaders    []string      `yaml:"expose_headers" json:"expose_headers"`
	AllowCredentials bool          `yaml:"allow_credentials" json:"allow_credentials"`
	MaxAge           time.Duration `yaml:"max_age" json:"max_age"`
}

var DefaultAccessLogFormat = `${client_ip} | ${time} | "${method} ${path}" | ${status} | ${bytes_out} | ${user_agent} | ${request_time} | ${request_id} | ${error}`

const (
	defaultMultipartMemory = 32 << 20 // 32MB
	defaultBodyLimit       = 8 << 20 // 8MB
)

func NewGin(cfg ServerConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	if len(cfg.AccessLogFormat) == 0 {
		cfg.AccessLogFormat = DefaultAccessLogFormat
	}

	// 设置默认值
	if cfg.MultipartMemory == 0 {
		cfg.MultipartMemory = defaultMultipartMemory
	}
	if cfg.BodyLimit == 0 {
		cfg.BodyLimit = defaultBodyLimit
	}

	router.Use(
		RequestIDMiddleware(),
		gin.Recovery(),
		AccessLogMiddleware(cfg.AccessLogFormat),
		corsMiddleware(cfg.CORSConfig),
		ErrorHandler(),
	)

	if len(cfg.StaticPrefix) > 0 && len(cfg.StaticDir) > 0 {
		router.Static(cfg.StaticPrefix, cfg.StaticDir)
	}

	// 文件上传最大内存
	router.MaxMultipartMemory = cfg.MultipartMemory

	// 请求body最大限制
	router.Use(bodyLimitMiddleware(cfg.BodyLimit))

	return router
}

func bodyLimitMiddleware(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

func defaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "X-Request-ID", "X-Response-Time"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func AccessLogMiddleware(format string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		var requestSize int64
		if c.Request.ContentLength > 0 {
			requestSize = c.Request.ContentLength
		}

		c.Next()

		latency := time.Since(start)
		requestID, _ := c.Get("request_id")
		if requestID == nil {
			requestID = "-"
		}

		fields := log.Fields{
			"status":       c.Writer.Status(),
			"method":       c.Request.Method,
			"path":         path,
			"client_ip":    c.ClientIP(),
			"host":         c.Request.Host,
			"request_id":   requestID,
			"user_agent":   c.Request.UserAgent(),
			"time":         time.Now().Format("2006-01-02 15:04:05"),
			"request_time": fmt.Sprintf("%.3fs", float64(latency.Microseconds())/1e6),
			"bytes_in":     requestSize,
			"bytes_out":    c.Writer.Size(),
		}

		if query != "" {
			fields["query"] = query
		}
		if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
			fields["real_ip"] = realIP
		}
		if referer := c.Request.Referer(); referer != "" {
			fields["referer"] = referer
		}
		if proto := c.Request.Proto; proto != "" {
			fields["protocol"] = proto
		}
		if len(c.Errors) > 0 {
			fields["error"] = c.Errors.String()
			fields["error_count"] = len(c.Errors)
		}

		if format == DefaultAccessLogFormat {
			statusCode := c.Writer.Status()
			logger := log.WithFields(fields)
			switch {
			case statusCode >= 500:
				logger.Error("server error")
			case statusCode >= 400:
				logger.Warn("client error")
			case statusCode >= 300:
				logger.Info("redirect")
			default:
				logger.Info("success")
			}
		} else {
			logMsg := format
			for k, v := range fields {
				placeholder := "${" + k + "}"
				logMsg = strings.ReplaceAll(logMsg, placeholder, fmt.Sprintf("%v", v))
			}
			log.Infoln(logMsg)
		}
	}
}

func corsMiddleware(config *CORSConfig) gin.HandlerFunc {
	if config == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	defaultConfig := defaultCORSConfig()
	if config.AllowOrigins == nil {
		config.AllowOrigins = defaultConfig.AllowOrigins
	}
	if config.AllowMethods == nil {
		config.AllowMethods = defaultConfig.AllowMethods
	}
	if config.AllowHeaders == nil {
		config.AllowHeaders = defaultConfig.AllowHeaders
	}
	if config.ExposeHeaders == nil {
		config.ExposeHeaders = defaultConfig.ExposeHeaders
	}
	if config.MaxAge == 0 {
		config.MaxAge = defaultConfig.MaxAge
	}

	// 处理 AllowOrigins 配置
	allowedOrigins := make(map[string]bool)
	wildcardSuffixes := make([]string, 0)
	for _, origin := range config.AllowOrigins {
		if origin == "*" {
			allowedOrigins["*"] = true
		} else if strings.HasPrefix(origin, "*.") {
			wildcardSuffixes = append(wildcardSuffixes, strings.TrimPrefix(origin, "*"))
		} else {
			allowedOrigins[origin] = true
		}
	}

	// 校验：AllowCredentials=true 时不能使用通配符 *
	if config.AllowCredentials && allowedOrigins["*"] {
		allowedOrigins["*"] = false
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allowed := false
		if allowedOrigins["*"] {
			allowed = true
		} else if allowedOrigins[origin] {
			allowed = true
		} else {
			// 严格匹配 *.domain.com 格式（通配符不能跨多个点）
			for _, suffix := range wildcardSuffixes {
				suffix := "." + suffix
				if strings.HasSuffix(origin, suffix) && len(origin) > len(suffix) && origin[len(origin)-len(suffix)-1] == '.' {
					allowed = true
					break
				}
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		if c.Request.Method == "OPTIONS" {
			c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
			c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
			c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
			c.Header("Access-Control-Max-Age", fmt.Sprintf("%.0f", config.MaxAge.Seconds()))
			if config.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		panicChan := make(chan any, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					panicChan <- r
				}
				close(done)
			}()
			c.Next()
		}()

		select {
		case <-done:
			select {
			case p := <-panicChan:
				panic(p)
			default:
			}
		case <-ctx.Done():
			c.AbortWithStatusJSON(http.StatusRequestTimeout, gin.H{
				"error": gin.H{
					"type":    "request_timeout",
					"message": "request timeout",
					"code":    http.StatusRequestTimeout,
				},
			})
		}
	}
}

type HTTPError struct {
	Type       string            `json:"type"`
	Message    string            `json:"message"`
	Code       int               `json:"code"`
	Details    []string          `json:"details,omitempty"`
	Validation map[string]string `json:"validation,omitempty"`
	Context    map[string]string `json:"context,omitempty"`
	RequestID  string            `json:"request_id"`
	Timestamp  string            `json:"timestamp"`
}

func NewHTTPError(c *gin.Context, errorType string, message string, code int) *HTTPError {
	return &HTTPError{
		Type:      errorType,
		Message:   message,
		Code:      code,
		RequestID: c.GetString("request_id"),
		Timestamp: time.Now().Format(time.RFC3339),
		Context: map[string]string{
			"method":       c.Request.Method,
			"path":         c.Request.URL.Path,
			"query":        c.Request.URL.String(),
			"client_ip":    c.ClientIP(),
			"user_agent":   c.Request.UserAgent(),
			"content_type": c.ContentType(),
		},
	}
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors[0]
		var apiError *HTTPError

		switch err.Type {
		case gin.ErrorTypeBind:
			apiError = NewHTTPError(c, "invalid_request", "invalid request parameters", http.StatusBadRequest)
			if validationErr, ok := err.Err.(validator.ValidationErrors); ok {
				apiError.Validation = make(map[string]string)
				for _, fieldErr := range validationErr {
					apiError.Validation[fieldErr.Field()] = fieldErr.Tag()
				}
			}
		case gin.ErrorTypeRender:
			apiError = NewHTTPError(c, "render_error", "response render failed", http.StatusInternalServerError)
		case gin.ErrorTypePrivate:
			apiError = NewHTTPError(c, "internal_error", "internal server error", http.StatusInternalServerError)
		case gin.ErrorTypePublic:
			switch {
			case strings.Contains(err.Error(), "not found"):
				apiError = NewHTTPError(c, "not_found", "resource not found", http.StatusNotFound)
			case strings.Contains(err.Error(), "unauthorized"):
				apiError = NewHTTPError(c, "unauthorized", "unauthorized access", http.StatusUnauthorized)
			case strings.Contains(err.Error(), "forbidden"):
				apiError = NewHTTPError(c, "forbidden", "access forbidden", http.StatusForbidden)
			case strings.Contains(err.Error(), "timeout"):
				apiError = NewHTTPError(c, "timeout", "request timeout", http.StatusRequestTimeout)
			case strings.Contains(err.Error(), "validation"):
				apiError = NewHTTPError(c, "validation_error", "validation failed", http.StatusUnprocessableEntity)
			default:
				apiError = NewHTTPError(c, "unknown_error", "unknown error", http.StatusInternalServerError)
			}
		default:
			apiError = NewHTTPError(c, "unknown_error", "unknown error", http.StatusInternalServerError)
		}

		log.WithFields(log.Fields{
			"error_type": apiError.Type,
			"status":     apiError.Code,
			"path":       apiError.Context["path"],
			"method":     apiError.Context["method"],
			"client_ip":  apiError.Context["client_ip"],
			"user_agent": apiError.Context["user_agent"],
			"request_id": apiError.RequestID,
		}).Error(err.Error())

		c.JSON(apiError.Code, gin.H{
			"error": apiError,
		})
	}
}

func NewServer(router *gin.Engine, cfg ServerConfig) (*http.Server, error) {
	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	// 尝试启动监听
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return nil, kiterrors.Wrap("HTTP_START", "failed to listen on "+cfg.Address, err)
	}

	// 关闭临时 listener，让 srv 接管
	listener.Close()

	errChan := make(chan error, 1)
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// 确认服务器已启动
	select {
	case err := <-errChan:
		return nil, kiterrors.Wrap("HTTP_START", "server error", err)
	case <-time.After(100 * time.Millisecond):
		return srv, nil
	}
}

func ShutdownServer(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}
	return nil
}
