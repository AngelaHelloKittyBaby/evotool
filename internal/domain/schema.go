package domain

import (
	"encoding/json"
	"fmt"
	"math"
)

// ValidateSchema checks the deliberately small MVP JSON schema subset.
// Only object schemas with primitive properties and required names are supported.
func ValidateSchema(schema map[string]any) error {
	if schema["type"] != "object" {
		return fmt.Errorf("schema type must be object")
	}
	for key := range schema {
		if key != "type" && key != "properties" && key != "required" && key != "description" {
			return fmt.Errorf("unsupported schema keyword %s", key)
		}
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return fmt.Errorf("schema properties required")
	}
	for name, value := range properties {
		p, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid property %s", name)
		}
		for key := range p {
			if key != "type" && key != "description" {
				return fmt.Errorf("unsupported property keyword %s", key)
			}
		}
		switch p["type"] {
		case "string", "number", "integer", "boolean":
		default:
			return fmt.Errorf("unsupported property type for %s", name)
		}
	}
	data, err := json.Marshal(schema["required"])
	if err != nil {
		return err
	}
	var required []string
	if err = json.Unmarshal(data, &required); err != nil {
		return fmt.Errorf("required must be string array")
	}
	for _, name := range required {
		if _, ok := properties[name]; !ok {
			return fmt.Errorf("required property %s is undefined", name)
		}
	}
	return nil
}

// ValidateObject enforces required properties and primitive types on JSON data.
func ValidateObject(schema map[string]any, value map[string]any) error {
	if err := ValidateSchema(schema); err != nil {
		return err
	}
	if value == nil {
		return fmt.Errorf("JSON object required")
	}
	data, _ := json.Marshal(schema["required"])
	var required []string
	_ = json.Unmarshal(data, &required)
	for _, name := range required {
		if _, ok := value[name]; !ok {
			return fmt.Errorf("missing required property %s", name)
		}
	}
	properties := schema["properties"].(map[string]any)
	for name, p := range properties {
		v, exists := value[name]
		if !exists {
			continue
		}
		valid := false
		switch p.(map[string]any)["type"] {
		case "string":
			_, valid = v.(string)
		case "boolean":
			_, valid = v.(bool)
		case "number":
			_, valid = v.(float64)
		case "integer":
			n, ok := v.(float64)
			valid = ok && math.Trunc(n) == n
		}
		if !valid {
			return fmt.Errorf("invalid type for property %s", name)
		}
	}
	return nil
}
