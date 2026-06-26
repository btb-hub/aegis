package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	catalogs   = map[string]map[string]string{}
	catalogsMu sync.RWMutex
)

func LoadMessages(dir string) error {
	catalogsMu.Lock()
	defer catalogsMu.Unlock()

	for _, locale := range []string{"en", "ru"} {
		path := filepath.Join(dir, locale+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		var messages map[string]string
		if err := json.Unmarshal(data, &messages); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		catalogs[locale] = messages
	}
	return nil
}

func T(locale, key string, vars map[string]string) string {
	catalogsMu.RLock()
	defer catalogsMu.RUnlock()

	if locale != "en" && locale != "ru" {
		locale = "en"
	}
	msg, ok := catalogs[locale][key]
	if !ok {
		msg = catalogs["en"][key]
	}
	if msg == "" {
		return key
	}
	for k, v := range vars {
		msg = strings.ReplaceAll(msg, "{{"+k+"}}", v)
	}
	return msg
}

func ValidateParity(dir string) error {
	enPath := filepath.Join(dir, "en.json")
	ruPath := filepath.Join(dir, "ru.json")

	enData, err := os.ReadFile(enPath)
	if err != nil {
		return err
	}
	ruData, err := os.ReadFile(ruPath)
	if err != nil {
		return err
	}

	var en, ru map[string]string
	if err := json.Unmarshal(enData, &en); err != nil {
		return err
	}
	if err := json.Unmarshal(ruData, &ru); err != nil {
		return err
	}

	var missing []string
	for key := range en {
		if _, ok := ru[key]; !ok {
			missing = append(missing, "ru:"+key)
		}
	}
	for key := range ru {
		if _, ok := en[key]; !ok {
			missing = append(missing, "en:"+key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("locale key mismatch: %s", strings.Join(missing, ", "))
	}
	return nil
}

func ResetForTests() {
	catalogsMu.Lock()
	defer catalogsMu.Unlock()
	catalogs = map[string]map[string]string{}
}
