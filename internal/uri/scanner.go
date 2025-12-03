package uri

import (
	"fmt"
	"regexp"
	"strings"
)

// URIComponents represents the parsed components of a tofukit:// URI
type URIComponents struct {
	Type string // e.g., "feature", "file", "stack", "kit/language", "kit/library"
	Name string // e.g., "pterm_logger", "readme", "python312"
}

// Scanner scans text for tofukit:// URIs
type Scanner struct {
	pattern *regexp.Regexp
}

// NewScanner creates a new URI scanner
func NewScanner() *Scanner {
	// Pattern matches: tofukit://<type>/<name> or tofukit://kit/<subtype>/<name>
	// Type must be lowercase, name can have alphanumeric, underscore, hyphen
	return &Scanner{
		pattern: regexp.MustCompile(`tofukit://[a-z]+(/[a-z]+)?/[a-zA-Z0-9_-]+`),
	}
}

// ExtractURIs scans all string values and returns unique URIs
func (s *Scanner) ExtractURIs(fields map[string]string) []string {
	seen := make(map[string]bool)
	var uris []string

	for _, value := range fields {
		matches := s.pattern.FindAllString(value, -1)
		for _, uri := range matches {
			if !seen[uri] {
				seen[uri] = true
				uris = append(uris, uri)
			}
		}
	}

	return uris
}

// ParseURI breaks down a URI into components
// Examples:
//   - tofukit://feature/logger → {Type: "feature", Name: "logger"}
//   - tofukit://kit/language/python312 → {Type: "kit/language", Name: "python312"}
//   - tofukit://kit/library/viper → {Type: "kit/library", Name: "viper"}
func (s *Scanner) ParseURI(uri string) (URIComponents, error) {
	// Remove scheme prefix
	if !strings.HasPrefix(uri, "tofukit://") {
		return URIComponents{}, fmt.Errorf("invalid URI scheme: %s", uri)
	}

	path := strings.TrimPrefix(uri, "tofukit://")
	parts := strings.Split(path, "/")

	// Validate URI structure
	if len(parts) < 2 {
		return URIComponents{}, fmt.Errorf("invalid URI format (missing name): %s", uri)
	}

	if len(parts) == 2 {
		// Simple type: tofukit://feature/name or tofukit://file/name
		if parts[0] == "" || parts[1] == "" {
			return URIComponents{}, fmt.Errorf("invalid URI format (empty parts): %s", uri)
		}
		return URIComponents{
			Type: parts[0],
			Name: parts[1],
		}, nil
	}

	if len(parts) == 3 && parts[0] == "kit" {
		// Kit type: tofukit://kit/language/name or tofukit://kit/library/name
		if parts[1] == "" || parts[2] == "" {
			return URIComponents{}, fmt.Errorf("invalid URI format (empty parts): %s", uri)
		}
		return URIComponents{
			Type: fmt.Sprintf("kit/%s", parts[1]),
			Name: parts[2],
		}, nil
	}

	return URIComponents{}, fmt.Errorf("invalid URI format (unexpected structure): %s", uri)
}

// ValidateURI checks if a URI is valid
func (s *Scanner) ValidateURI(uri string) error {
	_, err := s.ParseURI(uri)
	return err
}
