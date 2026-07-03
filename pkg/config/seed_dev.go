package config

import "fmt"

// SeedDevAllowed returns nil when dev user seeding is permitted.
// Allowed when PUBLIC_URL points at localhost, or when SEED_DEV=1 (still requires PUBLIC_URL to be set).
func SeedDevAllowed(publicURL string) error {
	if publicURL == "" {
		return fmt.Errorf("PUBLIC_URL is required to run dev seeds")
	}
	if parseBoolEnv("SEED_DEV") {
		return nil
	}
	return validateDevAuthHost(publicURL)
}
