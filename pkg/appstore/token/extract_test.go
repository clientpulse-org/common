package token

import (
	"testing"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		want     string
		wantLine string
		wantOk   bool
	}{
		{
			name:     "simple token extraction",
			html:     `<meta name="web-experience-app/config/environment" content="%7B%22MEDIA_API%22%3A%7B%22token%22%3A%22test-token-123%22%7D%7D">`,
			want:     "bearer test-token-123",
			wantLine: `<meta name="web-experience-app/config/environment" content="%7B%22MEDIA_API%22%3A%7B%22token%22%3A%22test-token-123%22%7D%7D">`,
			wantOk:   true,
		},
		{
			name:     "HTML with meta tag but no token",
			html:     `<meta name="web-experience-app/config/environment" content="%7B%22appVersion%22%3A1%2C%22modulePrefix%22%3A%22web-experience-app%22%7D">`,
			want:     "",
			wantLine: `<meta name="web-experience-app/config/environment" content="%7B%22appVersion%22%3A1%2C%22modulePrefix%22%3A%22web-experience-app%22%7D">`,
			wantOk:   false,
		},
		{
			name:     "HTML without meta tag",
			html:     `<html><head><title>Test</title></head><body>No token here</body></html>`,
			want:     "",
			wantLine: "",
			wantOk:   false,
		},
		{
			name:     "HTML with different meta tag name",
			html:     `<meta name="different-config" content="%7B%22token%22%3A%22test%22%7D">`,
			want:     "",
			wantLine: "",
			wantOk:   false,
		},
		{
			name:     "HTML with token in different format",
			html:     `<meta name="web-experience-app/config/environment" content="%7B%22MEDIA_API%22%3A%7B%22different_field%22%3A%22test%22%7D%7D">`,
			want:     "",
			wantLine: `<meta name="web-experience-app/config/environment" content="%7B%22MEDIA_API%22%3A%7B%22different_field%22%3A%22test%22%7D%7D">`,
			wantOk:   false,
		},
		{
			name:     "Empty HTML",
			html:     "",
			want:     "",
			wantLine: "",
			wantOk:   false,
		},
		{
			name: "HTML with multiple lines, token on second line",
			html: `<html>
<meta name="web-experience-app/config/environment" content="%7B%22MEDIA_API%22%3A%7B%22token%22%3A%22test-token-123%22%7D%7D">
<body>Content</body>`,
			want:     "bearer test-token-123",
			wantLine: `<meta name="web-experience-app/config/environment" content="%7B%22MEDIA_API%22%3A%7B%22token%22%3A%22test-token-123%22%7D%7D">`,
			wantOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotLine, gotOk := ExtractBearerToken(tt.html)

			if got != tt.want || gotLine != tt.wantLine || gotOk != tt.wantOk {
				t.Logf("Input HTML: %s", tt.html)
				t.Logf("Expected token: %q", tt.want)
				t.Logf("Got token: %q", got)
				t.Logf("Expected line: %q", tt.wantLine)
				t.Logf("Got line: %q", gotLine)
				t.Logf("Expected ok: %v", tt.wantOk)
				t.Logf("Got ok: %v", gotOk)
			}

			if got != tt.want {
				t.Errorf("ExtractBearerToken() got = %v, want %v", got, tt.want)
			}
			if gotLine != tt.wantLine {
				t.Errorf("ExtractBearerToken() gotLine = %v, want %v", gotLine, tt.wantLine)
			}
			if gotOk != tt.wantOk {
				t.Errorf("ExtractBearerToken() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestExtractBearerTokenRealExample(t *testing.T) {
	realHTML := `<meta name="web-experience-app/config/environment" content="%7B%22appVersion%22%3A1%2C%22modulePrefix%22%3A%22web-experience-app%22%2C%22environment%22%3A%22production%22%2C%22rootURL%22%3A%22%2F%22%2C%22locationType%22%3A%22history-hash-router-scroll%22%2C%22historySupportMiddleware%22%3Atrue%2C%22EmberENV%22%3A%7B%22FEATURES%22%3A%7B%7D%2C%22EXTEND_PROTOTYPES%22%3A%7B%22Date%22%3Afalse%7D%2C%22_APPLICATION_TEMPLATE_WRAPPER%22%3Afalse%2C%22_DEFAULT_ASYNC_OBSERVERS%22%3Atrue%2C%22_JQUERY_INTEGRATION%22%3Afalse%2C%22_TEMPLATE_ONLY_GLIMMER_COMPONENTS%22%3Atrue%7D%2C%22APP%22%3A%7B%22PROGRESS_BAR_DELAY%22%3A3000%2C%22CLOCK_INTERVAL%22%3A1000%2C%22LOADING_SPINNER_SPY%22%3Atrue%2C%22BREAKPOINTS%22%3A%7B%22large%22%3A%7B%22min%22%3A1069%2C%22content%22%3A980%7D%2C%22medium%22%3A%7B%22min%22%3A735%2C%22max%22%3A1068%2C%22content%22%3A692%7D%2C%22small%22%3A%7B%22min%22%3A320%2C%22max%22%3A734%2C%22content%22%3A280%7D%7D%2C%22buildVariant%22%3A%22apps%22%2C%22name%22%3A%22web-experience-app%22%2C%22version%22%3A%222532.1.0%2B09273a9c%22%7D%2C%22MEDIA_API%22%3A%7B%22token%22%3A%22eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlU4UlRZVjVaRFMifQ.eyJpc3MiOiI3TktaMlZQNDhaIiwiaWF0IjoxNzUzODA3MDIyLCJleHAiOjE3NjEwNjQ2MjIsInJvb3RfaHR0cHNfb3JpZ2luIjpbImFwcGxlLmNvbSJdfQ.J2kE8jfGDxL0E_FTh0Sm9Uuy-WLLoy59r_7k5XOJ3efOYMdW6sNSWIjcrtw7KHW2hk_VmE8SxgUO68CDphoirA%22%7D%2C%22i18n%22%3A%7B%22defaultLocale%22%3A%22en-gb%22%2C%22useDevLoc%22%3Afalse%2C%22pathToLocales%22%3A%22dist%2Flocales%22%7D%7D">`

	expectedToken := "bearer eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IlU4UlRZVjVaRFMifQ.eyJpc3MiOiI3TktaMlZQNDhaIiwiaWF0IjoxNzUzODA3MDIyLCJleHAiOjE3NjEwNjQ2MjIsInJvb3RfaHR0cHNfb3JpZ2luIjpbImFwcGxlLmNvbSJdfQ.J2kE8jfGDxL0E_FTh0Sm9Uuy-WLLoy59r_7k5XOJ3efOYMdW6sNSWIjcrtw7KHW2hk_VmE8SxgUO68CDphoirA"

	got, gotLine, gotOk := ExtractBearerToken(realHTML)

	t.Logf("Real HTML test:")
	t.Logf("Expected token: %q", expectedToken)
	t.Logf("Got token: %q", got)
	t.Logf("Got line: %q", gotLine)
	t.Logf("Got ok: %v", gotOk)

	if got != expectedToken {
		t.Errorf("Expected token %q, got %q", expectedToken, got)
	}

	if !gotOk {
		t.Errorf("Expected ok=true, got %v", gotOk)
	}

	if gotLine == "" {
		t.Errorf("Expected non-empty line, got empty")
	}
}
