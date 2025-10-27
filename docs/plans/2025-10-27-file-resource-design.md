# TofuKit File Resource Design

**Date**: 2025-10-27
**Status**: Design Phase
**Author**: Design discussion with user

## Problem Statement

Large stack definitions (like `tofukit-stack-go-viper-cobra-pterm/kit.tofu` at 797 lines) become unwieldy when they contain many files with long instruction prompts. Additionally, common files (like `.gitignore`, `main.go` templates) cannot be easily shared across multiple projects without duplicating entire stack definitions.

**Current limitations**:
1. Files must be defined inline within `files = {...}` maps, making large stacks hard to navigate
2. No way to share individual files across projects without creating wrapper stacks
3. No granular reusability - must group files into stacks even when atomic file sharing is desired

## Goals

1. **Granular reusability**: Enable sharing individual files across projects
2. **Cleaner organization**: Break up large file definitions into separate resource blocks
3. **Consistent API**: Maintain the existing `files = {...}` map interface
4. **Natural composition**: Files compose the same way stacks and kits do today

## Non-Goals

1. Replace inline file definitions (both patterns must coexist)
2. Change the existing `files` map schema
3. Add file lifecycle management beyond what projects already provide
4. Create a complex file dependency system

## Design

### High-Level Architecture

Add `tofukit_file` as a new resource type that follows the same registry pattern as `tofukit_stack`:

```
tofukit_file (resource)
    ↓ (registers in global registry)
Registry
    ↓ (referenced in projects)
tofukit_project.files = {
    "path" = tofukit_file.name  // Resource reference
    "path" = {content = "..."}   // Inline definition (still supported)
}
```

### Resource Schema

**Resource**: `tofukit_file`

```hcl
resource "tofukit_file" "main_go" {
  name        = "go-main-minimal"        # Required: Registry identifier
  description = "Minimal Go main.go"    # Optional: Documentation

  # File specification (mutually exclusive: content OR instructions)
  content = "package main\n..."         # Optional: Static content

  instructions = [{                     # Optional: Dynamic generation
    prompt      = "Create main.go..."
    constraints = ["NO imports", ...]
  }]

  verifications = [{                    # Optional: Validation commands
    command = "go build ..."
    expect  = "success"
  }]
}
```

**Attributes**:
- `name` (string, required) - Unique identifier for registry, used for lookup
- `description` (string, optional) - Human-readable documentation
- `content` (string, optional) - Static file content
- `instructions` (list[Instruction], optional) - Dynamic generation instructions
- `verifications` (list[Verification], optional) - Post-generation validation
- `id` (string, computed) - Terraform resource ID

**Validation**:
- Either `content` OR `instructions` must be specified (mutually exclusive)
- `verifications` can be used with either content or instructions

### Usage Patterns

**Pattern 1: Reference file resources in projects**
```hcl
resource "tofukit_file" "gitignore" {
  name = "standard-gitignore"
  content = "bin/\n*.exe\n*.so\n"
}

resource "tofukit_file" "main_go" {
  name = "go-main-minimal"
  instructions = [{
    prompt = "Create minimal main.go..."
  }]
}

resource "tofukit_project" "myapp" {
  files = {
    ".gitignore" = tofukit_file.gitignore
    "main.go"    = tofukit_file.main_go
    "custom.go"  = {content = "// custom"}  # Inline still works
  }
}
```

**Pattern 2: Override path from file resource**
```hcl
resource "tofukit_project" "myapp" {
  files = {
    # Use file resource but place at different path
    "cmd/main.go" = tofukit_file.main_go
  }
}
```

**Pattern 3: Share across projects**
```hcl
resource "tofukit_file" "common_gitignore" {
  name = "standard-gitignore"
  content = "bin/\n*.exe\n"
}

resource "tofukit_project" "app1" {
  files = {".gitignore" = tofukit_file.common_gitignore}
}

resource "tofukit_project" "app2" {
  files = {".gitignore" = tofukit_file.common_gitignore}
}
```

### File Merge Behavior

Files are merged in precedence order (existing logic in `collectAndMergeFiles()`):

1. Stack files (lowest precedence)
2. Kit files
3. **File resources** (new layer - higher than kits)
4. Inline project files (highest precedence - can override anything)

**Example precedence**:
```hcl
resource "tofukit_stack" "base" {
  files = {"main.go" = {content = "stack version"}}
}

resource "tofukit_file" "main" {
  name = "main-override"
  content = "file resource version"
}

resource "tofukit_project" "app" {
  stack = tofukit_stack.base
  files = {
    "main.go" = tofukit_file.main  # Overrides stack
    "other.go" = {content = "inline"}  # Highest precedence
  }
}
```

Result: `main.go` comes from file resource (overrides stack), `other.go` is inline.

### Registry Integration

The file resource uses the existing registry pattern (`internal/registry/registry.go`):

**New registry methods**:
```go
// Add to internal/registry/registry.go
func SaveFile(name string, spec *FileSpec) error
func GetFile(name string) (*FileSpec, error)
func RemoveFile(name string) error
```

**Resource lifecycle**:
- `Create()` - Saves file spec to registry via `SaveFile()`
- `Read()` - Retrieves from registry to refresh state
- `Update()` - Updates registry entry
- `Delete()` - Removes from registry via `RemoveFile()`

## Implementation Plan

### Phase 1: Schema and Data Structures

**Files to create**:
- `internal/resources/file.go` - New file resource implementation

**Files to modify**:
- `internal/schemas/common.go` - Ensure FileSpec schema is reusable
- `internal/registry/registry.go` - Add file storage methods

**Tasks**:
1. Define `FileResourceModel` struct matching inline file schema
2. Implement file resource schema with validation (content XOR instructions)
3. Add registry methods: `SaveFile()`, `GetFile()`, `RemoveFile()`
4. Add thread-safe storage in registry (map[string]FileSpec)

### Phase 2: Resource CRUD Operations

**Files to modify**:
- `internal/resources/file.go` - Implement lifecycle methods

**Tasks**:
1. Implement `Create()` - Validate spec, save to registry
2. Implement `Read()` - Fetch from registry, refresh state
3. Implement `Update()` - Update registry entry
4. Implement `Delete()` - Remove from registry
5. Generate unique resource ID (can use name as basis)

### Phase 3: Project Integration

**Files to modify**:
- `internal/resources/project.go` - Update file collection logic
- `internal/files/merger.go` - Add file resource layer to merge logic

**Tasks**:
1. Update `collectAndMergeFiles()` to handle file resources in the map
2. Detect when map value is a file resource vs inline spec
3. Insert file resource layer in merge precedence (after kits, before inline)
4. Update file operation computation to work with file resources

### Phase 4: Provider Registration

**Files to modify**:
- `internal/provider/provider.go` - Register new resource

**Tasks**:
1. Add `file.go` resource to provider's resource registry
2. Ensure schema validation works end-to-end

### Phase 5: Testing

**Test files to create**:
- `test/file_resource_test.go` - File resource CRUD tests
- `test/file_resource_project_test.go` - Integration with projects

**Test scenarios**:
1. Create file resource with static content
2. Create file resource with instructions
3. Reference file resource in project
4. Mix file resources and inline files in same project
5. Share file resource across multiple projects
6. Override file resource with inline definition (precedence)
7. Update file resource and verify project updates
8. Delete file resource and handle cleanup

**Test examples**:
```go
func TestFileResourceCreateSuccess(t *testing.T) {
  // Create tofukit_file with content
  // Verify registry contains file
  // Verify state is correct
}

func TestFileResourceInProjectSuccess(t *testing.T) {
  // Create tofukit_file resource
  // Create project referencing file resource
  // Verify file appears in project output
}

func TestFileResourcePrecedenceSuccess(t *testing.T) {
  // Create stack with file "main.go"
  // Create tofukit_file "main.go"
  // Create project with both stack and file resource
  // Verify file resource wins over stack
}
```

### Phase 6: Documentation

**Files to create/modify**:
- `examples/resources/file-resource/` - Example configurations
- `CLAUDE.md` - Update with file resource documentation

**Tasks**:
1. Create example showing file resource creation
2. Create example showing file resource reuse across projects
3. Update CLAUDE.md architecture section with file resource explanation
4. Document merge precedence rules

## Testing Strategy

### Unit Tests
- Registry operations (SaveFile, GetFile, RemoveFile)
- File resource validation (content XOR instructions)
- Schema validation

### Integration Tests
- File resource CRUD lifecycle
- Project references to file resources
- Merge precedence with stacks, kits, file resources, inline files
- Multi-project sharing of file resources

### Edge Cases
- File resource name conflicts
- Circular references (if possible)
- Empty file resources
- File resource with both content and instructions (should fail validation)
- Missing file resource reference (should fail plan)

## Open Questions

1. **Computed attributes**: Should file resources expose computed attributes like `file_hash`?
   - **Decision needed**: Likely yes, for consistency with inline files

2. **Path validation**: Should we validate that map keys are valid file paths?
   - **Decision needed**: Probably not - projects already handle this

3. **Registry scope**: Should file resources be scoped per-project or global?
   - **Current assumption**: Global (like stacks), as they're meant to be shared

4. **Terraform state**: Does the file resource need to store the full spec in state, or just metadata?
   - **Current assumption**: Full spec, so it can be used in projects without registry lookup

## Migration Path

**Backward compatibility**: All existing code continues to work unchanged. File resources are purely additive.

**Migration strategy for users**:
1. Identify commonly reused files in existing stacks
2. Extract them to `tofukit_file` resources
3. Reference file resources in projects
4. Gradually refactor large stacks to use file resources

**Example migration**:
```hcl
# Before: Large inline definition
resource "tofukit_project" "app" {
  files = {
    ".gitignore" = {
      content = "bin/\n*.exe\n..."  # 50 lines
    }
  }
}

# After: Extract to file resource
resource "tofukit_file" "gitignore" {
  name = "standard-gitignore"
  content = "bin/\n*.exe\n..."
}

resource "tofukit_project" "app" {
  files = {
    ".gitignore" = tofukit_file.gitignore
  }
}
```

## Success Metrics

1. Large stack files can be broken down into manageable file resources
2. Common files (gitignore, taskfiles) can be shared across projects without duplication
3. No breaking changes to existing code
4. Test coverage for all CRUD operations and integration scenarios
5. Documentation clearly explains when to use file resources vs inline files

## Alternatives Considered

### Alternative 1: Terraform `locals`
**Pros**: Works today, no code changes
**Cons**: Can't share across modules, no lifecycle management
**Decision**: Not sufficient for cross-project sharing

### Alternative 2: Terraform modules
**Pros**: Standard pattern, shareable via registry
**Cons**: Directory overhead, indirection through module calls
**Decision**: Could work but less ergonomic than native resources

### Alternative 3: Extend `tofukit_stack` to allow single-file stacks
**Pros**: No new resource type
**Cons**: Semantic mismatch (stack implies collection), doesn't improve organization
**Decision**: File resource is clearer and more granular

## Risks and Mitigation Strategies

### Risk 1: Schema Mismatch Between File Resource and Inline Files
**Risk**: File resource attributes might not perfectly match inline file schema, causing runtime errors or type mismatches when Terraform evaluates `files = {"path" = tofukit_file.x}`.

**Impact**: HIGH - Would break all file resource usage

**Mitigation**:
1. Use the EXACT same nested attribute schema from `common.FileSchema()` for the file resource
2. Create integration test that validates a file resource can be assigned to a project's files map
3. Add schema validation in project's `files` attribute to accept both inline and resource references
4. Run `terraform validate` in all test scenarios to catch schema mismatches early

**Double-check**:
```bash
# After implementation, verify schemas match
go test -v -run TestFileResourceSchemaCompatibility
```

### Risk 2: Registry Name Collisions
**Risk**: Multiple file resources with the same `name` could overwrite each other in the registry, causing unexpected behavior.

**Impact**: MEDIUM - Silent data corruption, wrong files in projects

**Mitigation**:
1. Registry.SaveFile() should check for existing name and return error if collision detected
2. Add uniqueness validation in file resource Create()
3. Include resource ID in error messages to help users identify conflicts
4. Document naming best practices in examples

**Double-check**:
```go
// Add test case
func TestFileResourceNameCollisionFailure(t *testing.T) {
    // Create two tofukit_file resources with same name
    // Verify second create fails with descriptive error
}
```

### Risk 3: Merge Precedence Confusion
**Risk**: Users might not understand when file resources override stacks, or when inline files override resources.

**Impact**: MEDIUM - User confusion, unexpected file content

**Mitigation**:
1. Document precedence clearly in CLAUDE.md with examples
2. Add debug logging showing merge order: "File 'X' from resource overrides stack version"
3. Create comprehensive test for all precedence combinations
4. Consider adding warning if same file comes from multiple sources

**Double-check**:
- Create test with stack, file resource, and inline file all defining "main.go"
- Verify inline wins, log messages show precedence chain

### Risk 4: Circular Dependencies
**Risk**: File resource A references project B which references file resource A (via dependencies).

**Impact**: LOW - Terraform handles dependency cycles, but could confuse users

**Mitigation**:
1. Rely on Terraform's built-in cycle detection
2. Document that file resources should be leaf nodes (no depends_on to projects)
3. Test cannot easily trigger this (Terraform would error during plan)

**Double-check**:
- Manual testing with circular depends_on
- Verify Terraform's error message is clear

### Risk 5: State Drift Detection
**Risk**: If file resource definition changes, projects using it need to detect the change and update.

**Impact**: HIGH - Stale content in projects

**Mitigation**:
1. File resource computes hash of its spec (content + instructions + verifications)
2. Project stores reference to file resource by ID + hash
3. When file resource updates, hash changes, triggering project update
4. Terraform's dependency tracking should handle this automatically via resource references

**Double-check**:
```go
func TestFileResourceUpdateTriggersProjectUpdate(t *testing.T) {
    // Create file resource with content "v1"
    // Create project referencing file resource
    // Update file resource to content "v2"
    // Verify project detects change and regenerates
}
```

### Risk 6: Instruction Enrichment Conflicts
**Risk**: File resources have instructions, but projects also add action-based instructions during diff. Could these conflict or duplicate?

**Impact**: MEDIUM - Confusing instructions sent to Claude

**Mitigation**:
1. File resource instructions are USER instructions (what to generate)
2. Action instructions are OPERATION instructions (add/modify/remove/rename)
3. These are merged by `mergeInstructions()` which already handles this
4. File resources follow same instruction model as inline files

**Double-check**:
- Test file resource with instructions being added to project
- Verify instructions are merged correctly (action + user)
- Check debug output shows both instruction types

### Risk 7: Performance Impact of Registry Lookups
**Risk**: Every project.Read() might query registry for file resources, slowing down operations.

**Impact**: LOW - Slight performance degradation

**Mitigation**:
1. File resources are resolved via Terraform's dependency graph, not manual registry lookup
2. Registry is just for state storage, not runtime resolution
3. In-memory registry is fast (map lookup)

**Double-check**:
- Performance test with 100 file resources
- Measure project.Read() time

### Risk 8: Delete Ordering Issues
**Risk**: Deleting a project before its file resource could leave orphaned state. Deleting file resource while project uses it could break project.

**Impact**: MEDIUM - State corruption or runtime errors

**Mitigation**:
1. Terraform's dependency graph handles delete ordering automatically
2. Projects reference file resources via `depends_on` implicitly through usage
3. Deleting file resource first will trigger project recreation (Terraform detects missing dependency)
4. Document proper destroy order in examples

**Double-check**:
```go
func TestFileResourceDeleteOrdering(t *testing.T) {
    // Create file resource + project
    // Try to delete file resource first
    // Verify Terraform handles gracefully
}
```

## Double-Check Procedures

### Pre-Implementation Checklist
- [ ] Read existing `stack.go` to understand registry pattern thoroughly
- [ ] Read existing `project.go` collectAndMergeFiles() to understand merge logic
- [ ] Read existing `files/operations.go` to understand diff computation
- [ ] Verify schema compatibility between FileSchema and planned file resource schema
- [ ] Document expected behavior for all edge cases before coding

### During Implementation Checklist

#### Phase 1: Schema and Data Structures
- [ ] File resource schema matches `FileSchema()` from common.go exactly
- [ ] Registry methods are thread-safe (use mutex)
- [ ] Registry SaveFile() checks for name collisions
- [ ] File resource validation prevents both content AND instructions
- [ ] Computed attributes (ID) are properly marked

#### Phase 2: Resource CRUD
- [ ] Create() saves to registry successfully
- [ ] Create() handles registry collision errors
- [ ] Read() refreshes from registry
- [ ] Read() handles missing registry entries gracefully
- [ ] Update() updates registry atomically
- [ ] Delete() removes from registry
- [ ] Delete() doesn't fail if registry entry already gone

#### Phase 3: Project Integration
- [ ] collectAndMergeFiles() detects file resource references in map
- [ ] File resources inserted in correct precedence order (after kits, before inline)
- [ ] File operations computation works with file resources
- [ ] Action instruction enrichment works with file resources
- [ ] Debug output shows when file comes from resource vs inline

#### Phase 4: Provider Registration
- [ ] File resource registered in provider.Resources()
- [ ] No import conflicts
- [ ] Schema validation runs end-to-end
- [ ] terraform validate passes with file resources

### Post-Implementation Verification

#### Unit Test Coverage
- [ ] File resource CRUD operations (all 4 methods)
- [ ] Registry collision detection
- [ ] Schema validation (content XOR instructions)
- [ ] Computed attribute generation

#### Integration Test Coverage
- [ ] File resource referenced in project
- [ ] File resource + inline files in same project
- [ ] File resource + stack in same project
- [ ] Precedence: inline > resource > kit > stack
- [ ] File resource update triggers project update
- [ ] Multiple projects share same file resource
- [ ] Delete ordering (resource before project, project before resource)

#### Edge Case Coverage
- [ ] Empty file resource (no content, no instructions) - should fail validation
- [ ] File resource with both content AND instructions - should fail validation
- [ ] Non-existent file resource reference - Terraform should error during plan
- [ ] File resource name collision - should fail on create
- [ ] Very long file resource name - should work (no limit unless we add one)

#### Manual Testing Checklist
- [ ] Run `task build && task install`
- [ ] Create simple example with one file resource + one project
- [ ] Run `tofu plan` - verify plan shows resource creation
- [ ] Run `tofu apply` - verify files created correctly
- [ ] Update file resource content
- [ ] Run `tofu plan` - verify project detects change
- [ ] Run `tofu apply` - verify project regenerates
- [ ] Run `tofu destroy` - verify clean deletion
- [ ] Check debug output for file source tracking

## Testing Strategy - Detailed Test Cases

### Tests to Add to `test/project_files_test.go`

Based on analysis of existing test structure, here are specific test cases to add:

#### Test 1: File Resource Basic CRUD
```go
// TestFileResourceCreateSuccess tests creating a file resource with static content
func TestFileResourceCreateSuccess(t *testing.T) {
    // Setup test directory
    testDir := createTestDirectory(t, "TestFileResourceCreateSuccess")

    // Create project.tofu with tofukit_file resource
    projectContent := `
resource "tofukit_file" "gitignore" {
  name = "standard-gitignore"
  description = "Standard Go .gitignore"

  content = "bin/\n*.exe\n*.dll\n"
}

resource "tofukit_project" "app" {
  name = "test-app"
  version = "1.0.0"

  files = {
    ".gitignore" = tofukit_file.gitignore
  }
}
`

    // Run init, plan, apply
    // Verify:
    // - tofukit_file resource created in state
    // - .gitignore exists with correct content
    // - State shows files map references file resource
}
```

#### Test 2: File Resource with Instructions
```go
// TestFileResourceInstructionsSuccess tests file resource with dynamic generation
func TestFileResourceInstructionsSuccess(t *testing.T) {
    // Create tofukit_file with instructions (not content)
    projectContent := `
resource "tofukit_file" "readme" {
  name = "standard-readme"

  instructions = [{
    prompt = "Create a README.md with project title and description"
    constraints = ["Keep it under 100 lines", "Use proper markdown"]
  }]

  verifications = [{
    command = "test -f README.md && echo 'found'"
    expect = "found"
  }]
}

resource "tofukit_project" "app" {
  name = "test-app"
  version = "1.0.0"

  files = {
    "README.md" = tofukit_file.readme
  }
}
`

    // Verify:
    // - README.md generated by Claude
    // - Verification passes
    // - Content follows instructions
}
```

#### Test 3: Multiple Projects Sharing File Resource
```go
// TestFileResourceSharedAcrossProjectsSuccess tests reusability
func TestFileResourceSharedAcrossProjectsSuccess(t *testing.T) {
    projectContent := `
resource "tofukit_file" "license" {
  name = "mit-license"
  content = "MIT License\nCopyright 2024\n"
}

resource "tofukit_project" "app1" {
  name = "app1"
  output_dir = "output/app1"
  files = {
    "LICENSE" = tofukit_file.license
  }
}

resource "tofukit_project" "app2" {
  name = "app2"
  output_dir = "output/app2"
  files = {
    "LICENSE" = tofukit_file.license
  }
}
`

    // Verify:
    // - output/app1/LICENSE exists with correct content
    // - output/app2/LICENSE exists with same content
    // - Both projects reference same file resource ID in state
}
```

#### Test 4: File Resource Precedence
```go
// TestFileResourcePrecedenceSuccess tests merge priority
func TestFileResourcePrecedenceSuccess(t *testing.T) {
    projectContent := `
resource "tofukit_stack" "base" {
  name = "base-stack"
  files = {
    "main.go" = {
      content = "// Stack version\n"
    }
  }
}

resource "tofukit_file" "main" {
  name = "main-override"
  content = "// File resource version\n"
}

resource "tofukit_project" "app" {
  name = "test-app"
  version = "1.0.0"

  stack = tofukit_stack.base

  files = {
    # File resource should override stack
    "main.go" = tofukit_file.main

    # Inline should override everything
    "other.go" = {
      content = "// Inline version\n"
    }
  }
}
`

    // Verify:
    // - main.go contains "File resource version" (not stack version)
    // - other.go contains "Inline version"
    // - Debug log shows precedence chain
}
```

#### Test 5: File Resource Update Triggers Project Update
```go
// TestFileResourceUpdateTriggersProjectUpdateSuccess tests state propagation
func TestFileResourceUpdateTriggersProjectUpdateSuccess(t *testing.T) {
    // Phase 1: Create file resource with content "v1"
    initialContent := `
resource "tofukit_file" "config" {
  name = "app-config"
  content = "version: v1\n"
}

resource "tofukit_project" "app" {
  name = "test-app"
  version = "1.0.0"
  files = {
    "config.yaml" = tofukit_file.config
  }
}
`

    // Apply initial config
    // Verify output/config.yaml contains "version: v1"

    // Phase 2: Update file resource to "v2"
    updatedContent := `
resource "tofukit_file" "config" {
  name = "app-config"
  content = "version: v2\n"  # CHANGED
}

resource "tofukit_project" "app" {
  name = "test-app"
  version = "1.0.0"
  files = {
    "config.yaml" = tofukit_file.config
  }
}
`

    // Run plan - should show tofukit_file update AND tofukit_project update
    // Run apply
    // Verify output/config.yaml now contains "version: v2"
}
```

#### Test 6: File Resource Name Collision
```go
// TestFileResourceNameCollisionFailure tests registry uniqueness
func TestFileResourceNameCollisionFailure(t *testing.T) {
    projectContent := `
resource "tofukit_file" "config1" {
  name = "duplicate-name"  # SAME NAME
  content = "config 1\n"
}

resource "tofukit_file" "config2" {
  name = "duplicate-name"  # SAME NAME
  content = "config 2\n"
}
`

    // Run plan - Terraform might catch this
    // Run apply - should fail with clear error about name collision
    // Error should mention both resource IDs
}
```

#### Test 7: File Resource with Both Content and Instructions (Should Fail)
```go
// TestFileResourceContentAndInstructionsFailure tests validation
func TestFileResourceContentAndInstructionsFailure(t *testing.T) {
    projectContent := `
resource "tofukit_file" "invalid" {
  name = "invalid-file"

  content = "static content"  # BOTH content

  instructions = [{           # AND instructions
    prompt = "generate something"
  }]
}
`

    // Run plan - should fail with validation error
    // Error should explain content and instructions are mutually exclusive
}
```

#### Test 8: File Resource Rename Detection
```go
// TestFileResourcePathOverrideSuccess tests using file at different path
func TestFileResourcePathOverrideSuccess(t *testing.T) {
    projectContent := `
resource "tofukit_file" "util" {
  name = "utility-functions"
  content = "func Helper() {}\n"
}

resource "tofukit_project" "app" {
  name = "test-app"
  version = "1.0.0"

  files = {
    # Use file resource at custom path
    "pkg/utils/helpers.go" = tofukit_file.util
  }
}
`

    // Verify:
    // - File created at pkg/utils/helpers.go (not at some default)
    // - Path comes from map key, not file resource
}
```

#### Test 9: File Resource Mixed with Inline Files
```go
// TestFileResourceMixedWithInlineSuccess tests both patterns together
func TestFileResourceMixedWithInlineSuccess(t *testing.T) {
    projectContent := `
resource "tofukit_file" "makefile" {
  name = "standard-makefile"
  content = "build:\n\tgo build\n"
}

resource "tofukit_project" "app" {
  name = "test-app"
  version = "1.0.0"

  files = {
    # File resource
    "Makefile" = tofukit_file.makefile

    # Inline files
    "main.go" = {
      content = "package main\n"
    }
    "README.md" = {
      instructions = [{
        prompt = "Create README"
      }]
    }
  }
}
`

    // Verify all three files created correctly
    // Verify no conflicts between resource and inline patterns
}
```

#### Test 10: File Resource Delete Ordering
```go
// TestFileResourceDeleteOrderingSuccess tests cleanup
func TestFileResourceDeleteOrderingSuccess(t *testing.T) {
    // Create file resource + project referencing it
    // Run apply

    // Try destroying file resource first
    // Terraform should either:
    // 1. Error saying project depends on it, or
    // 2. Auto-destroy project first

    // Verify clean destroy with no orphaned state
}
```

### Test Organization

Add these tests to `project_files_test.go` in a new section:

```go
// =============================================================================
// FILE RESOURCE OPERATIONS
// =============================================================================

// TestFileResourceCreateSuccess...
// TestFileResourceInstructionsSuccess...
// ... etc
```

### Test Execution Order

Run tests in this order during development:
1. TestFileResourceCreateSuccess (basic functionality)
2. TestFileResourceContentAndInstructionsFailure (validation)
3. TestFileResourceInstructionsSuccess (Claude integration)
4. TestFileResourceSharedAcrossProjectsSuccess (registry)
5. TestFileResourcePrecedenceSuccess (merge logic)
6. TestFileResourceUpdateTriggersProjectUpdateSuccess (state propagation)
7. TestFileResourceMixedWithInlineSuccess (compatibility)
8. TestFileResourceNameCollisionFailure (error handling)
9. TestFileResourcePathOverrideSuccess (flexibility)
10. TestFileResourceDeleteOrderingSuccess (cleanup)

## References

- Existing code: `internal/resources/project.go` (file collection logic)
- Existing code: `internal/resources/stack.go` (registry pattern reference)
- Existing code: `internal/registry/registry.go` (registry implementation)
- Existing code: `internal/files/merger.go` (merge precedence logic)
- Existing code: `internal/schemas/common.go` (FileSpec schema)
- Test patterns: `test/project_files_test.go` (existing file operation tests)
