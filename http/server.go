package httpkit

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tiamxu/kit/log"
)

type GinServerConfig struct {
	// Address 服务监听端口
	Address string `yaml:"address" default:":8803"`
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
}

var DefaultAccessLogFormat = `[GIN] ${time} | ${status} | ${latency} | ${client_ip} | ${method} ${path} ${error}`

func NewGin(cfg GinServerConfig) *gin.Engine {
	// 设置gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建gin实例
	router := gin.New()

	// 添加恢复中间件
	router.Use(gin.Recovery())

	// 添加日志中间件
	if len(cfg.AccessLogFormat) == 0 {
		cfg.AccessLogFormat = DefaultAccessLogFormat
	}
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: logFormatter(cfg.AccessLogFormat),
		Output:    log.DefaultLogger().Writer(),
	}))

	// 静态文件服务
	if len(cfg.StaticPrefix) > 0 && len(cfg.StaticDir) > 0 {
		router.Static(cfg.StaticPrefix, cfg.StaticDir)
	}

	// 设置body大小限制
	router.MaxMultipartMemory = cfg.BodyLimit

	return router
}

func logFormatter(format string) gin.LogFormatter {
	return func(param gin.LogFormatterParams) string {
		return format
	}
}

// ErrorHandler 错误处理
func ErrorHandler(c *gin.Context) {
	c.Next()

	// 处理错误
	if len(c.Errors) > 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"errors": c.Errors.Errors(),
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
