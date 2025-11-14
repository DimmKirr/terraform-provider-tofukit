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

### Resource URI Linking System

The provider implements a `tofukit://` URI scheme that enables cross-resource references in prompts and instructions. Resources can reference other resources (features, files, stacks, kits) using URIs, and the provider automatically resolves them and provides metadata to Claude.

**URI Format:**
```
tofukit://TYPE/NAME              # Simple resources
tofukit://kit/SUBTYPE/NAME       # Kit resources
```

**Examples:**
- `tofukit://feature/logging` - Reference a feature resource
- `tofukit://file/gitignore` - Reference a file resource
- `tofukit://stack/python-app` - Reference a stack resource
- `tofukit://kit/language/python` - Reference a language kit
- `tofukit://kit/framework/click` - Reference a framework kit

**How It Works:**

1. **`.link` Attribute** - All resources have a computed `.link` attribute containing their URI:
   ```hcl
   resource "tofukit_feature" "logging" {
     name = "structured-logging"
     # ...
   }

   # Access via: tofukit_feature.logging.link
   # Value: "tofukit://feature/structured-logging"
   ```

2. **URI Scanning** (`internal/uri/scanner.go`) - Scans all string fields (prompts, instructions, constraints, descriptions) for URIs using regex pattern

3. **Registry Building** (`internal/uri/registry_builder.go`) - Resolves found URIs and builds a resource registry with metadata for each referenced resource

4. **Metadata Extraction** (`internal/uri/metadata_*.go`) - Each resource type has a metadata extractor that returns basic info (type, name, subtype)

5. **Claude Integration** - The resource registry is passed to Claude in the prompt's `resource_registry` field, and the system prompt is enhanced with URI reference instructions

**Usage Example:**
```hcl
resource "tofukit_feature" "logging" {
  name   = "structured-logging"
  prompt = "Add structured logging capability"
  # ...
}

resource "tofukit_project" "app" {
  name = "my-app"

  requirements = [{
    name = "Setup"
    instructions = [{
      # Reference another feature using its URI
      prompt = "Use ${tofukit_feature.logging.link} for application logging"
    }]
  }]
}
```

**Key Implementation Files:**
- `/internal/uri/scanner.go` - URI extraction and parsing
- `/internal/uri/registry_builder.go` - Registry construction from URIs
- `/internal/uri/metadata_*.go` - Metadata extractors for each resource type
- `/internal/resources/project.go:collectStringFields()` - Collects all string fields for scanning
- `/internal/llm/claude/prompt_types.go` - Integration with Claude prompt structure

**URI Registry Flow in Project:**
1. `collectStringFields()` gathers all prompts, instructions, constraints, descriptions from project
2. `URIScanner.ExtractURIs()` finds all `tofukit://` URIs in the fields
3. `RegistryBuilder.BuildRegistry()` resolves each URI to metadata via global registry
4. Resource registry added to `outputData["_resource_registry"]`
5. `BuildProjectPrompt()` detects registry and enhances system prompt with URI usage instructions
6. Claude receives complete context about referenced resources

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
├── uri/              # **NEW** Resource URI linking system
│   ├── scanner.go    # URI extraction and parsing
│   ├── registry_builder.go # Builds resource registry from URIs
│   └── metadata_*.go # Metadata extractors for each resource type
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

**Kit Verification Enforcement** (lines 4145-4246):
- `CollectKitVerifications()` - Extracts verification commands from kit requirements for enforcement
  - Iterates through all kits in outputData map
  - Handles both `"verification"` (singular, from registry) and `"verifications"` (plural, from state)
  - Supports both `map[string]interface{}` (from JSON) and `map[string]string` (from registry)
  - Returns verifications with pseudo-paths: `kit:{kitName}:{reqName}:{idx}`
- Integration in `executeClaudeCode()` (lines 2340-2360):
  - Collects file verifications from merged files
  - Collects kit verifications via `CollectKitVerifications()`
  - Merges both into `allVerifications` array
  - Passes to `ExecuteWithPromptJSON()` for enforcement
  - Failed verifications trigger retries; after max retries, provider returns error

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

**Pattern for test configurations (Go 1.16+ embed):**

Tests use the `//go:embed` directive to load Terraform configurations from `testdata/configs/*.tofu` at compile time:

```go
// In test/helpers_test.go
import _ "embed"

//go:embed testdata/configs/file-resource-basic.tofu
var ConfigFileResourceBasic string

// In test files
func TestResourceFile_CreateGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceFile_CreateGeneratePromptSuccess")

	// Use embedded config from testdata
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(ConfigFileResourceBasic), 0644)
	require.NoError(t, err)

	// ... rest of test
}
```

**Benefits:**
- **Maintainability**: Configs in separate `.tofu` files instead of inline heredocs (40-70% shorter tests)
- **Reusability**: Same config can be used across multiple tests
- **Validation**: Configs can be validated independently with `terraform fmt` and `validate-configs.sh`
- **Readability**: Test logic clearly separated from configuration data
- **Compile-time loading**: No runtime file I/O overhead

**Config file structure:**
```
test/
├── testdata/
│   ├── configs/
│   │   ├── file-resource-basic.tofu          # Basic file resource test
│   │   ├── file-resource-drift.tofu          # Drift detection test
│   │   ├── feature-inline-basic.tofu         # Inline feature with files
│   │   ├── feature-precedence.tofu           # Feature vs project precedence
│   │   ├── feature-multiple-merge.tofu       # Multiple features merge
│   │   └── ... (other configs)
│   └── validate-configs.sh                   # Terraform validation script
├── helpers_test.go                           # Embed declarations
└── resource_*_test.go                        # Test implementations
```

**Validation script:**
```bash
# From test/testdata directory
./validate-configs.sh  # Runs terraform fmt -check on all .tofu files
```

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
  - `name` (required) - Unique name for the feature
  - `description` (optional) - Description of the feature
  - `requirements` (required, array) - At least one requirement defining what the feature does
    - Each requirement has: `name` (required), `instructions` (optional array with `prompt` and `constraints`), `verifications` (optional array)
  - `files` (optional) - Map of files keyed by path
  - `kits` (optional) - Kit dependencies for this feature
  - `verifications` (optional) - Feature-level verification commands
- **Usage patterns**:
  - **Standalone resource**: Define once, reference in multiple projects via `tofukit_feature.name`
  - **Inline definition**: Define directly in project's `features` map
- **Registry behavior**: Features save themselves to registry on Create/Update, enabling cross-project reuse
- **Validation**: Must have at least one `requirement`
- **Example**:
  ```hcl
  resource "tofukit_feature" "logging" {
    name = "structured-logging"
    description = "Structured logging capability"

    requirements = [{
      name = "Logging Setup"
      instructions = [{
        prompt = "Add structured logging capability"
        constraints = ["Do not use print statements"]
      }]
      verifications = [{
        command = "python -m pytest tests/test_logger.py"
      }]
    }]

    files = {
      "logger.py" = {
        content = "# Logging configuration\n"
      }
    }
  }

  resource "tofukit_project" "app" {
    features = {
      "logging" = tofukit_feature.logging  # Reference
    }
  }
  ```

**File Resource** (`tofukit_file`)
- Defines individual reusable files that can be shared across projects
- Saved to registry with the `name` attribute as the file path
- Supports both static `content` and dynamic `instructions` (mutually exclusive)
- Can include `verifications` for validation
- Referenced in project/stack `files` maps: `files = { "path" = tofukit_file.name }`
- **Example use cases**:
  - Share common files (.gitignore, LICENSE) across multiple projects
  - Break down large stack definitions into manageable file resources
  - Create libraries of reusable file templates
- **Validation**:
  - Must have either `content` OR `instructions`, but not both
  - `name` must be a filesystem-compliant path following these rules:
    - No spaces (use hyphens or underscores instead)
    - No invalid characters: `< > : " | ? * \`
    - No leading/trailing whitespace
    - No trailing dots
    - No Windows reserved names (CON, PRN, AUX, NUL, COM1-9, LPT1-9)
    - Each path component must not exceed 255 characters
    - Use forward slashes `/` for directory separators
  - **Example valid names**: `README.md`, `.gitignore`, `src/main.go`, `my-file_v2.txt`
  - **Example invalid names**: `hello world.txt`, `file<name>.txt`, `CON`, `file.`

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

### Drift Detection

The provider implements granular per-file drift detection following the `hashicorp/local_file` pattern.

**How It Works:**

1. **Hash Storage** (Create/Update):
   - After Claude execution, computes SHA256 hashes for all files
   - Stores `content_hash` (specification), `file_hash` (actual file), `file_modtime` (optimization)

2. **Drift Detection** (Read):
   - Called during `tofu plan`
   - Checks modification time first (fast)
   - If changed, computes current hash and compares with stored `file_hash`
   - Sets `drift_detected = true` if mismatch

3. **Drift Restoration** (Update):
   - Triggered when `drift_detected` changes to `true`
   - Adds drift warning instructions to affected files
   - Claude receives: "⚠️ DRIFT DETECTED: File X was modified, restore to spec"
   - After successful execution, clears drift flags

**Performance Optimization:**
- Uses modification time (`mtime`) to skip hash computation for unchanged files
- Scales efficiently to 100+ file projects

**Example:**
```
1. User manually edits .gitignore
2. tofu plan → detects drift, shows "drift_detected: false → true"
3. tofu apply → Claude restores .gitignore to specification
```

**State Fields:**
- `drift_detected` (bool): True if any files drifted
- `drifted_files` (list): Paths of files that changed outside Terraform
- Per-file: `content_hash`, `file_hash`, `file_modtime` (computed)

### Important Implementation Details

**Empty Project Validation**
- Projects MUST specify at least one of: `files`, `kits`, `stack`, `features`, or `requirements`
- Prevents ambiguous/unpredictable Claude execution
- Validation occurs in both `Create()` and `Update()` at lines 291-318, 588-616

**Verification Enforcement System**
- Configured via `max_retries` provider setting (default: 3)
- **Two types of verifications**:
  - **File verifications**: Commands defined in `file { verifications = [...] }` blocks
  - **Kit verifications**: Commands defined in `tofukit_tool { requirements { verifications = [...] } }` blocks
- **Enforcement flow**:
  1. Provider collects file verifications from merged files
  2. Provider collects kit verifications from all kits in project (via `CollectKitVerifications()`)
  3. All verifications merged and passed to `ExecuteWithPromptJSON()`
  4. After Claude execution, provider runs ALL verification commands
  5. If verifications fail, Claude receives error output and retries (up to `max_retries`)
  6. After max retries exhausted, provider returns error causing Terraform apply to fail
- **Kit verifications ensure declared dependencies** (languages, tools, frameworks) are actually available before allowing project creation
- **Error message format**: `❌ File 'kit:{kitName}:{reqName}:{idx}': Command '{command}' failed with error: {error}`

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
