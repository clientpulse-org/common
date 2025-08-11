package landing

import (
	"errors"
	"testing"
)

func TestBuildLandingURL(t *testing.T) {
	tests := []struct {
		name        string
		country     string
		appName     string
		appID       string
		expectedURL string
		expectedErr error
	}{
		{
			name:        "valid parameters",
			country:     "US",
			appName:     "instagram",
			appID:       "389801252",
			expectedURL: "https://apps.apple.com/us/app/instagram/id389801252",
			expectedErr: nil,
		},
		{
			name:        "missing country",
			country:     "",
			appName:     "instagram",
			appID:       "389801252",
			expectedErr: ErrCountryRequired,
		},
		{
			name:        "missing app name",
			country:     "us",
			appName:     "",
			appID:       "389801252",
			expectedErr: ErrAppNameRequired,
		},
		{
			name:        "missing app ID",
			country:     "us",
			appName:     "instagram",
			appID:       "",
			expectedErr: ErrAppIDRequired,
		},
		{
			name:        "invalid country format",
			country:     "usa",
			appName:     "instagram",
			appID:       "389801252",
			expectedErr: ErrCountryInvalid,
		},
		{
			name:        "invalid app id (non-numeric)",
			country:     "us",
			appName:     "instagram",
			appID:       "abc123",
			expectedErr: ErrAppIDInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := BuildLandingURL(tt.country, tt.appName, tt.appID)

			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if url != tt.expectedURL {
				t.Errorf("expected URL %s, got %s", tt.expectedURL, url)
			}
		})
	}
}

func TestNormalizeCountryCode(t *testing.T) {
	if got := NormalizeCountryCode(" US "); got != "us" {
		t.Errorf("expected 'us', got %q", got)
	}
}
