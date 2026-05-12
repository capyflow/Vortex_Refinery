package plugin

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// Validator validates plugin configuration against JSON Schema
type Validator struct{}

// NewValidator creates a new configuration validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates a plugin's config against its schema
// meta.ConfigSchema should be a valid JSON Schema document
func (v *Validator) Validate(meta *PluginMetadata, config json.RawMessage) error {
	if meta.ConfigSchema == nil || len(meta.ConfigSchema) == 0 {
		return nil // No schema defined, skip validation
	}

	// Convert our map-based schema to gojsonschema format
	schemaLoader, err := v.mapToJSONSchema(meta.ConfigSchema, meta.Name)
	if err != nil {
		return fmt.Errorf("invalid config schema for plugin %s: %w", meta.Name, err)
	}

	configLoader := gojsonschema.NewBytesLoader(config)

	result, err := gojsonschema.Validate(schemaLoader, configLoader)
	if err != nil {
		return fmt.Errorf("config validation error for plugin %s: %w", meta.Name, err)
	}

	if !result.Valid() {
		errMsg := fmt.Sprintf("config validation failed for plugin %s:\n", meta.Name)
		for _, err := range result.Errors() {
			errMsg += fmt.Sprintf("  - %s: %s\n", err.Field(), err.Description())
		}
		return fmt.Errorf(errMsg)
	}

	return nil
}

// mapToJSONSchema converts a map-style config schema to a proper JSON Schema document
func (v *Validator) mapToJSONSchema(configSchema map[string]interface{}, pluginName string) (gojsonschema.JSONLoader, error) {
	// Build a proper JSON Schema structure
	properties := make(map[string]interface{})
	required := []string{}

	for field, def := range configSchema {
		if m, ok := def.(map[string]interface{}); ok {
			props := make(map[string]interface{})
			for k, val := range m {
				props[k] = val
			}
			properties[field] = map[string]interface{}{
				"type": getType(m),
			}
			if isRequired(m) {
				required = append(required, field)
			}
		}
	}

	schemaDoc := map[string]interface{}{
		"$schema":    "http://json-schema.org/draft-07/schema#",
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schemaDoc["required"] = required
	}

	schemaBytes, err := json.Marshal(schemaDoc)
	if err != nil {
		return nil, err
	}

	return gojsonschema.NewBytesLoader(schemaBytes), nil
}

// getType extracts the type field from a config schema field definition
func getType(fieldDef map[string]interface{}) string {
	if t, ok := fieldDef["type"].(string); ok {
		return t
	}
	return "string" // default
}

// isRequired checks if a field is marked as required
func isRequired(fieldDef map[string]interface{}) bool {
	if req, ok := fieldDef["required"]; ok {
		if b, ok := req.(bool); ok {
			return b
		}
	}
	return false
}

// PluginConfigValidator is an optional interface plugins can implement
// for custom config validation logic
type PluginConfigValidator interface {
	ValidateConfig(config json.RawMessage) error
}
