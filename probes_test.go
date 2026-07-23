package cloud

import (
	"encoding/json"
	"testing"
)

func TestUpdateServiceInputReadinessProbeJSON(t *testing.T) {
	in := UpdateServiceInput{
		Action: "readiness_probe",
		Probe: &ServiceProbe{
			Path:                "/ready",
			Port:                8080,
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
			TimeoutSeconds:      2,
			FailureThreshold:    3,
			SuccessThreshold:    1,
		},
	}

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["action"] != "readiness_probe" {
		t.Fatalf("action = %v", got["action"])
	}
	probe, ok := got["probe"].(map[string]any)
	if !ok {
		t.Fatalf("probe = %#v", got["probe"])
	}
	if probe["path"] != "/ready" || probe["port"] != float64(8080) {
		t.Fatalf("probe = %#v", probe)
	}
}

func TestUpdateServiceInputDeleteReadinessProbeOmitsProbe(t *testing.T) {
	data, err := json.Marshal(UpdateServiceInput{Action: "delete_readiness_probe"})
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if _, exists := got["probe"]; exists {
		t.Fatalf("delete payload unexpectedly contains probe: %s", data)
	}
}
