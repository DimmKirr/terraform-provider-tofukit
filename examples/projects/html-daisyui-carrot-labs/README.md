# Project

Generated tofukit project

## Features

No product features identified.

## Using This Blueprint

Reference features from this blueprint in your own project:

```hcl
module "this_blueprint" {
  source = "path/to/this/blueprint"
}

resource "tofukit_project" "my_project" {
  name = "my-app"

  features = {
    "my_feature" = module.this_blueprint.features.<feature_name>
  }
}
```

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_tofukit"></a> [tofukit](#requirement\_tofukit) | ~> 0.1.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_tofukit"></a> [tofukit](#provider\_tofukit) | ~> 0.1.0 |

## Resources

| Name | Type |
|------|------|
| tofukit_feature.integration_google_font_api | resource |
| tofukit_feature.integration_open_meteo_api | resource |
| tofukit_project.labs | resource |

## Inputs

No inputs.

## Outputs

No outputs.
