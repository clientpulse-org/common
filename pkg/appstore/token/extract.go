package token

import (
	"regexp"
	"strings"
)

var (
	needle  = "web-experience-app/config/environment"
	tokenRe = regexp.MustCompile(`token%22%3A%22(.+?)%22`)
)

func ExtractBearerToken(html string) (string, string, bool) {
	for line := range strings.SplitSeq(html, "\n") {
		if strings.Contains(line, needle) {
			if m := tokenRe.FindStringSubmatch(line); len(m) >= 2 {
				return "bearer " + m[1], line, true
			}
			return "", line, false
		}
	}
	return "", "", false
}
