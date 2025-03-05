package export

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Collector 通用数据收集器接口
type Collector interface {
	Name() string
	Collect(ch chan<- prometheus.Metric)
	Describe(ch chan<- *prometheus.Desc)
}

// BaseCollector 基础收集器实现
type BaseCollector struct {
	mtx       sync.Mutex
	Namespace string
	Subsystem string
	Metrics   map[string]*prometheus.Desc
}

// CreateMetric 创建带标签的指标描述符（线程安全）
func (b *BaseCollector) CreateMetric(name, help string, labels []string) *prometheus.Desc {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.Metrics == nil {
		b.Metrics = make(map[string]*prometheus.Desc)
	}

	desc := prometheus.NewDesc(
		prometheus.BuildFQName(b.Namespace, b.Subsystem, name),
		help,
		labels,
		nil,
	)
	b.Metrics[name] = desc
	return desc
}

// GetMetric 获取已注册的指标描述符（线程安全）
func (b *BaseCollector) GetMetric(name string) (*prometheus.Desc, bool) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	desc, exists := b.Metrics[name]
	return desc, exists
}
