package config

import (
	"fmt"
	"strings"
)

// IntegrationSpec is one entry under the installer `integrations` list in
// config.yaml. The Integration field is a stable machine id (e.g. quay, github)
// used by Helm templates; Name is a human-readable label.
type IntegrationSpec struct {
	Integration string                 `yaml:"integration,omitempty"`
	Name        string                 `yaml:"name"`
	Properties  map[string]interface{} `yaml:"properties,omitempty"`
}

// Validate checks a single integration entry.
func (i IntegrationSpec) Validate() error {
	if strings.TrimSpace(i.Name) == "" {
		return fmt.Errorf("%w: integration entry missing name", ErrInvalidConfig)
	}
	return nil
}
