package mask

import "strings"

func Phone(s string) string {
	r := []rune(s)
	if len(r) < 7 {
		return s
	}
	return string(r[:3]) + "****" + string(r[len(r)-4:])
}

func Email(s string) string {
	parts := strings.Split(s, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return s
	}
	local := []rune(parts[0])
	return string(local[:1]) + "***@" + parts[1]
}

func IDCard(s string) string {
	r := []rune(s)
	if len(r) < 10 {
		return s
	}
	return string(r[:6]) + "********" + string(r[len(r)-4:])
}

func BankCard(s string) string {
	r := []rune(s)
	if len(r) < 8 {
		return s
	}
	return string(r[:4]) + "********" + string(r[len(r)-4:])
}

func Name(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	if len(r) == 1 {
		return s
	}
	return string(r[:1]) + strings.Repeat("*", len(r)-1)
}
