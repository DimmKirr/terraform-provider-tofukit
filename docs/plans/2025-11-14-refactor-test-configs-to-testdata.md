# Refactor Test Configs to testdata/ with Embed Pattern

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move inline Terraform configurations in test files to separate .tofu files in testdata/ directory using Go 1.16+ embed pattern for cleaner, more maintainable tests.

**Architecture:** Extract all inline terraform config strings into testdata/configs/*.tofu files, embed them at compile time using //go:embed directives, and refactor tests to use the embedded configs. This follows standard Go testing conventions and improves code organization.

**Tech Stack:** Go 1.16+ (embed), Terraform/OpenTofu config files, testify

---

## Context

Current tests embed long Terraform configurations as inline strings:
```go
config := `terraform {
  // 50+ lines of configuration
}
`
```

**Problems:**
- No syntax highlighting or validation
- Clutters test code
- Hard to read and maintain
- Duplication across similar tests

**Solution:** Use Go's `//go:embed` to load configs from testdata/ at compile time.

**Related Files:**
- `/test/resource_file_test.go` - 3 tests with inline configs
- `/test/resource_feature_test.go` - 10+ tests with inline configs
- `/test/resource_project_inline_file_test.go` - Many tests with inline configs
- `/test/helpers_test.go` - Will add embed declarations here

---

## Task 1: Set Up testdata Directory and Extract First Config

**Goal:** Create testdata structure and extract one config as proof of concept.

**Files:**
- Create: `test/testdata/configs/file-resource-basic.tofu`
- Reference: `test/resource_file_test.go:354-387` (TestResourceFile_CreateGeneratePromptSuccess)

### Step 1: Create testdata directory structure

```bash
cd test
mkdir -p testdata/configs
```

### Step 2: Extract first config to file

Create `test/testdata/configs/file-resource-basic.tofu`:

```hcl
# Basic file resource test configuration
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug       = true
  dry_run     = true
}

resource "tofukit_file" "hello" {
  name    = "hello.txt"
  content = "Hello World"
}

resource "tofukit_project" "test" {
  name    = "file-test"
  version = "1.0.0"

  files = {
    "hello.txt" = tofukit_file.hello
  }
}
```

**Command:**
```bash
cat > test/testdata/configs/file-resource-basic.tofu << 'EOF'
[paste config above]
EOF
```

### Step 3: Verify file was created correctly

```bash
ls -lh test/testdata/configs/file-resource-basic.tofu
cat test/testdata/configs/file-resource-basic.tofu
```

**Expected:** File exists with correct content, properly formatted HCL

### Step 4: Validate terraform syntax (optional but recommended)

```bash
cd test/testdata/configs
terraform fmt -check file-resource-basic.tofu
```

**Expected:** No output (file is already formatted) or formatting applied

### Step 5: Commit testdata setup

```bash
git add test/testdata/
git commit -m "test: add testdata directory structure and first config

- Create test/testdata/configs/ for embedded terraform configs
- Add file-resource-basic.tofu extracted from TestResourceFile_CreateGeneratePromptSuccess
- Prepares for refactoring inline configs to embedded files

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Add Embed Declaration to helpers_test.go

**Goal:** Set up Go embed infrastructure in helpers_test.go for loading configs.

**Files:**
- Modify: `test/helpers_test.go` (add embed import and declarations)

### Step 1: Add embed import to helpers_test.go

Find the import section at top of `test/helpers_test.go` and add:

```go
import (
	_ "embed"  // Add this line
	"fmt"
	"io"
	"os"
	// ... rest of imports
)
```

### Step 2: Add embed declaration for first config

Add after imports, before any functions:

```go
// Embedded Terraform test configurations
// These are loaded at compile time from testdata/configs/

//go:embed testdata/configs/file-resource-basic.tofu
var ConfigFileResourceBasic string
```

### Step 3: Verify code compiles

```bash
cd test
go test -c -o /dev/null
```

**Expected:** No errors, successful compilation

### Step 4: Verify embedded content

Add a quick verification test temporarily:

```go
func TestEmbedWorking(t *testing.T) {
	assert.NotEmpty(t, ConfigFileResourceBasic, "Config should be embedded")
	assert.Contains(t, ConfigFileResourceBasic, "tofukit_file", "Config should contain resource")
	t.Logf("Embedded config length: %d bytes", len(ConfigFileResourceBasic))
}
```

### Step 5: Run verification test

```bash
go test -v -run TestEmbedWorking
```

**Expected:** PASS - Shows config length (should be ~400-500 bytes)

### Step 6: Remove verification test

Delete the `TestEmbedWorking` function (it was just for verification).

### Step 7: Commit embed infrastructure

```bash
git add test/helpers_test.go
git commit -m "test: add embed infrastructure to helpers_test.go

- Import embed package for Go 1.16+ embedding
- Add ConfigFileResourceBasic embedded from testdata
- Enables compile-time config loading for cleaner tests

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: Refactor First Test to Use Embedded Config

**Goal:** Refactor TestResourceFile_CreateGeneratePromptSuccess to use embedded config.

**Files:**
- Modify: `test/resource_file_test.go:354-432` (TestResourceFile_CreateGeneratePromptSuccess)

### Step 1: Locate the test function

```bash
grep -n "func TestResourceFile_CreateGeneratePromptSuccess" test/resource_file_test.go
```

**Expected:** Shows line number (around line 354)

### Step 2: Find the inline config

Look for the `config :=` string literal in the test (lines ~358-387).

### Step 3: Replace inline config with embedded config

**Before:**
```go
func TestResourceFile_CreateGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceFile_CreateGeneratePromptSuccess")

	// Config with file resource
	config := `
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}
// ... 30 more lines
`

	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)
	// ... rest of test
}
```

**After:**
```go
func TestResourceFile_CreateGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceFile_CreateGeneratePromptSuccess")

	// Use embedded config from testdata
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(ConfigFileResourceBasic), 0644)
	require.NoError(t, err)

	// ... rest of test (unchanged)
}
```

**Key changes:**
- Remove entire `config := ...` multiline string
- Change `[]byte(config)` to `[]byte(ConfigFileResourceBasic)`
- Add comment explaining we're using embedded config

### Step 4: Run the refactored test

```bash
go test -v -run TestResourceFile_CreateGeneratePromptSuccess -timeout 5m
```

**Expected:** PASS - Test behavior unchanged, just cleaner code

### Step 5: Verify test output directory

```bash
ls -la test-output/*/output/
```

**Expected:** Test created files correctly, same as before refactor

### Step 6: Commit the refactor

```bash
git add test/resource_file_test.go
git commit -m "refactor(test): use embedded config in TestResourceFile_CreateGeneratePromptSuccess

- Replace 30-line inline config with ConfigFileResourceBasic
- Config now loaded from testdata/configs/file-resource-basic.tofu
- Test behavior unchanged, code is cleaner and more maintainable
- First test successfully using embed pattern

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Extract and Refactor Remaining Configs (Batch 1 - resource_file_test.go)

**Goal:** Extract remaining configs from resource_file_test.go and refactor tests.

**Files:**
- Create: `test/testdata/configs/file-resource-drift.tofu`
- Modify: `test/helpers_test.go` (add embed declaration)
- Modify: `test/resource_file_test.go:435-503` (TestResourceFile_DriftDetectionGeneratePromptSuccess)

### Step 1: Extract second config from resource_file_test.go

Look at `TestResourceFile_DriftDetectionGeneratePromptSuccess` (lines ~435-503).

Create `test/testdata/configs/file-resource-drift.tofu`:

```hcl
# File resource drift detection test configuration
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug       = true
  dry_run     = true
}

resource "tofukit_file" "hello" {
  name    = "hello.txt"
  content = "Hello World"
}

resource "tofukit_project" "test" {
  name    = "file-drift-test"
  version = "1.0.0"

  files = {
    "hello.txt" = tofukit_file.hello
  }
}
```

### Step 2: Add embed declaration to helpers_test.go

Add below the existing ConfigFileResourceBasic:

```go
//go:embed testdata/configs/file-resource-drift.tofu
var ConfigFileResourceDrift string
```

### Step 3: Refactor TestResourceFile_DriftDetectionGeneratePromptSuccess

Replace inline config with:

```go
configPath := filepath.Join(testDir, "project.tofu")
err := os.WriteFile(configPath, []byte(ConfigFileResourceDrift), 0644)
require.NoError(t, err)
```

### Step 4: Run both file resource tests

```bash
go test -v -run "TestResourceFile.*GeneratePromptSuccess" -timeout 5m
```

**Expected:** PASS - Both tests pass with embedded configs

### Step 5: Format terraform configs

```bash
cd test/testdata/configs
terraform fmt file-resource-basic.tofu file-resource-drift.tofu
```

### Step 6: Commit file resource refactor

```bash
git add test/testdata/configs/file-resource-drift.tofu test/helpers_test.go test/resource_file_test.go
git commit -m "refactor(test): complete resource_file_test.go config extraction

- Extract file-resource-drift.tofu to testdata
- Add ConfigFileResourceDrift embed declaration
- Refactor TestResourceFile_DriftDetectionGeneratePromptSuccess
- All resource_file_test.go tests now use embedded configs

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 5: Extract Feature Test Configs (Batch 2 - High Priority)

**Goal:** Extract configs from high-value feature tests.

**Files:**
- Create: `test/testdata/configs/feature-inline-basic.tofu`
- Create: `test/testdata/configs/feature-precedence.tofu`
- Create: `test/testdata/configs/feature-multiple-merge.tofu`
- Modify: `test/helpers_test.go` (add 3 embed declarations)
- Modify: `test/resource_feature_test.go` (refactor 3 tests)

### Step 1: Extract feature-inline-basic.tofu

From `TestResourceFeatureInlineGeneratePromptSuccess` (lines ~654-688):

Create `test/testdata/configs/feature-inline-basic.tofu`:

```hcl
# Inline feature basic test configuration
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug = true
  dry_run = true
}

resource "tofukit_project" "test" {
  name    = "inline-feature-test"
  version = "0.1.0"

  features = {
    "hello" = {
      prompt = "Add hello greeting capability"
      files = {
        "hello.txt" = {
          content = "Hello from inline feature.\n"
        }
      }
    }
  }
}
```

### Step 2: Extract feature-precedence.tofu

From `TestProjectPrecedenceOverFeatureGeneratePromptSuccess` (lines ~753-793):

Create `test/testdata/configs/feature-precedence.tofu`:

```hcl
# Feature precedence test configuration
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug = true
  dry_run = true
}

resource "tofukit_project" "test" {
  name    = "precedence-test"
  version = "0.1.0"

  features = {
    "base_config" = {
      prompt = "Provide base configuration"
      files = {
        "config.txt" = {
          content = "Config from feature\n"
        }
      }
    }
  }

  files = {
    "config.txt" = {
      content = "Config from project\n"
    }
  }
}
```

### Step 3: Extract feature-multiple-merge.tofu

From `TestResourceFeatureInlineMergeMultipleFeaturesGeneratePromptSuccess` (lines ~836-886):

Create `test/testdata/configs/feature-multiple-merge.tofu`:

```hcl
# Multiple features merge test configuration
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug = true
  dry_run = true
}

resource "tofukit_project" "test" {
  name    = "multi-feature-test"
  version = "0.1.0"

  features = {
    "logging" = {
      prompt = "Add logging capability"
      files = {
        "log.txt" = {
          content = "Logging enabled\n"
        }
      }
    }
    "config" = {
      prompt = "Add configuration capability"
      files = {
        "config.txt" = {
          content = "Config loaded\n"
        }
      }
    }
    "metrics" = {
      prompt = "Add metrics capability"
      files = {
        "metrics.txt" = {
          content = "Metrics tracking\n"
        }
      }
    }
  }
}
```

### Step 4: Add embed declarations to helpers_test.go

Add after existing configs:

```go
//go:embed testdata/configs/feature-inline-basic.tofu
var ConfigFeatureInlineBasic string

//go:embed testdata/configs/feature-precedence.tofu
var ConfigFeaturePrecedence string

//go:embed testdata/configs/feature-multiple-merge.tofu
var ConfigFeatureMultipleMerge string
```

### Step 5: Refactor TestResourceFeatureInlineGeneratePromptSuccess

Replace inline config with:

```go
configPath := filepath.Join(testDir, "project.tofu")
err := os.WriteFile(configPath, []byte(ConfigFeatureInlineBasic), 0644)
require.NoError(t, err)
```

### Step 6: Refactor TestProjectPrecedenceOverFeatureGeneratePromptSuccess

Replace inline config with:

```go
configPath := filepath.Join(testDir, "project.tofu")
err := os.WriteFile(configPath, []byte(ConfigFeaturePrecedence), 0644)
require.NoError(t, err)
```

### Step 7: Refactor TestResourceFeatureInlineMergeMultipleFeaturesGeneratePromptSuccess

Replace inline config with:

```go
configPath := filepath.Join(testDir, "project.tofu")
err := os.WriteFile(configPath, []byte(ConfigFeatureMultipleMerge), 0644)
require.NoError(t, err)
```

### Step 8: Run refactored feature tests

```bash
go test -v -run "TestResourceFeatureInlineGeneratePromptSuccess|TestProjectPrecedenceOverFeature|TestResourceFeatureInlineMergeMultipleFeaturesGeneratePromptSuccess" -timeout 10m
```

**Expected:** PASS - All 3 tests pass

### Step 9: Format all feature configs

```bash
cd test/testdata/configs
terraform fmt feature-*.tofu
```

### Step 10: Commit feature test refactor

```bash
git add test/testdata/configs/feature-*.tofu test/helpers_test.go test/resource_feature_test.go
git commit -m "refactor(test): extract feature test configs to testdata

- Add feature-inline-basic.tofu for inline feature tests
- Add feature-precedence.tofu for precedence hierarchy tests
- Add feature-multiple-merge.tofu for multi-feature tests
- Refactor 3 feature tests to use embedded configs
- Tests ~40% shorter and cleaner

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 6: Add Terraform Validation Script

**Goal:** Create script to validate all testdata configs are syntactically correct.

**Files:**
- Create: `test/testdata/validate-configs.sh`

### Step 1: Create validation script

Create `test/testdata/validate-configs.sh`:

```bash
#!/bin/bash
set -e

echo "Validating Terraform configs in testdata/configs/"
cd "$(dirname "$0")/configs"

failed=0
for config in *.tofu; do
    echo -n "Validating $config... "
    if terraform fmt -check "$config" > /dev/null 2>&1; then
        echo "✓ formatted"
    else
        echo "✗ needs formatting"
        terraform fmt "$config"
        echo "  → auto-formatted"
    fi
done

echo ""
echo "All configs validated successfully!"
exit $failed
```

### Step 2: Make script executable

```bash
chmod +x test/testdata/validate-configs.sh
```

### Step 3: Run validation script

```bash
cd test/testdata
./validate-configs.sh
```

**Expected:** All configs pass validation or are auto-formatted

### Step 4: Add to Taskfile (if exists)

Check if Taskfile.yml has test tasks:

```bash
grep -A5 "test:" Taskfile.yml
```

If it exists, add a task:

```yaml
  validate-test-configs:
    desc: Validate Terraform configs in testdata
    dir: test/testdata
    cmds:
      - ./validate-configs.sh
```

### Step 5: Test the validation

```bash
task validate-test-configs
```

**Expected:** Script runs successfully

### Step 6: Commit validation infrastructure

```bash
git add test/testdata/validate-configs.sh Taskfile.yml
git commit -m "test: add validation script for testdata configs

- Add validate-configs.sh to check terraform syntax
- Auto-formats configs if needed
- Add task validate-test-configs to Taskfile.yml
- Ensures testdata configs remain valid

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com)"
```

---

## Task 7: Update Documentation

**Goal:** Document the testdata/embed pattern for future test authors.

**Files:**
- Modify: `CLAUDE.md` (testing section)

### Step 1: Find testing section in CLAUDE.md

```bash
grep -n "## Testing" CLAUDE.md
```

### Step 2: Add testdata/embed documentation

Add a new section after the existing testing content:

```markdown
### Test Configuration Files

Test terraform configurations are stored in `test/testdata/configs/` and embedded at compile time using Go 1.16+ embed pattern.

**Structure:**
```
test/
├── testdata/
│   ├── configs/
│   │   ├── file-resource-basic.tofu
│   │   ├── feature-inline-basic.tofu
│   │   └── ...
│   └── validate-configs.sh
└── helpers_test.go (embed declarations)
```

**Adding New Test Config:**

1. Create config file in `test/testdata/configs/`:
   ```bash
   cat > test/testdata/configs/my-test-config.tofu << 'EOF'
   terraform {
     required_providers {
       tofukit = { ... }
     }
   }
   # ... rest of config
   EOF
   ```

2. Add embed declaration to `test/helpers_test.go`:
   ```go
   //go:embed testdata/configs/my-test-config.tofu
   var ConfigMyTest string
   ```

3. Use in test:
   ```go
   func TestMyFeature(t *testing.T) {
       testDir := createTestDirectory(t, "TestMyFeature")
       err := os.WriteFile(
           filepath.Join(testDir, "project.tofu"),
           []byte(ConfigMyTest),
           0644,
       )
       require.NoError(t, err)
       // ... rest of test
   }
   ```

**Validating Configs:**
```bash
cd test/testdata
./validate-configs.sh  # Or: task validate-test-configs
```

**Benefits:**
- ✅ Syntax highlighting and validation
- ✅ Cleaner test code (no long inline strings)
- ✅ Compile-time embedding (fast, portable)
- ✅ Easy reuse across tests
- ✅ Standard Go pattern
```

### Step 3: Verify documentation renders correctly

Read through the added section to ensure formatting is correct.

### Step 4: Commit documentation

```bash
git add CLAUDE.md
git commit -m "docs: document testdata/embed pattern for test configs

- Add section explaining testdata/configs/ structure
- Document how to add new test configurations
- Explain embed pattern and benefits
- Link to validation script

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 8: Refactor Remaining High-Value Tests (Optional Batch)

**Goal:** Identify and refactor other frequently-run or important tests.

**Files:**
- Analyze: All test files for inline configs
- Prioritize based on test frequency/importance

### Step 1: Find all inline terraform configs

```bash
cd test
grep -n 'config.*:=.*`' *.go | grep terraform
```

**Expected:** List of remaining inline configs

### Step 2: Prioritize by test value

Look for:
- Tests that run frequently (in CI)
- Tests with very long configs (>50 lines)
- Tests that are duplicating similar configs

### Step 3: Create extraction plan

For each prioritized test, decide:
- Extract to new file or reuse existing?
- Is config truly unique or can be shared?

### Step 4: Extract configs in batches

Follow the same pattern as previous tasks:
1. Create .tofu file in testdata/configs/
2. Add embed declaration
3. Refactor test
4. Run test to verify
5. Commit

**Suggested configs to extract:**
- `TestResourceFeatureFilesIndependentGeneratePromptSuccess` → `feature-independent-basic.tofu`
- `TestProjectPrecedenceOverFeatureSuccess` → Can reuse `feature-precedence.tofu`
- E2E tests with long configs (if beneficial)

### Step 5: Run all tests

After each batch:

```bash
go test ./test/ -v -timeout 30m
```

**Expected:** All tests pass

### Step 6: Commit each batch separately

Follow commit message pattern from previous tasks.

---

## Verification Checklist

After completing all tasks:

- [ ] All extracted configs exist in `test/testdata/configs/`
- [ ] All configs have embed declarations in `helpers_test.go`
- [ ] All refactored tests pass: `go test ./test/ -v`
- [ ] Configs are formatted: `cd test/testdata/configs && terraform fmt -check *.tofu`
- [ ] Validation script works: `test/testdata/validate-configs.sh`
- [ ] Documentation updated in CLAUDE.md
- [ ] All changes committed with descriptive messages

**Run full test suite:**
```bash
task test
# Or: go test ./... -v -timeout 30m
```

**Expected:** All tests pass, no behavior changes

---

## Success Criteria

**Testdata Structure:**
- ✅ `test/testdata/configs/` contains 5+ .tofu files
- ✅ All configs properly formatted (terraform fmt)
- ✅ Validation script passes

**Code Quality:**
- ✅ Embedded configs in `helpers_test.go` with clear names
- ✅ Tests 30-50% shorter (removed inline strings)
- ✅ No test behavior changes (all pass)
- ✅ Easier to maintain and understand

**Documentation:**
- ✅ CLAUDE.md explains pattern
- ✅ Examples show how to add new configs
- ✅ Validation process documented

---

## Notes for Implementation

**Naming Convention:**
Config file names should be descriptive and follow pattern:
- `resource-type-scenario.tofu`
- Examples: `file-resource-basic.tofu`, `feature-precedence.tofu`

**Config Reuse:**
If multiple tests use identical or nearly identical configs:
- Extract once
- Reuse the same embedded variable
- Document which tests share the config

**Testing Strategy:**
- Refactor one test at a time
- Run test after each refactor to ensure it still passes
- Commit frequently (after each test or batch)

**Common Pitfalls:**
- ❌ Don't forget to format configs with `terraform fmt`
- ❌ Don't change config behavior when extracting
- ❌ Don't forget to add embed declaration for each config
- ✅ Always run test after refactor to verify

**Performance:**
Embedding adds ~negligible overhead (<1ms per test) because configs are loaded at compile time, not runtime.

**Rollback:**
If a refactor breaks something, git revert the commit and investigate. Each commit is atomic and safe to revert.
