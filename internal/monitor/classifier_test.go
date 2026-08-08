package monitor

import "testing"

func TestClassifyOverloadWinsOverResourceExhausted(t *testing.T) {
	incident := Classify("RESOURCE_EXHAUSTED code 429: The model API is currently overloaded")
	if incident != IncidentServerOverloaded {
		t.Fatalf("incident = %q", incident)
	}
	if incident.AllowsAccountSwitch() {
		t.Fatal("server overload must not allow account switching")
	}
}

func TestClassifyWeeklyQuota(t *testing.T) {
	incident := Classify("RESOURCE_EXHAUSTED: weekly quota limit reached")
	if incident != IncidentWeeklyQuota || !incident.AllowsAccountSwitch() {
		t.Fatalf("incident = %q", incident)
	}
}

func TestClassifyAmbiguousResourceExhaustion(t *testing.T) {
	incident := Classify("RESOURCE_EXHAUSTED code 429: resource has been exhausted")
	if incident != IncidentResourceAmbiguous {
		t.Fatalf("incident = %q", incident)
	}
	if incident.AllowsAccountSwitch() {
		t.Fatal("ambiguous resource exhaustion must not allow account switching")
	}
}
