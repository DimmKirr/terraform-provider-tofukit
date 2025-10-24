package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// Ensure provider defined interfaces are implemented
var _ datasource.DataSource = &QueryDataSource{}

// NewQueryDataSource creates a new Query data source
func NewQueryDataSource() datasource.DataSource {
	return &QueryDataSource{}
}

// QueryDataSource defines the data source implementation.
type QueryDataSource struct {
	llmExecutor llm.LLMExecutor
	debug       bool
	outputPath  string
}

// QueryDataSourceModel describes the data source data model.
type QueryDataSourceModel struct {
	ID           types.String               `tfsdk:"id"`
	Instructions []schemas.InstructionModel `tfsdk:"instruction"`
	Model        types.String               `tfsdk:"model"`
	JSON         types.String               `tfsdk:"json"`
	Text         types.String               `tfsdk:"text"`
}

// ClaudeCLIResponse represents the JSON response structure from Claude CLI --output-format=json
type ClaudeCLIResponse struct {
	Type         string                 `json:"type"`
	Subtype      string                 `json:"subtype"`
	IsError      bool                   `json:"is_error"`
	Result       string                 `json:"result"`
	SessionID    string                 `json:"session_id"`
	DurationMS   int                    `json:"duration_ms"`
	NumTurns     int                    `json:"num_turns"`
	TotalCostUSD float64                `json:"total_cost_usd"`
	ModelUsage   map[string]interface{} `json:"modelUsage"`
}

func (d *QueryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_query"
}

func (d *QueryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Query data source that executes LLM with instructions and returns structured output",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier",
			},
			"model": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Model to use for the query. Defaults are provider-specific: 'haiku' for Claude, 'gpt-4' for OpenAI (when implemented), 'gemini-pro' for Gemini (when implemented)",
			},
			"json": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The full JSON response from the LLM",
			},
			"text": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The extracted text message from the JSON response (value of 'message' field)",
			},
		},
		Blocks: map[string]schema.Block{
			"instruction": schema.ListNestedBlock{
				MarkdownDescription: "Instructions to execute with the LLM",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"prompt": schema.StringAttribute{
							MarkdownDescription: "LLM-facing instruction describing what to query",
							Required:            true,
						},
						"constraints": schema.ListAttribute{
							MarkdownDescription: "Constraints that must be respected (what NOT to do)",
							Optional:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *QueryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider is not configured.
	if req.ProviderData == nil {
		return
	}

	// Use type assertion to extract provider configuration
	// The provider passes ProviderData as an interface{}
	providerData, ok := req.ProviderData.(interface {
		GetLLMExecutor() llm.LLMExecutor
		IsDebug() bool
		GetOutputPath() string
	})

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected provider data with GetLLMExecutor, IsDebug, and GetOutputPath methods, got: %T", req.ProviderData),
		)
		return
	}

	d.llmExecutor = providerData.GetLLMExecutor()
	d.debug = providerData.IsDebug()
	d.outputPath = providerData.GetOutputPath()

	if d.llmExecutor == nil {
		resp.Diagnostics.AddError(
			"Missing LLM Executor",
			"LLM executor is not configured in the provider",
		)
		return
	}
}

func (d *QueryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data QueryDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build instructions from instruction blocks
	var instructions []string
	for _, inst := range data.Instructions {
		instructions = append(instructions, inst.Prompt.ValueString())

		// Add constraints as additional instructions
		if inst.Constraints != nil && len(inst.Constraints) > 0 {
			instructions = append(instructions, "")
			instructions = append(instructions, "Constraints (what NOT to do):")
			for _, c := range inst.Constraints {
				if !c.IsNull() && !c.IsUnknown() {
					instructions = append(instructions, "- "+c.ValueString())
				}
			}
		}
	}

	// Extract model (empty string means use LLM provider's default)
	model := ""
	if !data.Model.IsNull() && !data.Model.IsUnknown() {
		model = data.Model.ValueString()
	}

	tflog.Info(ctx, "Executing LLM query", map[string]interface{}{
		"instruction_count": len(instructions),
		"model_requested":   model,
	})

	// Build the query - always request JSON with example schema
	queryInstructions := append(instructions,
		"",
		"Output your answer as a JSON object following this exact format:",
		`{"message": "your concise answer here"}`,
		"",
		`Example: If asked "What color is the sky?", respond with:`,
		`{"message": "blue"}`,
		"",
		"Be concise and direct in your answer.",
	)

	// Execute the query using LLM (no project creation, just a simple query)
	responseOutput, err := d.llmExecutor.Query(ctx, queryInstructions, model)
	if err != nil {
		resp.Diagnostics.AddError(
			"LLM Query Failed",
			fmt.Sprintf("Failed to execute query: %v", err),
		)
		return
	}

	tflog.Info(ctx, "Query executed successfully", map[string]interface{}{
		"output_length": len(responseOutput),
	})

	// Clean the output (remove ANSI escape codes and control characters)
	cleanOutput := d.cleanOutput(responseOutput)

	// Parse the outer Claude CLI JSON response structure (when using --output-format=json)
	var cliResponse ClaudeCLIResponse
	var resultText string

	if err := json.Unmarshal([]byte(cleanOutput), &cliResponse); err == nil {
		// Successfully parsed Claude CLI JSON response
		resultText = cliResponse.Result

		// Extract model names from modelUsage
		var modelNames []string
		for modelName := range cliResponse.ModelUsage {
			modelNames = append(modelNames, modelName)
		}

		tflog.Info(ctx, "Claude query completed successfully", map[string]interface{}{
			"models_used":    modelNames,
			"session_id":     cliResponse.SessionID,
			"duration_ms":    cliResponse.DurationMS,
			"num_turns":      cliResponse.NumTurns,
			"total_cost_usd": cliResponse.TotalCostUSD,
		})

		tflog.Debug(ctx, "Extracted result from Claude CLI JSON response", map[string]interface{}{
			"result_length": len(resultText),
		})

		// Check if there was an error in the CLI response
		if cliResponse.IsError {
			tflog.Error(ctx, "Claude CLI reported an error", map[string]interface{}{
				"type":    cliResponse.Type,
				"subtype": cliResponse.Subtype,
			})
		}
	} else {
		// Not a JSON response from Claude CLI, use as-is
		tflog.Debug(ctx, "Response is not Claude CLI JSON format, using as-is", map[string]interface{}{
			"error": err.Error(),
		})
		resultText = cleanOutput
	}

	// Extract JSON from the result text
	jsonOutput := d.extractJSON(resultText)

	// Parse the JSON to extract the message field
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &jsonData); err != nil {
		tflog.Warn(ctx, "JSON parsing failed, wrapping raw output", map[string]interface{}{
			"error": err.Error(),
		})
		// Fall back to wrapping clean output in JSON
		data.JSON = types.StringValue(fmt.Sprintf(`{"message": %q}`, resultText))
		data.Text = types.StringValue(resultText)
	} else {
		// Successfully parsed JSON
		data.JSON = types.StringValue(jsonOutput)

		// Extract the "message" field for text output
		if message, ok := jsonData["message"].(string); ok {
			data.Text = types.StringValue(message)
		} else {
			// If no "message" field, use the full JSON as text
			tflog.Warn(ctx, "No 'message' field in JSON response, using full JSON", map[string]interface{}{
				"json": jsonOutput,
			})
			data.Text = types.StringValue(jsonOutput)
		}
	}

	// Generate a unique ID for this query
	data.ID = types.StringValue(fmt.Sprintf("query_%d", time.Now().UnixNano()))

	tflog.Debug(ctx, "Query response processed", map[string]interface{}{
		"json_length": len(data.JSON.ValueString()),
		"text_length": len(data.Text.ValueString()),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// extractJSON attempts to extract JSON from LLM output
func (d *QueryDataSource) extractJSON(output string) string {
	// Try to find JSON object in the output
	// Look for patterns like {"key": "value"}

	// First, try direct JSON parsing
	var jsonData interface{}
	if err := json.Unmarshal([]byte(output), &jsonData); err == nil {
		return output
	}

	// Try to find JSON block between ```json and ``` markers
	if strings.Contains(output, "```json") {
		start := strings.Index(output, "```json") + 7
		end := strings.Index(output[start:], "```")
		if end > 0 {
			jsonStr := strings.TrimSpace(output[start : start+end])
			if json.Valid([]byte(jsonStr)) {
				return jsonStr
			}
		}
	}

	// Try to find JSON object with { and }
	start := strings.Index(output, "{")
	if start >= 0 {
		end := strings.LastIndex(output, "}")
		if end > start {
			jsonStr := strings.TrimSpace(output[start : end+1])
			if json.Valid([]byte(jsonStr)) {
				return jsonStr
			}
		}
	}

	// If all else fails, return the original output
	return output
}

// cleanOutput removes ANSI escape codes and control characters from LLM output
func (d *QueryDataSource) cleanOutput(output string) string {
	// Remove ANSI escape sequences (like \x1b[?25h, \x1b[?2004h, etc.)
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\[[?][0-9]+[hl]`)
	cleaned := ansiRegex.ReplaceAllString(output, "")

	// Trim leading and trailing whitespace
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}
