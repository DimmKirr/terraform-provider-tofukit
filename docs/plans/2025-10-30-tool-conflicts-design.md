# Tool Conflicts Design

**Date:** 2025-10-30
**Status:** Design/Brainstorm
**Author:** Dmitry + Claude

## Overview

Add support for declaring tool/framework/language conflicts to prevent incompatible technologies from being used together. This addresses scenarios where:
- Multiple tools serve the same purpose (uv vs pip vs poetry vs pipenv)
- Technologies are mutually exclusive (Go vs Python in same project)
- Using one tool makes another redundant or problematic

## Motivation

**Current problem:** When using Task for build automation, projects might accidentally include Makefile, creating confusion. Similarly, when using `uv` for Python package management, Claude might still create `requirements.txt`, `poetry.lock`, or `Pipfile` files.

**Current workarounds:**
- Manual constraints in each tool's requirements
- Verifications checking for absence of files
- Verbose repetitive instructions

**Goal:** Declaratively specify tool conflicts so the provider automatically generates appropriate constraints and verifications for Claude.

## Design

### Configuration Interface

Add optional `conflicts` field to kit resources: `tofukit_tool`, `tofukit_language`, `tofukit_framework`.

**Two supported formats:**

#### Simple Format (String List)
```hcl
resource "tofukit_tool" "uv" {
  name    = "uv"
  version = "0.9.5"

  conflicts = ["pip", "poetry", "pipenv"]

  requirements = [...]
}
```

#### Structured Format (With Reasons)
```hcl
resource "tofukit_tool" "uv" {
  name    = "uv"
  version = "0.9.5"

  conflicts = [
    {
      name   = "pip"
      reason = "uv is a faster, all-in-one replacement for pip"
    },
    {
      name   = "poetry"
      reason = "uv handles dependencies natively, no need for poetry"
    },
    {
      name   = "pipenv"
      reason = "uv provides better performance and simpler workflow"
    }
  ]

  requirements = [...]
}
```

**Schema:** Use `types.Dynamic` to accept both formats. Provider detects format and processes accordingly.

### Prompt Generation

When a tool with conflicts is referenced in a project, the provider generates:

1. **Constraint Instructions** - Added to the tool's requirements:
   ```
   "DO NOT use {conflict_name} - {reason}"
   ```

2. **Artifact Verifications** - Inferred file checks:
   ```hcl
   {
     command = "test ! -f requirements.txt && echo 'no requirements.txt'"
     expect  = "no requirements.txt"
   }
   ```

3. **Grouped by Tool** - All conflict constraints appear together with the tool's requirements.

#### Example Transformation

**Input Configuration:**
```hcl
resource "tofukit_tool" "uv" {
  name = "uv"

  conflicts = [
    {
      name   = "pip"
      reason = "uv is a faster, all-in-one replacement for pip"
    },
    {
      name   = "poetry"
      reason = "uv handles dependencies natively"
    }
  ]

  requirements = [{
    name = "Python Package Management"
    instructions = [{
      prompt = "Use uv for Python package management"
    }]
  }]
}
```

**Generated Claude Prompt (Relevant Sections):**
```json
{
  "specification": {
    "kits": {
      "tool.uv": {
        "name": "uv",
        "version": "0.9.5"
      }
    },

    "requirements": [
      {
        "name": "Python Package Management",
        "instructions": [{
          "prompt": "Use uv for Python package management",
          "constraints": [
            "DO NOT use pip - uv is a faster, all-in-one replacement for pip",
            "DO NOT use poetry - uv handles dependencies natively",
            "DO NOT create requirements.txt - use pyproject.toml with uv",
            "DO NOT create poetry.lock or Pipfile"
          ]
        }]
      }
    ],

    "verifications": [
      {
        "command": "test ! -f requirements.txt && echo 'no requirements.txt'",
        "expect": "no requirements.txt"
      },
      {
        "command": "test ! -f poetry.lock && echo 'no poetry.lock'",
        "expect": "no poetry.lock"
      },
      {
        "command": "test ! -f Pipfile && echo 'no Pipfile'",
        "expect": "no Pipfile"
      }
    ]
  }
}
```

## Implementation Outline

### Phase 1: Schema Extension
- Add `conflicts` attribute to `BaseKitModel` schema (used by tool/language/framework)
- Use `types.Dynamic` to accept both string list and structured object list
- Mark as optional (existing resources continue to work)

### Phase 2: Conflict Processing
- Create `internal/conflicts/processor.go` package
- `DetectFormat()` - Determine if conflicts is string list or structured list
- `ParseConflicts()` - Extract conflict names and optional reasons
- `GenerateConstraints()` - Convert conflicts to "DO NOT" constraint strings
- `InferArtifacts()` - Map tool names to common artifact files (e.g., pip → requirements.txt)

### Phase 3: Artifact Mapping
- Create conflict → artifact mapping:
  ```go
  var artifactMap = map[string][]string{
    "pip":    {"requirements.txt"},
    "poetry": {"poetry.lock", "pyproject.toml"}, // Only if poetry-specific
    "pipenv": {"Pipfile", "Pipfile.lock"},
    "make":   {"Makefile", "makefile"},
  }
  ```

### Phase 4: Integration with Prompt Builder
- Modify `internal/resources/project.go:buildOutputData()`
- When processing kits, check for conflicts field
- Generate constraints and add to kit's requirements
- Generate verifications for inferred artifacts
- Merge into existing prompt structure

### Phase 5: Documentation
- Update CLAUDE.md with conflicts field documentation
- Add examples for common conflict scenarios
- Document both simple and structured formats

## Examples

### Example 1: Python Package Managers

```hcl
resource "tofukit_tool" "uv" {
  name    = "uv"
  version = "0.9.5"

  conflicts = ["pip", "poetry", "pipenv"]

  requirements = [{
    name = "uv Installation"
    instructions = [{
      prompt = "Install uv: curl -LsSf https://astral.sh/uv/install.sh | sh"
    }]
  }]
}
```

**Generated constraints:**
- "DO NOT use pip"
- "DO NOT use poetry"
- "DO NOT use pipenv"
- "DO NOT create requirements.txt, poetry.lock, or Pipfile"

**Generated verifications:**
- Check no requirements.txt
- Check no poetry.lock
- Check no Pipfile

### Example 2: Build Tools with Context

```hcl
resource "tofukit_tool" "gotask" {
  name = "gotask"

  conflicts = [
    {
      name   = "make"
      reason = "Task provides modern YAML syntax vs Make's tab-sensitive format"
    }
  ]

  requirements = [...]
}
```

**Generated constraints:**
- "DO NOT use make - Task provides modern YAML syntax vs Make's tab-sensitive format"
- "DO NOT create Makefile or makefile"

**Generated verifications:**
- Check no Makefile

### Example 3: Language Conflicts

```hcl
resource "tofukit_language" "go" {
  name    = "go"
  version = "1.23.1"

  conflicts = ["python", "node", "rust"]

  requirements = [...]
}
```

**Generated constraints:**
- "DO NOT use python - this is a Go project"
- "DO NOT use node - this is a Go project"
- "DO NOT use rust - this is a Go project"

## Future Enhancements

### URI-Based References (Phase 2)
Support referencing conflicts via URIs:
```hcl
conflicts = [
  tofukit_tool.pip.link,
  tofukit_tool.poetry.link,
]
```

Benefits:
- Type safety (Terraform validates references)
- Access to full tool metadata
- Can extract descriptions/reasons from referenced tool

### Category-Based Conflicts (Phase 3)
Automatic exclusions based on categories:
```hcl
resource "tofukit_tool" "uv" {
  category = "python-package-manager"
  exclusive = true  # Auto-excludes other tools in same category
}
```

### Terraform Validation (Phase 4)
Detect conflicts during `terraform plan`:
```
Error: Conflicting tools detected
Tool 'uv' conflicts with 'pip' but both are referenced in the project.
```

## Open Questions

1. **Artifact inference accuracy** - How do we handle cases where artifact names overlap or are ambiguous?
2. **Feature-level conflicts** - Should `tofukit_feature` also support conflicts field?
3. **Stack-level conflicts** - Should stacks be able to declare conflicts that apply to all contained features?
4. **Validation vs warnings** - Should conflicting tools cause terraform errors or just warnings?

## Success Criteria

1. Can declare tool conflicts in two formats (simple and structured)
2. Provider generates appropriate constraints for Claude
3. Provider generates artifact verifications automatically
4. Existing configurations continue to work (backward compatible)
5. Reduces repetitive constraint writing across tool definitions

## Non-Goals (For Initial Implementation)

- URI-based conflict references (future enhancement)
- Category-based automatic exclusions (future enhancement)
- Terraform-level validation/errors (future enhancement)
- Conflict resolution strategies (pick one, warn, etc.)
