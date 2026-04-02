package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/tiamxu/kit/log"
)

type GinServerConfig struct {
	Address         string        `yaml:"address" json:"address"`
	KeepAlive       bool          `yaml:"keep_alive" json:"keep_alive"`
	ReadTimeout     time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout" json:"write_timeout"`
	AccessLogFormat string        `yaml:"access_log_format" json:"access_log_format"`
	StaticPrefix    string        `yaml:"static_prefix" json:"static_prefix"`
	StaticDir       string        `yaml:"static_dir" json:"static_dir"`
	BodyLimit       int64         `yaml:"body_limit" json:"body_limit"`
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

var DefaultAccessLogFormat = `${time} | ${status} | ${latency} | ${client_ip} | ${method} ${path} | ${request_id} | ${user_agent} | ${error}`

func NewGin(cfg GinServerConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	if len(cfg.AccessLogFormat) == 0 {
		cfg.AccessLogFormat = DefaultAccessLogFormat
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

	router.MaxMultipartMemory = cfg.BodyLimit

	return router
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
			"ip":           c.ClientIP(),
			"host":         c.Request.Host,
			"request_id":   requestID,
			"user_agent":   c.Request.UserAgent(),
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

	allowedOrigins := make(map[string]bool)
	for _, origin := range config.AllowOrigins {
		allowedOrigins[origin] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			c.Next()
			return
		}

		if allowedOrigins["*"] || allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			for allowedOrigin := range allowedOrigins {
				if strings.HasPrefix(allowedOrigin, "*") {
					domain := strings.TrimPrefix(allowedOrigin, "*")
					if strings.HasSuffix(origin, domain) {
						c.Header("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}
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
		panicChan := make(chan interface{}, 1)
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
					"message": "请求处理超时",
					"code":    http.StatusRequestTimeout,
				},
			})
		}
	}
}

type Error struct {
	Type       string            `json:"type"`
	Message    string            `json:"message"`
	Code       int               `json:"code"`
	Details    []string          `json:"details,omitempty"`
	Validation map[string]string `json:"validation,omitempty"`
	Context    map[string]string `json:"context,omitempty"`
	RequestID  string            `json:"request_id"`
	Timestamp  string            `json:"timestamp"`
}

func NewError(c *gin.Context, errorType string, message string, code int) *Error {
	return &Error{
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
		var apiError *Error

		switch err.Type {
		case gin.ErrorTypeBind:
			apiError = NewError(c, "invalid_request", "请求参数格式错误", http.StatusBadRequest)
			if validationErr, ok := err.Err.(validator.ValidationErrors); ok {
				apiError.Validation = make(map[string]string)
				for _, fieldErr := range validationErr {
					apiError.Validation[fieldErr.Field()] = fieldErr.Tag()
				}
			}
		case gin.ErrorTypeRender:
			apiError = NewError(c, "render_error", "响应渲染失败", http.StatusInternalServerError)
		case gin.ErrorTypePrivate:
			apiError = NewError(c, "internal_error", "服务器内部错误", http.StatusInternalServerError)
		case gin.ErrorTypePublic:
			switch {
			case strings.Contains(err.Error(), "not found"):
				apiError = NewError(c, "not_found", "请求的资源不存在", http.StatusNotFound)
			case strings.Contains(err.Error(), "unauthorized"):
				apiError = NewError(c, "unauthorized", "未授权的访问", http.StatusUnauthorized)
			case strings.Contains(err.Error(), "forbidden"):
				apiError = NewError(c, "forbidden", "禁止访问", http.StatusForbidden)
			case strings.Contains(err.Error(), "timeout"):
				apiError = NewError(c, "timeout", "请求超时", http.StatusRequestTimeout)
			case strings.Contains(err.Error(), "validation"):
				apiError = NewError(c, "validation_error", "数据验证失败", http.StatusUnprocessableEntity)
			default:
				apiError = NewError(c, "unknown_error", "未知错误", http.StatusInternalServerError)
			}
		default:
			apiError = NewError(c, "unknown_error", "未知错误", http.StatusInternalServerError)
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

func StartServer(router *gin.Engine, cfg GinServerConfig) (*http.Server, error) {
	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("server listen error: %v", err)
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return nil, err
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
