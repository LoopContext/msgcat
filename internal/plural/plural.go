// Package plural provides CLDR plural form selection for a given language and count.
// Form names: "zero", "one", "two", "few", "many", "other".
package plural

import "strings"

// Form returns the CLDR plural form for the given language tag and count.
// Language tag is normalized to base (e.g. "en-US" -> "en"). Unknown languages default to "other".
func Form(lang string, count int) string {
	base := strings.ToLower(strings.TrimSpace(lang))
	if idx := strings.Index(base, "-"); idx > 0 {
		base = base[:idx]
	}
	if idx := strings.Index(base, "_"); idx > 0 {
		base = base[:idx]
	}
	n := count
	if n < 0 {
		n = -n
	}
	switch base {
	case "ar":
		return formArabic(n)
	case "ru", "uk", "be", "sr", "hr", "bs", "sh":
		return formRussian(n)
	case "pl":
		return formPolish(n)
	case "cy", "br", "ga", "gd", "gv", "kw", "mt", "sm", "ak":
		return formWelsh(n)
	case "he", "iw":
		return formHebrew(n)
	case "en", "es", "fr", "de", "it", "pt", "nl", "no", "sv", "da", "fi", "tr", "el", "ja", "ko", "zh", "th", "vi", "id", "hi":
		return formOneOther(n)
	default:
		return "other"
	}
}

func formOneOther(n int) string {
	if n == 1 {
		return "one"
	}
	return "other"
}

func formArabic(n int) string {
	if n == 0 {
		return "zero"
	}
	if n == 1 {
		return "one"
	}
	if n == 2 {
		return "two"
	}
	if n >= 3 && n <= 10 {
		return "few"
	}
	if n >= 11 && n <= 99 {
		return "many"
	}
	return "other"
}

func formRussian(n int) string {
	n10 := n % 10
	n100 := n % 100
	if n10 == 1 && n100 != 11 {
		return "one"
	}
	if n10 >= 2 && n10 <= 4 && (n100 < 12 || n100 > 14) {
		return "few"
	}
	if n10 == 0 || (n10 >= 5 && n10 <= 9) || (n100 >= 11 && n100 <= 14) {
		return "many"
	}
	return "other"
}

func formPolish(n int) string {
	if n == 1 {
		return "one"
	}
	n10 := n % 10
	n100 := n % 100
	if n10 >= 2 && n10 <= 4 && (n100 < 12 || n100 > 14) {
		return "few"
	}
	if n10 == 0 || (n10 >= 5 && n10 <= 9) || (n100 >= 12 && n100 <= 14) {
		return "many"
	}
	return "other"
}

func formWelsh(n int) string {
	if n == 0 {
		return "zero"
	}
	if n == 1 {
		return "one"
	}
	if n == 2 {
		return "two"
	}
	if n == 3 {
		return "few"
	}
	if n == 6 {
		return "many"
	}
	return "other"
}

func formHebrew(n int) string {
	if n == 1 {
		return "one"
	}
	if n == 2 {
		return "two"
	}
	if n >= 3 && n <= 10 {
		return "few"
	}
	if n >= 11 && n <= 99 {
		return "many"
	}
	return "other"
}
