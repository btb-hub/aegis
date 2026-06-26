package locale

import "fmt"

func Validate(locale string) error {
	if locale == "en" || locale == "ru" {
		return nil
	}
	return fmt.Errorf("invalid locale: %s", locale)
}

func Normalize(locale string) string {
	if locale == "ru" {
		return "ru"
	}
	return "en"
}
