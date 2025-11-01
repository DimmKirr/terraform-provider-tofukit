package files

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// ComputeContentHash generates SHA256 hash of file specification
// For static files: hash of content string
// For generated files: hash of instructions JSON
func ComputeContentHash(file schemas.FileModel) string {
	var hashInput []byte

	if !file.Content.IsNull() && file.Content.ValueString() != "" {
		// Static file: hash the content
		hashInput = []byte(file.Content.ValueString())
	} else if len(file.Instructions) > 0 {
		// Generated file: hash the instructions JSON
		jsonBytes, err := json.Marshal(file.Instructions)
		if err != nil {
			return ""
		}
		hashInput = jsonBytes
	} else {
		return "" // No hashable content
	}

	hash := sha256.Sum256(hashInput)
	return hex.EncodeToString(hash[:])
}

// ComputeFileHash generates SHA256 hash of actual file content
func ComputeFileHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}
