// Package config provides utilities for loading routing slips from
// JSON configuration, environment variables, or plain Go maps.
package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/raywall/routing-slip-pattern/slip"
)

// WorkflowConfig is the top-level configuration structure.
// It can be loaded from a JSON file or constructed in code.
type WorkflowConfig struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	ErrorPolicy string       `json:"error_policy"` // "stop" | "continue" | "skip"
	Steps       []StepConfig `json:"steps"`
}

// StepConfig maps to a single StepDef in the routing slip.
type StepConfig struct {
	Name    string         `json:"name"`
	Enabled bool           `json:"enabled"` // false → step is skipped at load time
	Params  map[string]any `json:"params"`
}

// ToSlip converts the WorkflowConfig into a []slip.StepDef ready for attachment.
func (wc *WorkflowConfig) ToSlip() []slip.StepDef {
	var steps []slip.StepDef
	for _, s := range wc.Steps {
		if !s.Enabled {
			continue
		}
		steps = append(steps, slip.StepDef{
			Name:   s.Name,
			Params: s.Params,
		})
	}
	return steps
}

// ErrorPolicyFromString converts a string to a slip.ErrorPolicy.
func ErrorPolicyFromString(s string) (slip.ErrorPolicy, error) {
	switch s {
	case "stop", "":
		return slip.StopOnError, nil
	case "continue":
		return slip.ContinueOnError, nil
	case "skip":
		return slip.SkipOnError, nil
	default:
		return slip.StopOnError, fmt.Errorf("unknown error policy: %q", s)
	}
}

// LoadFromFile parses a JSON workflow configuration from the given file path.
func LoadFromFile(path string) (*WorkflowConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: cannot read %q: %w", path, err)
	}
	return LoadFromJSON(data)
}

// LoadFromJSON parses a WorkflowConfig from raw JSON bytes.
func LoadFromJSON(data []byte) (*WorkflowConfig, error) {
	var cfg WorkflowConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: invalid JSON: %w", err)
	}
	return &cfg, nil
}

// MustLoadFromFile is like LoadFromFile but panics on error.
func MustLoadFromFile(path string) *WorkflowConfig {
	cfg, err := LoadFromFile(path)
	if err != nil {
		panic(err)
	}
	return cfg
}
