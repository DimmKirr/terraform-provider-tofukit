package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tofukit/opentofu-provider-tofukit/internal/testutil"
)

// TestProjectResourceFinal_Schema tests the resource schema
func TestProjectResourceFinal_Schema(t *testing.T) {
	ctx := context.Background()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	// Create resource and get schema
	r := NewProjectResourceFinal()
	r.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() returned errors: %v", resp.Diagnostics.Errors())
	}

	schema := resp.Schema

	// Validate required attributes exist
	requiredAttrs := []string{"name", "version"}
	for _, attr := range requiredAttrs {
		if _, ok := schema.Attributes[attr]; !ok {
			t.Errorf("Required attribute %s not found in schema", attr)
		}
	}

	// Validate computed attributes exist
	computedAttrs := []string{"id", "execution_status", "execution_started", "execution_completed", "project_path", "execution_error"}
	for _, attr := range computedAttrs {
		if schemaAttr, ok := schema.Attributes[attr]; !ok {
			t.Errorf("Computed attribute %s not found in schema", attr)
		} else if !schemaAttr.IsComputed() {
			t.Errorf("Attribute %s should be computed", attr)
		}
	}

	// Validate blocks exist
	if _, ok := schema.Blocks["requirement"]; !ok {
		t.Error("requirement block not found in schema")
	}
}

// TestProjectResourceFinal_Metadata tests the resource metadata
func TestProjectResourceFinal_Metadata(t *testing.T) {
	ctx := context.Background()
	req := resource.MetadataRequest{
		ProviderTypeName: "tofukit",
	}
	resp := &resource.MetadataResponse{}

	r := NewProjectResourceFinal()
	r.Metadata(ctx, req, resp)

	expectedTypeName := "tofukit_project"
	if resp.TypeName != expectedTypeName {
		t.Errorf("Expected TypeName %s, got %s", expectedTypeName, resp.TypeName)
	}
}

// TestProjectResourceFinal_ValidateConfig tests configuration validation
func TestProjectResourceFinal_ValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    ProjectModelFinal
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid config",
			config: ProjectModelFinal{
				Name:        types.StringValue("test-project"),
				Description: types.StringValue("Test project"),
				Version:     types.StringValue("1.0.0"),
			},
			wantError: false,
		},
		{
			name: "empty name",
			config: ProjectModelFinal{
				Name:        types.StringValue(""),
				Description: types.StringValue("Test project"),
				Version:     types.StringValue("1.0.0"),
			},
			wantError: true,
			errorMsg:  "Project name cannot be empty",
		},
		{
			name: "very long name",
			config: ProjectModelFinal{
				Name:        types.StringValue("this-is-a-very-long-project-name-that-exceeds-reasonable-limits-and-might-cause-filesystem-issues-when-creating-directories"),
				Description: types.StringValue("Test project"),
				Version:     types.StringValue("1.0.0"),
			},
			wantError: false, // Should be warning, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			r := NewProjectResourceFinal()

			// Create a mock provider data
			r.ProviderData = &mockProviderData{
				dryRun:           true,
				claudeHomeDir:    "~/.claude",
				outputPath:       "/tmp/test",
				outputFormat:     "json",
			}

			req := resource.ValidateConfigRequest{}
			resp := &resource.ValidateConfigResponse{}

			// Set the config - in real usage, this would be populated from HCL
			// For unit tests, we simulate the populated config
			req.Config = testConfigFromModel(tt.config)

			r.ValidateConfig(ctx, req, resp)

			hasError := resp.Diagnostics.HasError()
			if hasError != tt.wantError {
				t.Errorf("ValidateConfig() error = %v, wantError %v", hasError, tt.wantError)
				if hasError {
					t.Logf("Actual errors: %v", resp.Diagnostics.Errors())
				}
			}

			if tt.wantError && tt.errorMsg != "" {
				found := false
				for _, diag := range resp.Diagnostics.Errors() {
					if diag.Summary() == "Invalid project name" && diag.Detail() == tt.errorMsg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message '%s' not found", tt.errorMsg)
				}
			}
		})
	}
}

// TestProjectResourceFinal_buildOutputData tests the output data building
func TestProjectResourceFinal_buildOutputData(t *testing.T) {
	ctx := context.Background()
	r := NewProjectResourceFinal()

	model := testutil.TestProjectModelFinal()

	outputData := r.buildOutputData(ctx, model)

	// Validate project section
	project, ok := outputData["project"].(map[string]interface{})
	if !ok {
		t.Fatal("project section not found or invalid type")
	}

	if project["name"] != "test-hello" {
		t.Errorf("Expected project name 'test-hello', got %v", project["name"])
	}

	if project["description"] != "Test project" {
		t.Errorf("Expected project description 'Test project', got %v", project["description"])
	}

	if project["version"] != "1.0.0" {
		t.Errorf("Expected project version '1.0.0', got %v", project["version"])
	}

	// Validate requirements section
	requirements, ok := outputData["requirements"].([]map[string]interface{})
	if !ok {
		t.Fatal("requirements section not found or invalid type")
	}

	if len(requirements) != 1 {
		t.Errorf("Expected 1 requirement, got %d", len(requirements))
	}

	if requirements[0]["name"] != "Create hello.txt" {
		t.Errorf("Expected requirement name 'Create hello.txt', got %v", requirements[0]["name"])
	}

	// Validate instructions
	instructions, ok := requirements[0]["instructions"].([]string)
	if !ok {
		t.Fatal("instructions not found or invalid type")
	}

	if len(instructions) != 2 {
		t.Errorf("Expected 2 instructions, got %d", len(instructions))
	}

	// Validate verification
	verification, ok := requirements[0]["verification"].(map[string]string)
	if !ok {
		t.Fatal("verification not found or invalid type")
	}

	if verification["command"] != "test -f hello.txt" {
		t.Errorf("Expected verification command 'test -f hello.txt', got %v", verification["command"])
	}
}

// TestProjectResourceFinal_executeClaudeCode tests Claude Code execution
func TestProjectResourceFinal_executeClaudeCode(t *testing.T) {
	ctx := context.Background()
	r := NewProjectResourceFinal()

	tmpDir := testutil.CreateTempProjectDir(t)
	model := testutil.TestProjectModelFinal()
	outputData := map[string]interface{}{
		"project": map[string]interface{}{
			"name":        "test-project",
			"description": "Test project",
			"version":     "1.0.0",
		},
		"requirements": []map[string]interface{}{
			{
				"name": "Create hello.txt",
				"instructions": []string{
					"Create a file named hello.txt",
					"Add content: Hello, World!",
				},
			},
		},
	}

	// Test with dry run
	err := r.executeClaudeCode(ctx, &model, outputData, tmpDir, "~/.claude", true)
	if err != nil {
		t.Fatalf("executeClaudeCode() with dry run failed: %v", err)
	}

	// Verify status was updated
	if model.ExecutionStatus.ValueString() != "completed" {
		t.Errorf("Expected execution status 'completed', got %v", model.ExecutionStatus.ValueString())
	}

	if model.ExecutionStarted.ValueString() == "" {
		t.Error("Expected execution_started to be set")
	}

	if model.ExecutionCompleted.ValueString() == "" {
		t.Error("Expected execution_completed to be set")
	}

	if model.ProjectPath.ValueString() == "" {
		t.Error("Expected project_path to be set")
	}
}

// Mock provider data for testing
type mockProviderData struct {
	dryRun           bool
	claudeHomeDir    string
	outputPath       string
	outputFormat     string
}

func (m *mockProviderData) GetDryRun() bool {
	return m.dryRun
}

func (m *mockProviderData) GetClaudeHomeDirectory() string {
	return m.claudeHomeDir
}

func (m *mockProviderData) GetOutputPath() string {
	return m.outputPath
}

func (m *mockProviderData) GetOutputFormat() string {
	return m.outputFormat
}

// Helper function to create a config from a model for testing
func testConfigFromModel(model ProjectModelFinal) interface{} {
	// In real usage, this would be a proper Config type from the framework
	// For unit tests, we simulate with the model itself
	return &model
}

// Benchmark tests
func BenchmarkProjectResourceFinal_buildOutputData(b *testing.B) {
	ctx := context.Background()
	r := NewProjectResourceFinal()
	model := testutil.TestProjectModelFinal()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.buildOutputData(ctx, model)
	}
}