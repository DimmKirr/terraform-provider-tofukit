# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TofuKit is a custom Terraform/OpenTofu provider that enables declarative management of LLM-generated software components. It uses Claude (or other LLMs) to generate and manage files based on structured specifications. 

### Potential Usages of this project:
1. Store `project.tofu` in an opensource repo to show what DNA it has. 
   1. Helps recreating a similar from scratch if needed (templating)
   2. Helps understand components that it has   
2. Encapsulate kits and requirements into packagable modules and communicate their dependencies clearly, for example:
   1. **Stack**: Python Click app
      1. **Kit**: Python 3.12
      2. **Kit**: Click 8.1.3
      3. **Kit**: uv 0.9.5

## Build and Development Commands

### Building
```bash
task build              # Build the provider binary
task install            # Build and install provider locally to ~/.terraform.d/plugins

# Don't build directly, use task
```

**Build Output Location**:
Binaries are built to platform-specific directories to prevent confusion when switching between environments:
```
bin/linux_arm64/terraform-provider-tofukit_v0.1.0    # Linux builds (container)
bin/darwin_arm64/terraform-provider-tofukit_v0.1.0   # macOS builds (host)
```

### Testing
```bash
# Run all tests
task test

# Note: Individual test tasks are not defined. Run specific tests with:
# go test ./test -v -run TestProjectFileNestedCreateSuccess

# Don't run tests directly if task exists, use task

```

### Provider Installation Location
After `task install`, the provider will be located in:
```
~/.terraform.d/plugins/registry.terraform.io/DimmKirr/tofukit/0.1.0/<OS>_<ARCH>/terraform-provider-tofukit_v0.1.0
```
It will be picked up by Terraform/OpenTofu becasue environment contains this
```shell
export TF_CLI_CONFIG_FILE=${HOME}/.terraformrc
```

And .terraformrc contains
```terraform
provider_installation {
  # add a filesystem mirror so init can “install” locally
  filesystem_mirror {
    path = "/Users/dmitry/.terraform.d/plugins"
  }


  direct {}
}


```

## Architecture Overview

### Core Concept: Diff-Based File Operations

The provider computes explicit file operations (add/remove/rename/modify/unchanged) by comparing Terraform state with the plan, then sends these operations to Claude as structured instructions. This ensures Claude knows exactly what changed and what to do.

**Key Flow:**
1. User updates Terraform config (e.g., renames a file from `hello.txt` to `hello2.txt`)
2. Provider's `Update()` method compares old state vs new plan
3. `ComputeFileOperations()` detects: "rename hello.txt → hello2.txt"
4. `EnrichFilesWithInstructions()` adds action-specific instruction: "Rename file from 'hello.txt' to 'hello2.txt'"
5. Claude receives the enriched file spec with explicit rename instruction
6. Claude executes: moves file, removes old file, cleans up empty directories

### Directory Structure

```
internal/
├── provider/          # Terraform provider implementation
│   ├── provider.go   # Provider schema, configuration, LLM adapter setup
│   └── llm_adapter.go # Multi-LLM abstraction layer
├── resources/         # Terraform resources
│   ├── project.go    # Main project resource (Create/Read/Update/Delete)
│   ├── stack.go      # Stack resource (reusable file collections)
│   ├── feature.go    # Feature resource (reusable capability bundles)
│   └── base_component.go # Shared component logic
├── datasources/       # Terraform data sources
│   └── query.go      # Execute Claude queries for ad-hoc tasks
├── llm/              # LLM integrations
│   ├── interface.go  # Common LLM interface
│   ├── claude/       # Claude CLI integration (default)
│   ├── openai/       # OpenAI API integration
│   └── gemini/       # Google Gemini API integration
├── files/            # File operation logic
│   ├── operations.go # **CRITICAL** Diff computation and instruction generation
│   ├── merger.go     # Merges files from stacks, kits, and project
│   └── manager.go    # Physical file system operations
├── schemas/          # Terraform schema definitions
│   ├── common.go     # Shared schemas (File, Instruction, Verification)
│   └── kits.go       # Kit schemas (language, framework configs)
└── registry/         # Component registry for cross-resource dependencies
    └── registry.go   # Thread-safe registry for stacks and components
```

### Key Files to Understand

#### `/internal/files/operations.go` - **The Most Important File**
This file is the heart of the diff-based system:

- `ComputeFileOperations(oldFiles, newFiles)` - Compares old/new file lists, detects renames via content matching
- `EnrichFilesWithInstructions(oldFiles, newFiles)` - Adds action-based instructions to file models
- `generateInstructionForAction(op)` - Creates action-specific instructions:
  - **ADD**: "Create new file 'X' with exact content"
  - **MODIFY**: "Update existing file 'X' with new content"
  - **REMOVE**: "Delete file 'X' from the project"
  - **RENAME**: "Rename file from 'X' to 'Y'" (preserves content)
  - **UNCHANGED**: "File 'X' should remain unchanged"
- `mergeInstructions(op, userInstructions)` - Smart merging:
  - REMOVE/RENAME: Use only action instruction (user instruction doesn't apply)
  - ADD/MODIFY: Combine action + user instructions
  - UNCHANGED: Keep user instructions if any

#### `/internal/resources/project.go`
Main Terraform resource managing project lifecycle:

- `Create()` - Initial project creation (all files are "add" operations)
- `Update()` - **Critical** Computes diff, enriches files, executes Claude
  - Line 588-606: Validates project has content (files/kits/stack/requirements)
  - Line 618-636: Computes file operations and enriches with instructions
  - Line 638-676: Executes Claude with enriched files
- `Read()` - Refreshes state from filesystem
- `Delete()` - Removes generated files
- `collectAndMergeFiles()` - Merges files from stacks, kits, features, and project (with precedence)

**Feature Processing Functions** (lines ~1940-2160):
- `parseFeature()` - Extracts FeatureModel from Dynamic attribute (handles both inline and resource references)
- `collectFeatureFiles()` - Gathers all files from features for merging
- `convertFeaturesToRequirements()` - Transforms features into requirements for Claude execution
- `collectFeatureKitIDs()` - Extracts and deduplicates kit dependencies from features

**IMPORTANT: Dynamic Type Handling**
- The `features` field uses `types.Dynamic` instead of `types.Map` to comply with Terraform's framework restrictions
- Dynamic types cannot be nested inside collection types (map/list)
- All feature-processing functions must first extract the underlying value: `data.Features.UnderlyingValue()`, then cast to `types.Map`
- This pattern is critical for avoiding "Dynamic types inside collections" schema validation errors

#### `/internal/llm/claude/prompt_types.go`
Structures the JSON prompt sent to Claude:

- `ProjectPromptRequest` - Contains project info, files array with instructions, kits, requirements
- `buildOutputData()` - Constructs the structured JSON Claude receives
- Files array includes `instructions` field with action-based guidance

### Multi-LLM Support

The provider supports multiple LLM backends:

**Claude (Default)** - Uses Claude CLI binary (`/usr/bin/claude`)
- No API key needed (uses CLI authentication)
- Configured via `claude_home_directory` (default: `~/.claude`)
- Best integration, most tested

**OpenAI** - Uses OpenAI API
- Requires `llm = "openai"` and `api_key = "sk-..."`
- Uses `gpt-4` model by default

**Gemini** - Uses Google Gemini API
- Requires `llm = "gemini"` and `api_key = "..."`
- Uses `gemini-1.5-pro` model by default

Configured in provider block:
```hcl
provider "tofukit" {
  llm = "claude"  # or "openai", "gemini"
  api_key = "..."  # Required for openai/gemini
  debug = true     # Enable debug output
  dangerously_skip_permissions = true  # Default: skip Claude CLI permission prompts
  max_retries = 3  # Verification retry attempts (default: 3)
}
```

**Security Note: Claude CLI Permissions**
- `dangerously_skip_permissions = true` (default): Claude CLI skips all permission prompts and has full filesystem access. Use for non-interactive automation.
- `dangerously_skip_permissions = false`: Claude CLI will prompt for permissions before file operations. More secure but requires interactive mode (may hang in automated environments).
- The `--add-dir` flag is always passed to restrict access to the output directory, but it is ignored when `--dangerously-skip-permissions` is enabled.

### Test Structure and Mock Setup

Tests are organized by operation type with mock file setup for tests that need existing state:

**Pattern for tests with existing files:**
```go
// MOCK SETUP: Create existing file structure using Go filesystem operations
projectPath := filepath.Join(testDir, "output")
demoDir := filepath.Join(projectPath, "demo")
err := os.MkdirAll(demoDir, 0755)
require.NoError(t, err)

helloPath := filepath.Join(demoDir, "hello.txt")
err = os.WriteFile(helloPath, []byte("hello from demo\n"), 0644)
require.NoError(t, err)
```

This approach:
- Creates files directly using `os.MkdirAll()` + `os.WriteFile()`
- No Claude execution needed for setup
- Fast and deterministic
- Tests focus on the specific operation being tested

### Resource Schemas

**Project Resource** (`tofukit_project`)
- `files` - Map of files keyed by path with optional `instructions` and `verifications`
- `kits` - Language/framework configurations (merged into files)
- `stack` - Reference to reusable stack resource
- `features` - Map of feature definitions (inline or resource references) keyed by feature name
- `requirements` - List of requirements with instructions and verifications
- **Validation**: Must have at least one of: files, kits, stack, features, or requirements

**Stack Resource** (`tofukit_stack`)
- Reusable collections of files
- Saved to registry for use by projects
- Same `files` structure as projects

**Feature Resource** (`tofukit_feature`) - **NEW**
- Bundles capabilities (files + kits + verifications) into reusable components
- Can be used as standalone registry entries or inline definitions in projects
- **Key properties**:
  - `prompt` - What the feature does (LLM-facing requirement)
  - `constraints` - Implementation constraints (what NOT to do)
  - `files` - Map of files keyed by path
  - `kits` - Kit dependencies for this feature
  - `verifications` - Verification commands for this feature
- **Usage patterns**:
  - **Standalone resource**: Define once, reference in multiple projects via `tofukit_feature.name`
  - **Inline definition**: Define directly in project's `features` map
- **Registry behavior**: Features save themselves to registry on Create/Update, enabling cross-project reuse
- **Validation**: Must have at least one of: files, kits, or verifications
- **Example**:
  ```hcl
  resource "tofukit_feature" "logging" {
    name   = "structured-logging"
    prompt = "Add structured logging capability"

    files = {
      "logger.py" = {
        content = "# Logging configuration\n"
      }
    }

    verifications = [{
      command = "python -m pytest tests/test_logger.py"
    }]
  }

  resource "tofukit_project" "app" {
    features = {
      "logging" = tofukit_feature.logging  # Reference
    }
  }
  ```

**File Resource** (`tofukit_file`)
- Defines individual reusable files that can be shared across projects
- Saved to registry with unique `name` identifier
- Supports both static `content` and dynamic `instructions` (mutually exclusive)
- Can include `verifications` for validation
- Referenced in project/stack `files` maps: `files = { "path" = tofukit_file.name }`
- **Example use cases**:
  - Share common files (.gitignore, LICENSE) across multiple projects
  - Break down large stack definitions into manageable file resources
  - Create libraries of reusable file templates
- **Validation**: Must have either `content` OR `instructions`, but not both

**Query Data Source** (`tofukit_query`)
- Execute ad-hoc Claude queries
- Returns output directly without file management

### File Merge Precedence Hierarchy

**IMPORTANT: File Precedence Order** (internal/files/merger.go)

Files are merged using a precedence hierarchy where higher layers override lower layers:

```
1. Stack files        (SourceStack)      - Lowest precedence
2. Kit files          (SourceProject)    - From stack and project kits
3. Feature files      (SourceFeature)    - NEW: Feature-defined files
4. Project files      (SourceProject)    - Highest precedence
```

**How it works:**
- If the same file path appears in multiple layers, the highest precedence wins
- Stack provides base files
- Kits add/override with language/framework-specific files
- Features add/override with capability-specific files
- Project always has final say (can customize anything)

**Example:**
```hcl
resource "tofukit_stack" "base" {
  files = {
    "config.txt" = { content = "From stack\n" }
  }
}

resource "tofukit_project" "app" {
  stack = tofukit_stack.base.id

  features = {
    "custom_config" = {
      prompt = "Customize configuration"
      files = {
        "config.txt" = { content = "From feature\n" }  # Overrides stack
      }
    }
  }

  files = {
    "config.txt" = { content = "From project\n" }  # Overrides feature AND stack
  }
}
# Result: config.txt contains "From project\n"
```

**Key Implementation:**
- `internal/files/merger.go` defines `FileSource` constants
- `collectAndMergeFiles()` in project.go adds files in precedence order
- Features were added as a new layer between kits and project files

### File Instruction Flow

1. **User Config**: User may add `instruction {}` blocks to files for generation guidance
2. **Diff Detection**: Provider computes what changed (add/remove/rename/modify)
3. **Instruction Enrichment**: Action-based instructions are merged with user instructions
4. **Claude Execution**: Enriched files sent to Claude in JSON prompt
5. **State Storage**: Original user config (without action instructions) stored in state
6. **Next Apply**: Instructions recomputed fresh from new diff

**Result**: Action-based instructions are ephemeral (computed each time), preventing state drift while giving Claude explicit guidance.

### Important Implementation Details

**Empty Project Validation**
- Projects MUST specify at least one of: `files`, `kits`, `stack`, `features`, or `requirements`
- Prevents ambiguous/unpredictable Claude execution
- Validation occurs in both `Create()` and `Update()` at lines 291-318, 588-616

**Verification Retry System**
- Configured via `max_retries` provider setting (default: 3)
- If verification fails, Claude receives error output and retries
- Continues until verification passes or max retries reached

**File Hash Tracking**
- `file_hash` - Hash of file specifications (detects config changes)
- `output_hash` - Hash of actual filesystem output (detects drift)
- `prompt_hash` - Hash of entire prompt (includes files, kits, requirements, system prompt)

**Registry Pattern**
- Thread-safe registry (`internal/registry/`) stores stacks, files, features, and components
- Enables cross-resource dependencies (projects can reference stacks, files, and features)
- Resources save themselves to registry on Create/Update
- Resources remove themselves on Delete
- Resource-specific methods:
  - File resources: SetFile/GetFile/RemoveFile
  - Feature resources: SetFeature/GetFeature/RemoveFeature
  - Stack resources: SetStack/GetStack/RemoveStack

## Common Pitfalls

1. **HEREDOC vs String Literals in Tests**
   - Tests should use `content = "hello\n"` not `content = <<-EOF\nhello\nEOF`
   - HEREDOC can introduce unwanted characters (especially `!` gets escaped as `\!`)

2. **Test Caching Issues**
   - Run `go clean -testcache` if tests show stale results
   - Test directory names include random IDs to avoid conflicts

3. **Provider Installation**
   - Always run `task install` after code changes
   - Provider must be in correct plugin directory structure
   - On macOS, binary is code-signed automatically

4. **State Persistence**
   - Action-based instructions are NOT stored in state (they're ephemeral)
   - Only original user config is persisted
   - Instructions recomputed fresh on each apply from diff

5. **File Operations**
   - Rename detection uses content matching for static files
   - Rename detection uses instruction matching for generated files
   - Always check `/internal/files/operations.go` for operation logic

6. **Dynamic Type Usage for Features**
   - The `features` field in ProjectModelFinal MUST be `types.Dynamic`, not `types.Map`
   - Terraform framework does not allow dynamic types nested inside collections
   - When processing features, always extract underlying value first: `data.Features.UnderlyingValue()`
   - Then cast to `types.Map` before iterating: `featuresMap, ok := underlyingVal.(types.Map)`
   - This pattern prevents "Dynamic types inside collections" schema validation errors

## Testing Philosophy

- **Granular Tests**: One test per operation (create, add, remove, rename, etc.)
- **Mock Setup**: Use `os.MkdirAll()` + `os.WriteFile()` for existing files
- **Descriptive Names**: All tests end with "Success" (e.g., `TestProjectFileNestedCreateSuccess`)
- **Independent**: Each test runs in isolated directory (`test-output/TestName-<random>/`)
- **Preserved Output**: Test directories kept for inspection (set `CLEANUP_TEST_OUTPUT=true` to remove)

## Project-Specific Conventions

- **File operations** are computed by comparing `oldFiles` vs `newFiles` arrays
- **Instructions** follow a precedence model: action instructions override user instructions for REMOVE/RENAME
- **Claude execution** is always performed (no dry-run mode in current version)
- **Debug mode** (`debug = true`) writes execution metadata to `output/.debug/` directory
- **Trailing newlines** must be explicit in file content strings (use `\n` at end)
