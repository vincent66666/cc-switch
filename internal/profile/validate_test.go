package profile

import "testing"

func TestValidateProfile_RequiresTokenAndBaseURL(t *testing.T) {
	t.Run("missing token", func(t *testing.T) {
		err := ValidateProfile("demo", Profile{
			Env: map[string]string{
				"ANTHROPIC_BASE_URL": "https://example.com",
			},
		})
		if err == nil {
			t.Fatal("expected missing token to fail validation")
		}
	})

	t.Run("missing base url", func(t *testing.T) {
		err := ValidateProfile("demo", Profile{
			Env: map[string]string{
				"ANTHROPIC_AUTH_TOKEN": "token",
			},
		})
		if err == nil {
			t.Fatal("expected missing base url to fail validation")
		}
	})
}

func TestValidateProfile_AllowsMissingOptionalModels(t *testing.T) {
	err := ValidateProfile("demo", Profile{
		Env: map[string]string{
			"ANTHROPIC_AUTH_TOKEN": "token",
			"ANTHROPIC_BASE_URL":   "https://example.com",
		},
	})
	if err != nil {
		t.Fatalf("expected optional models to be optional, got %v", err)
	}
}
