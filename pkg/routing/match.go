package routing

import (
	"encoding/json"
	"sort"
)

type Rule struct {
	TeamID      string
	MatchLabels map[string]string
	Priority    int
}

func MatchTeam(rules []Rule, alertLabels map[string]string) (string, bool) {
	if len(rules) == 0 {
		return "", false
	}

	sorted := append([]Rule(nil), rules...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Priority == sorted[j].Priority {
			return sorted[i].TeamID < sorted[j].TeamID
		}
		return sorted[i].Priority > sorted[j].Priority
	})

	for _, rule := range sorted {
		if labelsMatch(rule.MatchLabels, alertLabels) {
			return rule.TeamID, true
		}
	}
	return "", false
}

func labelsMatch(required, alert map[string]string) bool {
	if len(required) == 0 {
		return false
	}
	for key, want := range required {
		got, ok := alert[key]
		if !ok || got != want {
			return false
		}
	}
	return true
}

func ParseMatchLabels(raw []byte) (map[string]string, error) {
	if len(raw) == 0 {
		return map[string]string{}, nil
	}
	var labels map[string]string
	if err := json.Unmarshal(raw, &labels); err != nil {
		return nil, err
	}
	if labels == nil {
		labels = map[string]string{}
	}
	return labels, nil
}
