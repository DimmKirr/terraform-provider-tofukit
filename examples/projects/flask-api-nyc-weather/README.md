# Flask NYC Weather API Example

This example demonstrates the **separation of concerns** pattern in TofuKit by using:
- **Integration resource** for API-specific details (reusable across projects)
- **Feature resource** for product-specific requirements (NYC weather)
- **Project resource** for application implementation (Flask API)

## Module Composition

This project demonstrates TofuKit's module composition:

- **Integration Module**: `tofukit-integration-openmeteo` - Reusable Open-Meteo API integration
- **Stack Module**: `tofukit-stack-python-flask-app` - Base Flask application setup
- **Project**: NYC-specific weather endpoint implementation

The project focuses only on business logic while leveraging reusable modules for infrastructure.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Integration (API Details)                                    │
│ tofukit_integration.open_meteo                              │
│                                                              │
│ • Generic Open-Meteo API                                    │
│ • Base URL, docs, rate limits                               │
│ • Available endpoints & parameters                          │
│ • Reusable for ANY location                                 │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ references via ${tofukit_integration.open_meteo.link}
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ Feature (Product Requirements)                               │
│ tofukit_feature.nyc_weather                                 │
│                                                              │
│ • NYC-specific coordinates                                   │
│ • Weather parameters to fetch                                │
│ • Business logic for NYC weather                            │
│ • References integration for API details                    │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ used in features = {}
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ Project (Implementation)                                     │
│ tofukit_project.flask_weather                               │
│                                                              │
│ • Flask application implementation                           │
│ • API endpoints (/weather, /health)                         │
│ • Files: app.py, requirements.txt, etc.                     │
│ • Uses nyc_weather feature for data logic                   │
└─────────────────────────────────────────────────────────────┘
```

## File Structure

```
examples/projects/flask-api-nyc-weather/
├── features.tofu       # NYC Weather feature (product-specific)
├── project.tofu        # Flask API project (implementation)
└── README.md           # This file

# Integration now in reusable module:
examples/integrations/tofukit-integration-openmeteo/
└── integration.tofu    # Open-Meteo API (generic, reusable)
```

## Why This Pattern?

### ✅ Before Refactoring (Tightly Coupled)

```hcl
# Before: integration.tofu - Mixed concerns
resource "tofukit_integration" "open_meteo" {
  metadata = {
    latitude  = "40.7834"  # ❌ NYC-specific data in integration!
    longitude = "-73.9663"
  }
}

# Before: project.tofu - Inline requirements
resource "tofukit_project" "flask_weather" {
  requirements = [
    {
      # ❌ All logic duplicated inline
      instructions = [...]
    }
  ]
}
```

**Problems:**
- Integration contains product-specific data (NYC coordinates)
- Can't reuse Open-Meteo integration for other cities
- Requirements duplicated if multiple projects need NYC weather
- No clear separation between API details and business logic

### ✅ After Refactoring (Separated Concerns)

```hcl
# examples/integrations/tofukit-integration-openmeteo/integration.tofu
# Pure API details in reusable module
resource "tofukit_integration" "open_meteo" {
  name     = "open-meteo"
  base_url = "https://api.open-meteo.com/v1"
  metadata = {
    # ✅ Only generic API characteristics
    auth_required = "false"
    rate_limit    = "10000/day"
    # NO location-specific data
  }
}

# features.tofu - Business requirements
resource "tofukit_feature" "nyc_weather" {
  name = "nyc-weather-data"
  requirements = [
    {
      # ✅ NYC-specific logic using integration
      prompt = "...coordinates: 40.7834, -73.9663..."
      # References: ${tofukit_integration.open_meteo.link}
    }
  ]
}

# project.tofu - Implementation
resource "tofukit_project" "flask_weather" {
  features = {
    "nyc_weather" = tofukit_feature.nyc_weather  # ✅ Reusable!
  }
}
```

**Benefits:**
- ✅ Integration is reusable for ANY location
- ✅ Feature encapsulates NYC weather logic
- ✅ Project focuses on Flask implementation
- ✅ Clear dependency chain: Project → Feature → Integration

## Reusability Example

Now you can create a **Seattle weather project** by reusing the integration:

```hcl
# Reuse the same Open-Meteo integration!
resource "tofukit_feature" "seattle_weather" {
  name = "seattle-weather-data"
  requirements = [
    {
      prompt = <<-EOF
        Fetch Seattle weather using ${tofukit_integration.open_meteo.link}
        Coordinates: 47.6062° N, 122.3321° W
      EOF
    }
  ]
}

resource "tofukit_project" "seattle_api" {
  features = {
    "seattle_weather" = tofukit_feature.seattle_weather
  }
}
```

## Usage

```bash
# Navigate to example directory
cd examples/projects/flask-api-nyc-weather/

# Initialize Terraform
tofu init

# Plan (see what will be created)
tofu plan

# Apply (generate Flask application)
tofu apply

# Generated files will be in the output directory
ls -la output/
# app.py, requirements.txt, README.md, .env.example, .gitignore
```

## What Gets Generated

The project generates a complete Flask application:

1. **app.py** - Flask server with weather endpoints
2. **requirements.txt** - Python dependencies
3. **README.md** - Application documentation
4. **.env.example** - Environment configuration template
5. **.gitignore** - Git exclusions

## Key Concepts Demonstrated

### 1. Integration Resource (`tofukit_integration`)
- Represents external APIs/services
- Contains only integration-specific data
- Reusable across multiple projects/features
- Provides `.link` attribute for references

### 2. Feature Resource (`tofukit_feature`)
- Encapsulates product requirements
- Contains business logic and constraints
- References integrations via URI links
- Can be shared across projects

### 3. Project Resource (`tofukit_project`)
- Implements the actual application
- Uses features for capabilities
- Defines files to generate
- Focuses on implementation details

### 4. Dependency Chain
```
Integration (API) → Feature (Requirements) → Project (Implementation)
```

## Design Principles

1. **Single Responsibility**: Each resource has one clear purpose
2. **Reusability**: Integration and features can be shared
3. **Composability**: Projects compose features together
4. **Separation of Concerns**: API details ≠ business logic ≠ implementation
5. **DRY (Don't Repeat Yourself)**: Define once, reference everywhere

## Comparison

| Aspect | Before Refactoring | After Refactoring |
|--------|-------------------|-------------------|
| **Integration** | NYC-specific | Generic, reusable |
| **Feature** | N/A (inline requirements) | Standalone, reusable |
| **Project** | All logic inline | Clean, compositional |
| **Reusability** | None (everything coupled) | High (all layers reusable) |
| **Maintainability** | Low (scattered logic) | High (clear separation) |

## Next Steps

To use this pattern in your own projects:

1. **Create integrations** for your external APIs
2. **Define features** that use those integrations
3. **Build projects** that compose features together
4. **Reuse** integrations and features across multiple projects

## Related Examples

- **Integration module**: See `../../integrations/tofukit-integration-openmeteo/` for reusable API resource
- **Feature definition**: See `features.tofu` for requirement encapsulation
- **Project composition**: See `project.tofu` for feature usage

## Learn More

- [Integration Resource Documentation](../../../docs/resources/integration.md)
- [Feature Resource Documentation](../../../docs/resources/feature.md)
- [Project Resource Documentation](../../../docs/resources/project.md)
- [Design Patterns](../../../docs/patterns/separation-of-concerns.md)
