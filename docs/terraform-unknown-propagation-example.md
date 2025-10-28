# Terraform Unknown Value Propagation Example

## The Problem: Computed IDs in Module Outputs

### Module Definition (examples/stacks/tofukit-stack-python-click-app/)

```hcl
# features.tofu
resource "tofukit_feature" "readme" {
  name = "readme"
  # ... feature configuration
}

# outputs.tofu
output "features" {
  value = {
    readme = tofukit_feature.readme.id  # ID is computed, Unknown during plan
  }
}
```

### Project Using Module (examples/projects/click-cli-hello-world/)

```hcl
module "click_app" {
  source = "../stacks/tofukit-stack-python-click-app"
}

resource "tofukit_project" "click_cli" {
  features = module.click_app.features  # Receives Unknown map
}
```

## What Terraform Shows During Plan

```
# module.click_app.tofukit_feature.readme will be created
+ resource "tofukit_feature" "readme" {
    + id = (known after apply)  # ← ID is Unknown
    + name = "readme"
}

# tofukit_project.click_cli will be created
+ resource "tofukit_project" "click_cli" {
    + features = (known after apply)  # ← ENTIRE map is Unknown!
}
```

**Note:** Terraform doesn't show `features = { readme = (known after apply) }` because when ANY value in a map is Unknown, the entire map becomes Unknown.

## In Provider Code (internal/resources/project.go)

```go
func (r *ProjectResourceFinal) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var data ProjectModelFinal
    req.Plan.Get(ctx, &data)
    
    // During Create(), we receive the PLAN data:
    data.Features.IsUnknown() == true  // ← Can't access the map!
    
    // Cannot iterate:
    featuresMap := data.Features.UnderlyingValue().(types.Map)  // OK
    for key, value := range featuresMap.Elements() {  // ← PANIC or empty!
        // This doesn't work with Unknown maps
    }
    
    // Result: collectFeatureFiles() returns empty []
}
```

## Timeline of Events

1. **Plan Phase:**
   - Feature resources planned (IDs are Unknown)
   - Module output computed: `features = { readme = <unknown> }`
   - Project sees: `features = (known after apply)`
   - Project.Create() receives plan with Unknown features

2. **Apply Phase:**
   - Features created first (dependency order)
   - Features saved to registry with IDs: `"feature.readme"`
   - **Project.Create() runs** - receives PLAN data where features was Unknown
   - collectFeatureFiles() returns empty (can't iterate Unknown map)
   - Empty JSON sent to Claude

3. **After Apply (State):**
   - State correctly shows: `"features": { "readme": "feature.readme" }`
   - But Create() already ran with empty data

## Why Our Fix Didn't Work

We changed from passing feature objects to passing feature IDs (strings):

```hcl
# Changed from:
output "features" {
  value = {
    readme = tofukit_feature.readme  # Entire object Unknown
  }
}

# To:
output "features" {
  value = {
    readme = tofukit_feature.readme.id  # Just string ID
  }
}
```

**Still doesn't work because:**
- `tofukit_feature.readme.id` is computed
- Computed values are Unknown during plan
- Unknown string in map → entire map marked Unknown
- Project.Create() can't iterate over Unknown map

## The Core Terraform Limitation

**Terraform doesn't re-evaluate attribute values between plan and apply for the consuming resource.**

When project.Create() runs:
- It receives the data from the PLAN
- In the plan, features was Unknown
- Even though features ARE in the registry by now, we can't access them
- The features map itself is Unknown, blocking iteration

## Solution Options

1. **Don't use module outputs for features** - Reference feature resources directly in project
2. **Use known (non-computed) feature identifiers** - Define predictable IDs upfront
3. **Defer feature processing to Update()** - Let Create() finish, process features in first Update()
4. **Use data sources** - Query registry in project using data source (re-evaluated during apply)

