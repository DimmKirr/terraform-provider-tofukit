package uri

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewScanner(t *testing.T) {
	scanner := NewScanner()
	assert.NotNil(t, scanner)
	assert.NotNil(t, scanner.pattern)
}

func TestExtractURIs_EmptyInput(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{}

	uris := scanner.ExtractURIs(fields)

	assert.Empty(t, uris)
}

func TestExtractURIs_NoURIs(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{
		"description": "Just a plain description",
		"content":     "Some file content without URIs",
	}

	uris := scanner.ExtractURIs(fields)

	assert.Empty(t, uris)
}

func TestExtractURIs_SingleFeatureURI(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{
		"prompt": "Use tofukit://feature/logging for structured logging",
	}

	uris := scanner.ExtractURIs(fields)

	assert.Equal(t, []string{"tofukit://feature/logging"}, uris)
}

func TestExtractURIs_SingleFileURI(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{
		"instruction": "Include tofukit://file/gitignore in the project",
	}

	uris := scanner.ExtractURIs(fields)

	assert.Equal(t, []string{"tofukit://file/gitignore"}, uris)
}

func TestExtractURIs_SingleStackURI(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{
		"description": "Based on tofukit://stack/python-click-app",
	}

	uris := scanner.ExtractURIs(fields)

	assert.Equal(t, []string{"tofukit://stack/python-click-app"}, uris)
}

func TestExtractURIs_SingleKitLanguageURI(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{
		"requirement": "Use tofukit://kit/language/python for the language",
	}

	uris := scanner.ExtractURIs(fields)

	assert.Equal(t, []string{"tofukit://kit/language/python"}, uris)
}

func TestExtractURIs_SingleKitFrameworkURI(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{
		"prompt": "Setup tofukit://kit/framework/click for CLI",
	}

	uris := scanner.ExtractURIs(fields)

	assert.Equal(t, []string{"tofukit://kit/framework/click"}, uris)
}

func TestExtractURIs_MultipleURIsInOneField(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{
		"prompt": "Use tofukit://feature/logging and tofukit://feature/config together",
	}

	uris := scanner.ExtractURIs(fields)

	assert.ElementsMatch(t, []string{"tofukit://feature/logging", "tofukit://feature/config"}, uris)
}

func TestExtractURIs_URIsAcrossMultipleFields(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{
		"prompt":      "Use tofukit://feature/logging",
		"constraint":  "Based on tofukit://kit/language/python",
		"instruction": "Include tofukit://file/readme",
	}

	uris := scanner.ExtractURIs(fields)

	assert.ElementsMatch(t, []string{
		"tofukit://feature/logging",
		"tofukit://kit/language/python",
		"tofukit://file/readme",
	}, uris)
}

func TestExtractURIs_DeduplicatesDuplicates(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{
		"prompt":     "Use tofukit://feature/logging",
		"constraint": "Also use tofukit://feature/logging here",
		"note":       "And again tofukit://feature/logging",
	}

	uris := scanner.ExtractURIs(fields)

	assert.Equal(t, []string{"tofukit://feature/logging"}, uris)
}

func TestExtractURIs_HandlesHyphensAndUnderscores(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{
		"prompt": "Use tofukit://feature/my-feature_v2 and tofukit://kit/framework/some_framework-v1",
	}

	uris := scanner.ExtractURIs(fields)

	assert.ElementsMatch(t, []string{
		"tofukit://feature/my-feature_v2",
		"tofukit://kit/framework/some_framework-v1",
	}, uris)
}

func TestExtractURIs_RejectsInvalidScheme(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{
		"prompt": "Use http://example.com and ftp://files.com",
	}

	uris := scanner.ExtractURIs(fields)

	assert.Empty(t, uris)
}

func TestExtractURIs_RejectsInvalidFormat(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{
		"prompt": "tofukit://invalid tofukit://",
	}

	uris := scanner.ExtractURIs(fields)

	// ExtractURIs uses regex which may match some invalid URIs
	// Those will fail later during ParseURI validation
	// This test verifies that obvious invalid formats are not matched
	assert.Empty(t, uris)
}

func TestExtractURIs_MatchesButMayBeInvalid(t *testing.T) {
	scanner := NewScanner()
	fields := map[string]string{
		"prompt": "tofukit://too/many/parts/here",
	}

	uris := scanner.ExtractURIs(fields)

	// The regex may match this, but ParseURI will reject it
	// Let's verify that if it matches, ParseURI rejects it
	if len(uris) > 0 {
		for _, uri := range uris {
			err := scanner.ValidateURI(uri)
			assert.Error(t, err, "URI should fail validation: %s", uri)
		}
	}
}

func TestParseURI_FeatureURI(t *testing.T) {
	scanner := NewScanner()

	components, err := scanner.ParseURI("tofukit://feature/logging")

	assert.NoError(t, err)
	assert.Equal(t, "feature", components.Type)
	assert.Equal(t, "logging", components.Name)
}

func TestParseURI_FileURI(t *testing.T) {
	scanner := NewScanner()

	components, err := scanner.ParseURI("tofukit://file/gitignore")

	assert.NoError(t, err)
	assert.Equal(t, "file", components.Type)
	assert.Equal(t, "gitignore", components.Name)
}

func TestParseURI_StackURI(t *testing.T) {
	scanner := NewScanner()

	components, err := scanner.ParseURI("tofukit://stack/python-app")

	assert.NoError(t, err)
	assert.Equal(t, "stack", components.Type)
	assert.Equal(t, "python-app", components.Name)
}

func TestParseURI_KitLanguageURI(t *testing.T) {
	scanner := NewScanner()

	components, err := scanner.ParseURI("tofukit://kit/language/python")

	assert.NoError(t, err)
	assert.Equal(t, "kit/language", components.Type)
	assert.Equal(t, "python", components.Name)
}

func TestParseURI_KitFrameworkURI(t *testing.T) {
	scanner := NewScanner()

	components, err := scanner.ParseURI("tofukit://kit/framework/click")

	assert.NoError(t, err)
	assert.Equal(t, "kit/framework", components.Type)
	assert.Equal(t, "click", components.Name)
}

func TestParseURI_KitToolURI(t *testing.T) {
	scanner := NewScanner()

	components, err := scanner.ParseURI("tofukit://kit/tool/pytest")

	assert.NoError(t, err)
	assert.Equal(t, "kit/tool", components.Type)
	assert.Equal(t, "pytest", components.Name)
}

func TestParseURI_InvalidScheme(t *testing.T) {
	scanner := NewScanner()

	_, err := scanner.ParseURI("http://feature/logging")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URI scheme")
}

func TestParseURI_InvalidFormat(t *testing.T) {
	scanner := NewScanner()

	_, err := scanner.ParseURI("tofukit://invalid")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URI format")
}

func TestParseURI_TooManyParts(t *testing.T) {
	scanner := NewScanner()

	_, err := scanner.ParseURI("tofukit://too/many/parts/here")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URI format")
}

func TestParseURI_EmptyParts(t *testing.T) {
	scanner := NewScanner()

	// "tofukit:///" splits to ["", ""] which triggers "missing name" error
	_, err := scanner.ParseURI("tofukit://feature/")

	assert.Error(t, err)
}

func TestParseURI_WithHyphensAndUnderscores(t *testing.T) {
	scanner := NewScanner()

	components, err := scanner.ParseURI("tofukit://feature/my-feature_v2")

	assert.NoError(t, err)
	assert.Equal(t, "feature", components.Type)
	assert.Equal(t, "my-feature_v2", components.Name)
}
