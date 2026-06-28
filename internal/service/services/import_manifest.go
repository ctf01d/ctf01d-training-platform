package services

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	serviceManifestYAML = ".ctf01d-service.yml"
	serviceManifestYML  = ".ctf01d-service.yaml"
)

var serviceManifestCandidates = []string{serviceManifestYAML, serviceManifestYML}

// manifestTrainingFieldCount is a capacity hint for the number of training
// metadata keys a manifest can contribute when merged.
const manifestTrainingFieldCount = 6

type ServiceManifest struct {
	ID          string
	ServiceName string
	ScriptPath  string
	ScriptWait  int
	RoundSleep  int
	Enabled     *bool
	Raw         map[string]any
}

func parseServiceManifest(data []byte) (*ServiceManifest, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("%s is empty", serviceManifestYAML)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode %s: %w", serviceManifestYAML, err)
	}

	section := raw
	var checkerSections []string
	checkerConfig := make(map[string]map[string]any)
	for key, value := range raw {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(key)), "checker-config-") {
			continue
		}
		nested, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: section %q must be an object", serviceManifestYAML, key)
		}
		checkerSections = append(checkerSections, key)
		checkerConfig[key] = nested
	}
	if len(checkerSections) > 1 {
		sort.Strings(checkerSections)
		return nil, fmt.Errorf("%s: multiple checker-config sections found: %s", serviceManifestYAML, strings.Join(checkerSections, ", "))
	}
	if len(checkerSections) == 1 {
		section = checkerConfig[checkerSections[0]]
	}

	manifest := &ServiceManifest{
		ID:          trimManifestString(section["id"]),
		ServiceName: trimManifestString(section["service_name"]),
		ScriptPath:  trimManifestString(section["script_path"]),
		ScriptWait:  manifestInt(section["script_wait_in_sec"]),
		RoundSleep:  manifestInt(section["time_sleep_between_run_scripts_in_sec"]),
		Raw:         raw,
	}
	if enabled, ok := manifestBool(section["enabled"]); ok {
		manifest.Enabled = &enabled
	}

	return manifest, nil
}

func mergeTrainingMetadata(training map[string]any, manifest *ServiceManifest) json.RawMessage {
	merged := make(map[string]any, len(training)+manifestTrainingFieldCount)
	for key, value := range training {
		merged[key] = value
	}

	if manifest != nil {
		if manifest.ServiceName != "" {
			if _, ok := merged["display_name"]; !ok {
				merged["display_name"] = manifest.ServiceName
			}
			merged["service_name"] = manifest.ServiceName
		}
		if manifest.ID != "" {
			merged["checker_id"] = manifest.ID
		}
		if manifest.ScriptPath != "" {
			merged["script_rel"] = manifest.ScriptPath
		}
		if manifest.ScriptWait > 0 {
			merged["script_wait"] = manifest.ScriptWait
		}
		if manifest.RoundSleep > 0 {
			merged["round_sleep"] = manifest.RoundSleep
		}
		if manifest.Enabled != nil {
			merged["enabled"] = *manifest.Enabled
		}
		if manifest.Raw != nil {
			merged["ctf01d_service"] = manifest.Raw
		}
	}

	if len(merged) == 0 {
		return json.RawMessage("{}")
	}

	data, err := json.Marshal(merged)
	if err != nil {
		return json.RawMessage("{}")
	}
	return data
}

func trimManifestString(value any) string {
	s, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func manifestInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return n
		}
	}
	return 0
}

func manifestBool(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "yes", "on", "1":
			return true, true
		case "false", "no", "off", "0":
			return false, true
		}
	}
	return false, false
}
