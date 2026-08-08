package monitor

import "strings"

type Incident string

const (
	IncidentNone                Incident = "none"
	IncidentServerOverloaded    Incident = "server_overloaded"
	IncidentFiveHourQuota       Incident = "five_hour_quota_exhausted"
	IncidentWeeklyQuota         Incident = "weekly_quota_exhausted"
	IncidentQuotaExhausted      Incident = "quota_exhausted"
	IncidentResourceAmbiguous   Incident = "resource_exhausted_ambiguous"
)

// Classify groups adjacent log lines from one request into a conservative
// incident. Server overload always wins over generic RESOURCE_EXHAUSTED text so
// callers never rotate accounts for a temporary provider-side failure.
func Classify(text string) Incident {
	value := strings.ToLower(text)
	if strings.Contains(value, "model api is currently overloaded") ||
		strings.Contains(value, "server overloaded") ||
		strings.Contains(value, "temporarily overloaded") {
		return IncidentServerOverloaded
	}
	resourceExhausted := strings.Contains(value, "resource_exhausted") ||
		strings.Contains(value, "resource has been exhausted") ||
		strings.Contains(value, "code 429") ||
		strings.Contains(value, "http 429")
	if !resourceExhausted {
		return IncidentNone
	}
	if strings.Contains(value, "weekly") &&
		(strings.Contains(value, "quota") || strings.Contains(value, "limit")) {
		return IncidentWeeklyQuota
	}
	if strings.Contains(value, "five hour") ||
		strings.Contains(value, "5 hour") ||
		strings.Contains(value, "5-hour") {
		return IncidentFiveHourQuota
	}
	if strings.Contains(value, "quota exceeded") ||
		strings.Contains(value, "quota has been exhausted") ||
		strings.Contains(value, "exceeded your current quota") ||
		strings.Contains(value, "account quota") {
		return IncidentQuotaExhausted
	}
	return IncidentResourceAmbiguous
}

func (i Incident) AllowsAccountSwitch() bool {
	switch i {
	case IncidentFiveHourQuota, IncidentWeeklyQuota, IncidentQuotaExhausted:
		return true
	default:
		return false
	}
}
