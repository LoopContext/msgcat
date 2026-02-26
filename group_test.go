package msgcat

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestOptionalGroup_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    OptionalGroup
		wantErr bool
	}{
		{"string", `"api"`, "api", false},
		{"int", `0`, "0", false},
		{"int_one", `1`, "1", false},
		{"empty", `""`, "", false},
		{"null", `null`, "", false},
		{"invalid", `true`, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var g OptionalGroup
			err := yaml.Unmarshal([]byte(tt.yaml), &g)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && g != tt.want {
				t.Errorf("UnmarshalYAML() got = %q, want %q", g, tt.want)
			}
		})
	}
}

func TestOptionalGroup_MarshalYAML(t *testing.T) {
	tests := []struct {
		name string
		g    OptionalGroup
		// After marshal+unmarshal we expect the same value
		roundTripInt bool
	}{
		{"numeric", "0", true},
		{"numeric_nonzero", "42", true},
		{"string", "api", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tt.g.MarshalYAML()
			if err != nil {
				t.Fatalf("MarshalYAML() error = %v", err)
			}
			if out == nil {
				if tt.g != "" {
					t.Errorf("MarshalYAML() returned nil for non-empty %q", tt.g)
				}
				return
			}
			buf, err := yaml.Marshal(out)
			if err != nil {
				t.Fatalf("yaml.Marshal: %v", err)
			}
			var g2 OptionalGroup
			if err := yaml.Unmarshal(buf, &g2); err != nil {
				t.Fatalf("round-trip unmarshal: %v", err)
			}
			if g2 != tt.g {
				t.Errorf("round-trip got %q, want %q", g2, tt.g)
			}
		})
	}
}

func TestMessages_WithGroup(t *testing.T) {
	yamlContent := `
group: api
default:
  short: Unexpected error
  long: Unexpected message
set:
  greeting.hello:
    short: Hello
    long: Hello there
`
	var m Messages
	if err := yaml.Unmarshal([]byte(yamlContent), &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if m.Group != "api" {
		t.Errorf("Group = %q, want api", m.Group)
	}
	if len(m.Set) != 1 {
		t.Errorf("Set length = %d, want 1", len(m.Set))
	}
	// Round-trip: marshal and unmarshal
	out, err := yaml.Marshal(&m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var m2 Messages
	if err := yaml.Unmarshal(out, &m2); err != nil {
		t.Fatalf("Round-trip Unmarshal: %v", err)
	}
	if m2.Group != m.Group {
		t.Errorf("Round-trip Group = %q, want %q", m2.Group, m.Group)
	}
}

func TestMessages_WithGroupInt(t *testing.T) {
	yamlContent := `
group: 0
default:
  short: Err
  long: Error
set:
  x:
    short: x
    long: x
`
	var m Messages
	if err := yaml.Unmarshal([]byte(yamlContent), &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if m.Group != "0" {
		t.Errorf("Group = %q, want 0", m.Group)
	}
	out, err := yaml.Marshal(&m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// Should emit as int (numeric) for readability
	if len(out) == 0 {
		t.Fatal("Marshal produced empty output")
	}
	var m2 Messages
	if err := yaml.Unmarshal(out, &m2); err != nil {
		t.Fatalf("Round-trip Unmarshal: %v", err)
	}
	if m2.Group != "0" {
		t.Errorf("Round-trip Group = %q", m2.Group)
	}
}
