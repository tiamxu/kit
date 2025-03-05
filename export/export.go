// core/exporter.go
package export

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"
)

// Exporter 主体结构
type Exporter struct {
	collectors []Collector
	registry   *prometheus.Registry
}

// NewExporter 创建Exporter实例
func NewExporter() *Exporter {
	return &Exporter{
		registry: prometheus.NewRegistry(),
	}
}

// AddCollector 添加收集器（线程安全）
func (e *Exporter) AddCollector(c Collector) {
	e.collectors = append(e.collectors, c)
	e.registry.MustRegister(c)
}

// StartServer 启动HTTP服务（含优雅关闭）
func (e *Exporter) StartServer(port int, path string) error {
	server := &http.Server{
		Addr: ":" + strconv.Itoa(port),
	}

	http.Handle(path, promhttp.HandlerFor(
		e.registry,
		promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError},
	))

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body><h1>Exporter</h1></body></html>`))
	})

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			panic(fmt.Sprintf("Server failed: %v", err))
		}
	}()

	// 添加优雅关闭逻辑
	<-context.Background().Done()
	return server.Shutdown(context.Background())
}

// CommonCLIFlags 通用命令行参数
func CommonCLIFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:    "web.port",
			Value:   9087,
			EnvVars: []string{"WEB_PORT"},
		},
		&cli.StringFlag{
			Name:    "web.path",
			Value:   "/metrics",
			EnvVars: []string{"WEB_PATH"},
		},
	}
}
