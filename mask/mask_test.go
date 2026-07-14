package mask

import "testing"

func TestPhone(t *testing.T) {
	got := Phone("13812345678")
	if got != "138****5678" {
		t.Fatalf("expected masked phone, got %q", got)
	}
}

func TestEmail(t *testing.T) {
	got := Email("test@example.com")
	if got != "t***@example.com" {
		t.Fatalf("expected masked email, got %q", got)
	}
}

func TestIDCard(t *testing.T) {
	got := IDCard("110101199001011234")
	if got != "110101********1234" {
		t.Fatalf("expected masked id card, got %q", got)
	}
}

func TestBankCard(t *testing.T) {
	got := BankCard("6222021234567890")
	if got != "6222********7890" {
		t.Fatalf("expected masked bank card, got %q", got)
	}
}

func TestName(t *testing.T) {
	got := Name("张三")
	if got != "张*" {
		t.Fatalf("expected masked name, got %q", got)
	}
}

func TestInvalidOrShortInputReturnsOriginal(t *testing.T) {
	if got := Phone("12345"); got != "12345" {
		t.Fatalf("expected short phone unchanged, got %q", got)
	}
	if got := Email("invalid"); got != "invalid" {
		t.Fatalf("expected invalid email unchanged, got %q", got)
	}
	if got := Name(""); got != "" {
		t.Fatalf("expected empty name unchanged, got %q", got)
	}
}
