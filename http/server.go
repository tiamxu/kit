package httpkit

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
	"golang.org/x/time/rate"
)

type GinServerConfig struct {
	// Address 服务监听端口
	Address string `yaml:"address" default:":8803"`
	// KeepAlive
	KeepAlive bool `yaml:"keep_alive" default:"true"`
	// ReadTimeout 读取超时
	ReadTimeout time.Duration `yaml:"read_timeout" default:"30s"`
	// WriteTimeout 写入超时
	WriteTimeout time.Duration `yaml:"write_timeout" default:"30s"`
	// AccessLogFormat 访问日志格式
	AccessLogFormat string `yaml:"access_log_format"`
	// StaticPrefix 静态路径前缀
	StaticPrefix string `yaml:"static_prefix"`
	// StaticDir 静态文件目录
	StaticDir string `yaml:"static_dir"`
	// BodyLimit body大小限制
	BodyLimit int64 `yaml:"body_limit" default:"10485760"` // 10MB
	// CORSConfig 跨域配置
	CORSConfig *CORSConfig `yaml:"cors"`
}

// CORSConfig 跨域配置
type CORSConfig struct {
	AllowOrigins     []string      `yaml:"allow_origins"`
	AllowMethods     []string      `yaml:"allow_methods"`
	AllowHeaders     []string      `yaml:"allow_headers"`
	ExposeHeaders    []string      `yaml:"expose_headers"`
	AllowCredentials bool          `yaml:"allow_credentials"`
	MaxAge           time.Duration `yaml:"max_age"`
}

var DefaultAccessLogFormat = `${time} | ${status} | ${latency} | ${client_ip} | ${method} ${path} | ${request_id} | ${user_agent} | ${error}`

// 新的日志格式示例：
// 2025-01-24T17:47:04+08:00 | 200 | 15ms | 192.168.1.1 | GET /api/users | abc123 | Mozilla/5.0 | -
func NewGin(cfg GinServerConfig) *gin.Engine {
	// 设置gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建gin实例
	router := gin.New()

	// 添加中间件，注意顺序
	if len(cfg.AccessLogFormat) == 0 {
		cfg.AccessLogFormat = DefaultAccessLogFormat
	}
	router.Use(
		RequestIDMiddleware(),                    // 请求ID中间件放在最前面
		gin.Recovery(),                           // 恢复中间件
		AccessLogMiddleware(cfg.AccessLogFormat), // 访问日志中间件
	)
	router.Use(RequestIDMiddleware())

	// 静态文件服务
	if len(cfg.StaticPrefix) > 0 && len(cfg.StaticDir) > 0 {
		router.Static(cfg.StaticPrefix, cfg.StaticDir)
	}

	// 设置body大小限制
	router.MaxMultipartMemory = cfg.BodyLimit

	// 添加CORS中间件
	router.Use(corsMiddleware(cfg.CORSConfig))

	// 添加错误处理中间件
	router.Use(ErrorHandler())
	// 在业务代码中抛出错误： c.Error(err).SetType(gin.ErrorTypePublic)

	return router
}

// defaultCORSConfig 返回默认CORS配置
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

// RequestIDMiddleware 生成和传递请求ID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取请求ID，如果没有则生成新的
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 设置请求ID到上下文和响应头
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// AccessLogMiddleware 访问日志中间件
func AccessLogMiddleware(format string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 获取请求体大小
		var requestSize int64
		if c.Request.ContentLength > 0 {
			requestSize = c.Request.ContentLength
		}

		// 处理请求
		c.Next()

		// 计算处理时间
		end := time.Now()
		latency := end.Sub(start)

		// 获取请求ID
		requestID, _ := c.Get("request_id")
		if requestID == nil {
			requestID = "-"
		}

		// 构建日志字段
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

		// 添加可选字段
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
		// 添加错误信息
		if len(c.Errors) > 0 {
			fields["error"] = c.Errors.String()
			fields["error_count"] = len(c.Errors)
		} else {
			// 根据状态码使用不同的日志级别
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
		}

		// 如果指定了自定义格式，则额外输出格式化日志
		if format == DefaultAccessLogFormat {
			log.WithFields(fields).Info("access_log")
		} else {
			logMsg := format
			for k, v := range fields {
				placeholder := "${" + k + "}"
				logMsg = strings.ReplaceAll(logMsg, placeholder, fmt.Sprintf("%v", v))
				// logMsg = strings.ReplaceAll(logMsg, "${time}", end.Format(time.RFC3339))

			}
			// 使用 Info 级别输出格式化日志
			log.Infoln(logMsg)
		}
	}
}

// corsMiddleware CORS中间件
func corsMiddleware(config *CORSConfig) gin.HandlerFunc {
	// 如果配置为空，不应用CORS
	if config == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// 合并默认配置
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

	// 将允许的域名转换为map提高查找效率
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

		// 检查域名是否允许
		if allowedOrigins["*"] || allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			// 检查通配符匹配
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

		// 处理预检请求
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

// RateLimitMiddleware 请求速率限制中间件
func RateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Every(window/time.Duration(limit)), limit)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"type":    "rate_limit_exceeded",
					"message": "请求过于频繁，请稍后再试",
					"code":    http.StatusTooManyRequests,
				},
			})
			return
		}
		c.Next()
	}
}

// TimeoutMiddleware 请求超时处理中间件
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			return
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

// Error 自定义错误结构
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

// NewError 创建新的错误响应
func NewError(c *gin.Context, errorType string, message string, code int) *Error {
	return &Error{
		Type:      errorType,
		Message:   message,
		Code:      code,
		RequestID: c.GetHeader("X-Request-ID"),
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

// ErrorHandler 统一错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		// 获取第一个错误
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

		// 记录错误日志
		log.WithFields(log.Fields{
			"error_type": apiError.Type,
			"status":     apiError.Code,
			"path":       apiError.Context["path"],
			"method":     apiError.Context["method"],
			"client_ip":  apiError.Context["client_ip"],
			"user_agent": apiError.Context["user_agent"],
			"request_id": apiError.RequestID,
		}).Error(err.Error())

		// 返回错误响应
		c.JSON(apiError.Code, gin.H{
			"error": apiError,
		})
	}
}

// StartServer 启动服务
func StartServer(router *gin.Engine, cfg GinServerConfig) *http.Server {
	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	return srv
}

// ShutdownServer 优雅关闭服务
func ShutdownServer(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalln("Server forced to shutdown:", err)
	}
}
