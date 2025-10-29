package resources

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// FileNameValidator validates that a file name is filesystem-compliant across platforms
type FileNameValidator struct{}

// Description returns a description of the validator
func (v FileNameValidator) Description(ctx context.Context) string {
	return "value must be a valid filesystem path without invalid characters, reserved names, or leading/trailing whitespace"
}

// MarkdownDescription returns a markdown description of the validator
func (v FileNameValidator) MarkdownDescription(ctx context.Context) string {
	return "Value must be a valid filesystem path:\n" +
		"- No invalid characters: `< > : \" | ? * \\`\n" +
		"- No spaces (use hyphens or underscores)\n" +
		"- No leading/trailing whitespace\n" +
		"- No trailing dots\n" +
		"- No Windows reserved names (CON, PRN, AUX, NUL, COM1-9, LPT1-9)\n" +
		"- Each path component must not exceed 255 characters\n" +
		"- Use forward slashes `/` for directory separators"
}

// ValidateString validates the file name
func (v FileNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	name := req.ConfigValue.ValueString()

	// 1. Must not be empty
	if name == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid File Name",
			"File name cannot be empty",
		)
		return
	}

	// 2. No leading/trailing whitespace
	if strings.TrimSpace(name) != name {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid File Name",
			fmt.Sprintf("File name cannot have leading or trailing whitespace: %q", name),
		)
		return
	}

	// 3. No spaces anywhere (enforce clean naming)
	if strings.Contains(name, " ") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid File Name",
			fmt.Sprintf("File name cannot contain spaces. Use hyphens or underscores instead: %q", name),
		)
		return
	}

	// 4. No trailing dots
	if strings.HasSuffix(name, ".") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid File Name",
			fmt.Sprintf("File name cannot end with a dot: %q", name),
		)
		return
	}

	// 5. Invalid characters (Windows is most restrictive)
	invalidChars := `<>:"|?*\` + "\x00" // Include null character
	invalidCharsPattern := regexp.MustCompile(`[<>:"|?*\\\x00-\x1f]`)
	if invalidCharsPattern.MatchString(name) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid File Name",
			fmt.Sprintf("File name contains invalid characters. Cannot use: %s\nGot: %q", invalidChars, name),
		)
		return
	}

	// 6. Check each path component
	components := strings.Split(name, "/")
	for i, component := range components {
		// Empty components (e.g., "dir//file.txt")
		if component == "" && i < len(components)-1 {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid File Name",
				fmt.Sprintf("File path cannot have empty components (double slashes): %q", name),
			)
			return
		}

		// Component too long
		if len(component) > 255 {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid File Name",
				fmt.Sprintf("Path component %q exceeds 255 character limit", component),
			)
			return
		}

		// Windows reserved names (case-insensitive)
		componentUpper := strings.ToUpper(component)
		// Remove extension for reserved name check
		componentBase := strings.TrimSuffix(componentUpper, filepath.Ext(componentUpper))

		reservedNames := []string{
			"CON", "PRN", "AUX", "NUL",
			"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
			"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
		}
		for _, reserved := range reservedNames {
			if componentBase == reserved {
				resp.Diagnostics.AddAttributeError(
					req.Path,
					"Invalid File Name",
					fmt.Sprintf("File name cannot use Windows reserved name: %q (component: %q)", reserved, component),
				)
				return
			}
		}
	}
}

// ValidFileName returns a validator that checks if a string is a valid filesystem path
func ValidFileName() validator.String {
	return FileNameValidator{}
}
