package domain

import "testing"

func TestValidateObject(t *testing.T) {
	schema := map[string]any{"type": "object", "required": []any{"count"}, "properties": map[string]any{"count": map[string]any{"type": "integer"}, "label": map[string]any{"type": "string"}}}
	for _, tc := range []struct {
		name  string
		value map[string]any
		valid bool
	}{
		{"integer", map[string]any{"count": float64(2)}, true},
		{"fraction", map[string]any{"count": 2.5}, false},
		{"missing required", map[string]any{"label": "ok"}, false},
		{"wrong optional type", map[string]any{"count": float64(2), "label": false}, false},
		{"null", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateObject(schema, tc.value); (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}

func TestValidateSchemaRejectsUnsupportedKeywords(t *testing.T) {
	if err := ValidateSchema(map[string]any{"type": "object", "properties": map[string]any{}, "oneOf": []any{}}); err == nil {
		t.Fatal("unsupported keyword silently ignored")
	}
}
