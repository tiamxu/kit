package page

import "testing"

func TestNormalizeAppliesDefaults(t *testing.T) {
	p := Normalize(0, 0)

	if p.Page != DefaultPage {
		t.Fatalf("expected page %d, got %d", DefaultPage, p.Page)
	}
	if p.PageSize != DefaultPageSize {
		t.Fatalf("expected page size %d, got %d", DefaultPageSize, p.PageSize)
	}
}

func TestNormalizeCapsPageSize(t *testing.T) {
	p := Normalize(1, 1000)

	if p.PageSize != DefaultMaxPageSize {
		t.Fatalf("expected page size %d, got %d", DefaultMaxPageSize, p.PageSize)
	}
}

func TestParamsLimitAndOffset(t *testing.T) {
	p := Normalize(3, 20)

	if p.Limit() != 20 {
		t.Fatalf("expected limit 20, got %d", p.Limit())
	}
	if p.Offset() != 40 {
		t.Fatalf("expected offset 40, got %d", p.Offset())
	}
}

func TestNormalizeWithConfig(t *testing.T) {
	p := NormalizeWithConfig(-1, 500, Config{
		DefaultPage:     2,
		DefaultPageSize: 30,
		MaxPageSize:     50,
	})

	if p.Page != 2 {
		t.Fatalf("expected page 2, got %d", p.Page)
	}
	if p.PageSize != 50 {
		t.Fatalf("expected page size 50, got %d", p.PageSize)
	}
}
