package files

import (
	"testing"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

func TestComputeContentHash_StaticFile(t *testing.T) {
	fileModel := schemas.FileModel{
		Content: types.StringValue("*.log\n*.tmp\n"),
	}

	hash := ComputeContentHash(fileModel)

	assert.NotEmpty(t, hash, "Hash should not be empty")
	assert.Equal(t, 64, len(hash), "SHA256 hash should be 64 hex characters")
}

func TestComputeContentHash_InstructionFile(t *testing.T) {
	fileModel := schemas.FileModel{
		Instructions: []schemas.InstructionModel{
			{
				Prompt: types.StringValue("Create hello.go"),
				Constraints: []types.String{
					types.StringValue("Use package main"),
				},
			},
		},
	}

	hash := ComputeContentHash(fileModel)

	assert.NotEmpty(t, hash, "Hash should not be empty")
	assert.Equal(t, 64, len(hash), "SHA256 hash should be 64 hex characters")
}

func TestComputeContentHash_EmptyFile(t *testing.T) {
	fileModel := schemas.FileModel{}

	hash := ComputeContentHash(fileModel)

	assert.Empty(t, hash, "Hash should be empty for file with no content or instructions")
}

func TestComputeFileHash(t *testing.T) {
	content := []byte("package main\n\nfunc main() {}\n")

	hash := ComputeFileHash(content)

	assert.NotEmpty(t, hash, "Hash should not be empty")
	assert.Equal(t, 64, len(hash), "SHA256 hash should be 64 hex characters")

	// Same content should produce same hash
	hash2 := ComputeFileHash(content)
	assert.Equal(t, hash, hash2, "Same content should produce same hash")

	// Different content should produce different hash
	differentContent := []byte("package main\n\nfunc main() { println(\"hello\") }\n")
	hash3 := ComputeFileHash(differentContent)
	assert.NotEqual(t, hash, hash3, "Different content should produce different hash")
}
