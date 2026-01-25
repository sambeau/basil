package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSecretStringUnmarshal(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		wantValue  string
		wantSecret bool
	}{
		{
			name:       "plain string",
			yaml:       "key: plainvalue",
			wantValue:  "plainvalue",
			wantSecret: false,
		},
		{
			name:       "secret string",
			yaml:       "key: !secret mysecret",
			wantValue:  "mysecret",
			wantSecret: true,
		},
		{
			name:       "secret auto",
			yaml:       "key: !secret auto",
			wantValue:  "auto",
			wantSecret: true,
		},
		{
			name:       "empty secret",
			yaml:       "key: !secret \"\"",
			wantValue:  "",
			wantSecret: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Key SecretString `yaml:"key"`
			}

			if err := yaml.Unmarshal([]byte(tt.yaml), &result); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			if result.Key.Value() != tt.wantValue {
				t.Errorf("value = %q, want %q", result.Key.Value(), tt.wantValue)
			}

			if result.Key.IsSecret() != tt.wantSecret {
				t.Errorf("isSecret = %v, want %v", result.Key.IsSecret(), tt.wantSecret)
			}
		})
	}
}

func TestSecretStringString(t *testing.T) {
	tests := []struct {
		name string
		s    SecretString
		want string
	}{
		{
			name: "secret value",
			s:    NewSecretString("mysecret"),
			want: "[hidden]",
		},
		{
			name: "empty secret",
			s:    NewSecretString(""),
			want: "",
		},
		{
			name: "plain value",
			s:    SecretString{value: "plain", isSecret: false},
			want: "plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSecretStringIsAuto(t *testing.T) {
	tests := []struct {
		name string
		s    SecretString
		want bool
	}{
		{
			name: "auto value",
			s:    NewSecretString("auto"),
			want: true,
		},
		{
			name: "not auto",
			s:    NewSecretString("other"),
			want: false,
		},
		{
			name: "empty",
			s:    NewSecretString(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.IsAuto(); got != tt.want {
				t.Errorf("IsAuto() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecretStringMarshal(t *testing.T) {
	tests := []struct {
		name      string
		s         SecretString
		wantTag   bool
		wantValue string
	}{
		{
			name:      "secret value",
			s:         NewSecretString("mysecret"),
			wantTag:   true,
			wantValue: "mysecret",
		},
		{
			name:      "plain value",
			s:         SecretString{value: "plain", isSecret: false},
			wantTag:   false,
			wantValue: "plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := yaml.Marshal(struct {
				Key SecretString `yaml:"key"`
			}{Key: tt.s})
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			result := string(data)
			hasTag := strings.Contains(result, "!secret")
			if hasTag != tt.wantTag {
				t.Errorf("hasTag = %v, want %v, yaml: %s", hasTag, tt.wantTag, result)
			}

			if !strings.Contains(result, tt.wantValue) {
				t.Errorf("yaml %q does not contain value %q", result, tt.wantValue)
			}
		})
	}
}

func TestSecretTracker(t *testing.T) {
	tracker := NewSecretTracker()

	// Initially no secrets
	if tracker.IsSecret("foo") {
		t.Error("expected foo to not be secret")
	}

	// Mark as secret
	tracker.MarkSecret("auth.session_secret")
	tracker.MarkSecret("stripe.secret_key")

	// Check
	if !tracker.IsSecret("auth.session_secret") {
		t.Error("expected auth.session_secret to be secret")
	}
	if !tracker.IsSecret("stripe.secret_key") {
		t.Error("expected stripe.secret_key to be secret")
	}
	if tracker.IsSecret("auth.enabled") {
		t.Error("expected auth.enabled to not be secret")
	}

	// Check paths
	paths := tracker.Paths()
	if len(paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(paths))
	}
}

func TestGenerateSecureSecret(t *testing.T) {
	secret1, err := GenerateSecureSecret()
	if err != nil {
		t.Fatalf("GenerateSecureSecret failed: %v", err)
	}

	secret2, err := GenerateSecureSecret()
	if err != nil {
		t.Fatalf("GenerateSecureSecret failed: %v", err)
	}

	// Should be different each time
	if secret1 == secret2 {
		t.Error("expected different secrets, got same")
	}

	// Should be base64 encoded (44 chars for 32 bytes)
	if len(secret1) != 44 {
		t.Errorf("expected secret length 44, got %d", len(secret1))
	}
}

func TestResolveSecretValue(t *testing.T) {
	tests := []struct {
		name     string
		s        SecretString
		envValue string
		wantAuto bool // if true, expect a generated value
		want     string
	}{
		{
			name:     "configured value",
			s:        NewSecretString("configured"),
			envValue: "",
			want:     "configured",
		},
		{
			name:     "env override",
			s:        NewSecretString("configured"),
			envValue: "fromenv",
			want:     "fromenv",
		},
		{
			name:     "auto generates",
			s:        NewSecretString("auto"),
			envValue: "",
			wantAuto: true,
		},
		{
			name:     "env overrides auto",
			s:        NewSecretString("auto"),
			envValue: "fromenv",
			want:     "fromenv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveSecretValue(tt.s, tt.envValue)
			if err != nil {
				t.Fatalf("ResolveSecretValue failed: %v", err)
			}

			if tt.wantAuto {
				// Should be a generated value (44 chars base64)
				if len(got) != 44 {
					t.Errorf("expected generated secret (44 chars), got %d chars: %q", len(got), got)
				}
			} else {
				if got != tt.want {
					t.Errorf("got %q, want %q", got, tt.want)
				}
			}
		})
	}
}
