package page

const (
	DefaultPage        = 1
	DefaultPageSize    = 20
	DefaultMaxPageSize = 100
)

type Config struct {
	DefaultPage     int
	DefaultPageSize int
	MaxPageSize     int
}

type Params struct {
	Page     int
	PageSize int
}

func Normalize(page, pageSize int) Params {
	return NormalizeWithConfig(page, pageSize, Config{
		DefaultPage:     DefaultPage,
		DefaultPageSize: DefaultPageSize,
		MaxPageSize:     DefaultMaxPageSize,
	})
}

func NormalizeWithConfig(page, pageSize int, cfg Config) Params {
	cfg = normalizeConfig(cfg)
	if page <= 0 {
		page = cfg.DefaultPage
	}
	if pageSize <= 0 {
		pageSize = cfg.DefaultPageSize
	}
	if pageSize > cfg.MaxPageSize {
		pageSize = cfg.MaxPageSize
	}
	return Params{Page: page, PageSize: pageSize}
}

func (p Params) Limit() int {
	if p.PageSize <= 0 {
		return DefaultPageSize
	}
	return p.PageSize
}

func (p Params) Offset() int {
	page := p.Page
	if page <= 0 {
		page = DefaultPage
	}
	return (page - 1) * p.Limit()
}

func normalizeConfig(cfg Config) Config {
	if cfg.DefaultPage <= 0 {
		cfg.DefaultPage = DefaultPage
	}
	if cfg.DefaultPageSize <= 0 {
		cfg.DefaultPageSize = DefaultPageSize
	}
	if cfg.MaxPageSize <= 0 {
		cfg.MaxPageSize = DefaultMaxPageSize
	}
	if cfg.DefaultPageSize > cfg.MaxPageSize {
		cfg.DefaultPageSize = cfg.MaxPageSize
	}
	return cfg
}
