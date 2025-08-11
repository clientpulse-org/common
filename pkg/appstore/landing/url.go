package landing

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

const (
	Scheme              = "https"
	LandingHost         = "apps.apple.com"
	LandingPathTemplate = "{country}/app/{app_name}/id{app_id}"
)

var (
	ErrCountryRequired = errors.New("country is required")
	ErrAppNameRequired = errors.New("app name is required")
	ErrAppIDRequired   = errors.New("app ID is required")
	ErrCountryInvalid  = errors.New("country must be a 2-letter ISO code")
	ErrAppIDInvalid    = errors.New("app ID must be numeric")
)

var (
	countryCodeRegex = regexp.MustCompile(`^[a-z]{2}$`)
	appIDRegex       = regexp.MustCompile(`^[0-9]+$`)
)

func BuildLandingURL(country, appName, appID string) (string, error) {
	country = NormalizeCountryCode(country)
	appName = strings.TrimSpace(appName)
	appID = strings.TrimSpace(appID)

	if country == "" {
		return "", ErrCountryRequired
	}
	if appName == "" {
		return "", ErrAppNameRequired
	}
	if appID == "" {
		return "", ErrAppIDRequired
	}
	if !countryCodeRegex.MatchString(country) {
		return "", ErrCountryInvalid
	}
	if !appIDRegex.MatchString(appID) {
		return "", ErrAppIDInvalid
	}

	path := buildLandingPath(country, appName, appID)

	u := url.URL{
		Scheme: Scheme,
		Host:   LandingHost,
		Path:   "/" + path,
	}
	return u.String(), nil
}

func NormalizeCountryCode(country string) string {
	return strings.ToLower(strings.TrimSpace(country))
}

func buildLandingPath(country, appSlug, appID string) string {
	path := LandingPathTemplate
	replacer := strings.NewReplacer(
		"{country}", country,
		"{app_name}", appSlug,
		"{app_id}", appID,
	)
	return replacer.Replace(path)
}
