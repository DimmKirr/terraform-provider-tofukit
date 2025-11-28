# Basic Documentation Feature Module

A reusable TofuKit feature module that generates project README.md documentation based on a dynamic list of features.

## Overview

This module provides a `tofukit_feature` that creates a README.md file documenting your project's features. It accepts a map of features and their descriptions as input variables, then generates professional documentation.

## Features

- **Dynamic Feature List**: Pass features as a map variable and they're automatically documented
- **Customizable**: Configure project name, description, and feature list
- **Reusable**: Use across multiple projects as a Terraform module
- **Standards-Compliant**: Generates clean markdown following best practices

## Module Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `project_name` | string | `"my-project"` | Name of the project for the README |
| `project_description` | string | `"A software project"` | Brief description of the project |
| `features` | map(string) | See below | Map of feature names to descriptions |

**Default features map:**
```hcl
{
  "feature1" = "First feature description"
  "feature2" = "Second feature description"
}
```

## Module Outputs

| Output | Description |
|--------|-------------|
| `feature` | The complete feature resource |
| `feature_id` | The feature ID |
| `feature_link` | URI link for referencing this feature |

## Usage

### Basic Usage

```hcl
module "documentation" {
  source = "./path/to/tofukit-feature-basic-documentation"

  project_name        = "my-cli-app"
  project_description = "A command-line tool for data processing"

  features = {
    "csv-export"    = "Export data to CSV format"
    "json-import"   = "Import data from JSON files"
    "data-validate" = "Validate data against schemas"
  }
}
```

### Using in a Project

```hcl
resource "tofukit_project" "app" {
  name    = "my-app"
  version = "1.0.0"

  # Include the documentation feature
  features = {
    "docs" = module.documentation.feature
  }

  # ... other project configuration
}
```

### Complete Example

See `example-usage.tofu` for a complete working example.

## How It Works

1. **Variable Input**: You provide a map of features with their descriptions
2. **Template Processing**: The module uses Terraform's `join()` and `for` expressions to format the feature list
3. **Prompt Generation**: A structured prompt is created with:
   - Project name and description
   - Formatted list of all features
   - Documentation structure requirements
4. **Claude Execution**: TofuKit/Claude generates the README.md based on the prompt
5. **Verification**: The module verifies the README exists and contains expected content

## Feature Iteration Pattern

The module demonstrates how to iterate over a map variable in TofuKit prompts:

```hcl
prompt = <<-EOF
  FEATURES TO DOCUMENT:
  ${join("\n", [for key, desc in var.features : "- ${key}: ${desc}"])}
EOF
```

This pattern:
- Uses Terraform's `for` expression to iterate over the `var.features` map
- Formats each entry as `- key: description`
- Joins all entries with newlines into a bulleted list

## File Structure

```
tofukit-feature-basic-documentation/
├── README.md           # This file
├── feature.tofu        # Feature resource definition
├── variables.tofu      # Input variables
├── outputs.tofu        # Module outputs
└── example-usage.tofu  # Complete usage example
```

## Testing

To test this module:

1. Copy the module to your examples directory
2. Create a test project using `example-usage.tofu` as a template
3. Run `tofu init` and `tofu apply`
4. Check the generated `output/README.md` file

## License

Part of the TofuKit provider examples.
