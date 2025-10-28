# Resource URI Examples

This document provides comprehensive examples of using the TofuKit resource URI system.

## Basic URI Formats

### Simple Resources

```hcl
# Feature URI
tofukit://feature/structured-logging

# File URI
tofukit://file/gitignore

# Stack URI
tofukit://stack/python-click-app
```

### Kit Resources (with subtype)

```hcl
# Language Kit
tofukit://kit/language/python

# Framework Kit
tofukit://kit/framework/click

# Tool Kit
tofukit://kit/tool/pytest

# Methodology Kit
tofukit://kit/methodology/tdd

# Style Kit
tofukit://kit/style/pep8

# Infrastructure Kit
tofukit://kit/infrastructure/docker

# Integration Kit
tofukit://kit/integration/github
```

## Complete Examples

### Example 1: Referencing a Feature in Project Requirements

```hcl
# Define a reusable feature
resource "tofukit_feature" "logging" {
  name   = "structured-logging"
  prompt = "Add structured logging with multiple output formats"

  files = {
    "logger.py" = {
      content = <<-PYTHON
        import logging
        import json

        class StructuredLogger:
            def __init__(self, name):
                self.logger = logging.getLogger(name)
      PYTHON
    }
  }

  verifications = [{
    command = "python -c 'import logger; print(logger.StructuredLogger)'"
    expect  = "class"
  }]
}

# Reference the feature in a project
resource "tofukit_project" "api_service" {
  name = "api-service"

  requirements = [{
    name = "Logging Setup"
    instructions = [{
      # Use the .link attribute to reference the feature
      prompt = <<-EOT
        Integrate ${tofukit_feature.logging.link} into the API service.
        Ensure all HTTP requests are logged with structured JSON format.
      EOT
    }]
  }]
}
```

### Example 2: Sharing Common Files Across Projects

```hcl
# Define reusable file resources
resource "tofukit_file" "gitignore_python" {
  name = ".gitignore"
  content = <<-GITIGNORE
    __pycache__/
    *.py[cod]
    *$py.class
    *.so
    .Python
    build/
    dist/
    *.egg-info/
    .env
    venv/
  GITIGNORE
}

resource "tofukit_file" "dockerfile_python" {
  name = "Dockerfile"
  content = <<-DOCKERFILE
    FROM python:3.12-slim
    WORKDIR /app
    COPY requirements.txt .
    RUN pip install -r requirements.txt
    COPY . .
    CMD ["python", "main.py"]
  DOCKERFILE
}

# Use the files in multiple projects
resource "tofukit_project" "web_app" {
  name = "web-app"

  files = {
    ".gitignore" = tofukit_file.gitignore_python
    "Dockerfile" = tofukit_file.dockerfile_python
  }

  requirements = [{
    name = "Project Structure"
    instructions = [{
      prompt = <<-EOT
        Create a Flask web application.
        Use ${tofukit_file.gitignore_python.link} for Git ignore rules.
        Use ${tofukit_file.dockerfile_python.link} for containerization.
      EOT
    }]
  }]
}

resource "tofukit_project" "api_service" {
  name = "api-service"

  files = {
    ".gitignore" = tofukit_file.gitignore_python
    "Dockerfile" = tofukit_file.dockerfile_python
  }

  requirements = [{
    name = "FastAPI Service"
    instructions = [{
      prompt = "Create a FastAPI service with the shared configuration files"
    }]
  }]
}
```

### Example 3: Building on Top of Stacks

```hcl
# Define a base stack
resource "tofukit_stack" "fastapi_base" {
  name = "fastapi-base"

  files = {
    "main.py" = {
      content = <<-PYTHON
        from fastapi import FastAPI

        app = FastAPI()

        @app.get("/")
        async def root():
            return {"message": "Hello World"}
      PYTHON
    }

    "requirements.txt" = {
      content = <<-TXT
        fastapi==0.104.1
        uvicorn[standard]==0.24.0
      TXT
    }
  }
}

# Reference the stack in a project with additional features
resource "tofukit_project" "user_api" {
  name = "user-api"
  stack = tofukit_stack.fastapi_base.id

  requirements = [{
    name = "Extend Base Stack"
    instructions = [{
      prompt = <<-EOT
        Starting from ${tofukit_stack.fastapi_base.link}, add:
        - User authentication endpoints
        - Database models with SQLAlchemy
        - JWT token handling
        - Password hashing with bcrypt
      EOT
    }]
  }]
}
```

### Example 4: Composing Features with Kit Dependencies

```hcl
# Define a feature that references kits
resource "tofukit_language" "python312" {
  name    = "python312"
  version = "3.12"
}

resource "tofukit_framework" "fastapi" {
  name    = "fastapi"
  version = "0.104.1"
}

resource "tofukit_feature" "api_authentication" {
  name   = "jwt-authentication"
  prompt = "Add JWT-based authentication"

  constraints = [
    "Must use ${tofukit_language.python312.link}",
    "Must integrate with ${tofukit_framework.fastapi.link}"
  ]

  files = {
    "auth.py" = {
      instructions = [{
        prompt = <<-EOT
          Create JWT authentication middleware compatible with FastAPI.
          Use python-jose for JWT handling.
        EOT
      }]
    }
  }

  verifications = [{
    command = "python -c 'import auth; print(auth.create_access_token)'"
    expect  = "function"
  }]
}

# Use the feature in a project
resource "tofukit_project" "secure_api" {
  name = "secure-api"

  requirements = [{
    name = "Security"
    instructions = [{
      prompt = <<-EOT
        Implement ${tofukit_feature.api_authentication.link} across all endpoints.
        Protect all routes except /login and /health.
      EOT
    }]
  }]
}
```

### Example 5: Cross-Feature Dependencies

```hcl
# Define interconnected features
resource "tofukit_feature" "database" {
  name   = "postgres-database"
  prompt = "Setup PostgreSQL database connection"

  files = {
    "database.py" = {
      instructions = [{
        prompt = "Create SQLAlchemy database connection with async support"
      }]
    }
  }
}

resource "tofukit_feature" "caching" {
  name   = "redis-caching"
  prompt = "Setup Redis caching layer"

  files = {
    "cache.py" = {
      instructions = [{
        prompt = "Create Redis cache manager with connection pooling"
      }]
    }
  }
}

resource "tofukit_feature" "user_service" {
  name   = "user-service"
  prompt = "User management service with caching"

  constraints = [
    "Requires ${tofukit_feature.database.link} for persistence",
    "Requires ${tofukit_feature.caching.link} for performance"
  ]

  files = {
    "services/user_service.py" = {
      instructions = [{
        prompt = <<-EOT
          Create user service that:
          - Uses ${tofukit_feature.database.link} for CRUD operations
          - Uses ${tofukit_feature.caching.link} for frequently accessed users
          - Implements cache invalidation on updates
        EOT
      }]
    }
  }
}

# Use all features together
resource "tofukit_project" "user_management_api" {
  name = "user-management"

  features = {
    "database" = tofukit_feature.database
    "caching"  = tofukit_feature.caching
    "users"    = tofukit_feature.user_service
  }

  requirements = [{
    name = "Integration"
    instructions = [{
      prompt = <<-EOT
        Wire up all features:
        - ${tofukit_feature.database.link} connects on startup
        - ${tofukit_feature.caching.link} connects to Redis
        - ${tofukit_feature.user_service.link} uses both for operations
      EOT
    }]
  }]
}
```

### Example 6: Tool and Methodology References

```hcl
# Define development tools and methodologies
resource "tofukit_tool" "pytest" {
  name    = "pytest"
  version = "7.4.0"

  requirements = [{
    name = "Testing Framework"
    instructions = [{
      prompt = "Setup pytest with coverage and fixtures"
    }]
  }]
}

resource "tofukit_methodology" "tdd" {
  name = "test-driven-development"

  requirements = [{
    name = "TDD Workflow"
    instructions = [{
      prompt = <<-EOT
        Follow Test-Driven Development:
        1. Write failing test first
        2. Implement minimal code to pass
        3. Refactor while keeping tests green
      EOT
    }]
  }]
}

# Use in project with explicit methodology
resource "tofukit_project" "calculator" {
  name = "calculator-tdd"

  requirements = [
    {
      name = "Development Process"
      instructions = [{
        prompt = <<-EOT
          Use ${tofukit_methodology.tdd.link} to build a calculator library.
          Use ${tofukit_tool.pytest.link} for all tests.
        EOT
      }]
    },
    {
      name = "Calculator Implementation"
      instructions = [{
        prompt = <<-EOT
          Following ${tofukit_methodology.tdd.link}:
          1. Write tests for add, subtract, multiply, divide
          2. Implement calculator functions
          3. Ensure 100% test coverage with ${tofukit_tool.pytest.link}
        EOT
      }]
    }
  ]
}
```

### Example 7: Infrastructure and Integration Kits

```hcl
# Define infrastructure components
resource "tofukit_infrastructure" "docker" {
  name = "docker-compose"

  requirements = [{
    name = "Docker Setup"
    instructions = [{
      prompt = "Create docker-compose.yml with services for app, db, and redis"
    }]
  }]
}

resource "tofukit_integration" "github_actions" {
  name = "github-ci"

  requirements = [{
    name = "CI/CD Pipeline"
    instructions = [{
      prompt = <<-EOT
        Create GitHub Actions workflow:
        - Run tests on push
        - Build Docker image
        - Deploy to staging on main branch
      EOT
    }]
  }]
}

# Use in production-ready project
resource "tofukit_project" "production_api" {
  name = "production-ready-api"

  requirements = [
    {
      name = "Infrastructure"
      instructions = [{
        prompt = <<-EOT
          Setup ${tofukit_infrastructure.docker.link} for local development.
          Include services: api, postgres, redis.
        EOT
      }]
    },
    {
      name = "CI/CD"
      instructions = [{
        prompt = <<-EOT
          Implement ${tofukit_integration.github_actions.link} with:
          - Linting and type checking
          - Unit and integration tests
          - Docker build and push
          - Deployment to staging environment
        EOT
      }]
    }
  ]
}
```

### Example 8: Multi-Project Shared Configuration

```hcl
# Define shared configuration files
resource "tofukit_file" "editorconfig" {
  name = ".editorconfig"
  content = <<-EDITORCONFIG
    root = true

    [*]
    charset = utf-8
    end_of_line = lf
    insert_final_newline = true
    trim_trailing_whitespace = true

    [*.py]
    indent_style = space
    indent_size = 4
  EDITORCONFIG
}

resource "tofukit_file" "pre_commit_config" {
  name = ".pre-commit-config.yaml"
  content = <<-YAML
    repos:
      - repo: https://github.com/pre-commit/pre-commit-hooks
        rev: v4.5.0
        hooks:
          - id: trailing-whitespace
          - id: end-of-file-fixer
          - id: check-yaml
  YAML
}

resource "tofukit_style" "black" {
  name = "python-black"

  requirements = [{
    name = "Code Formatting"
    instructions = [{
      prompt = "Configure Black formatter with line-length=88, target-version=py312"
    }]
  }]
}

# Apply to multiple projects
locals {
  shared_files = {
    ".editorconfig"     = tofukit_file.editorconfig
    ".pre-commit-config.yaml" = tofukit_file.pre_commit_config
  }
}

resource "tofukit_project" "frontend" {
  name = "frontend-app"
  files = local.shared_files

  requirements = [{
    name = "Code Quality"
    instructions = [{
      prompt = "Apply shared configuration files and ${tofukit_style.black.link}"
    }]
  }]
}

resource "tofukit_project" "backend" {
  name = "backend-api"
  files = local.shared_files

  requirements = [{
    name = "Code Quality"
    instructions = [{
      prompt = "Apply shared configuration files and ${tofukit_style.black.link}"
    }]
  }]
}
```

## How URIs Are Resolved

When Claude receives a prompt with URIs, the provider:

1. **Scans all string fields** (prompts, instructions, constraints, descriptions)
2. **Extracts URIs** using regex pattern
3. **Resolves each URI** to resource metadata from the global registry
4. **Builds resource registry** map of URI → metadata
5. **Passes to Claude** in the `resource_registry` field
6. **Enhances system prompt** with instructions on using resource references

### Example Registry Output

For a project referencing `tofukit://feature/logging`, Claude receives:

```json
{
  "type": "project",
  "project_info": {
    "name": "my-app"
  },
  "resource_registry": {
    "tofukit://feature/logging": {
      "type": "feature",
      "name": "logging",
      "description": "",
      "prompt": "",
      "files": [],
      "file_count": 0
    }
  },
  "specification": {
    "requirements": [
      {
        "name": "Setup",
        "instructions": [
          {
            "prompt": "Use tofukit://feature/logging for application logging"
          }
        ]
      }
    ]
  }
}
```

## Best Practices

### 1. Use Descriptive Resource Names

```hcl
# Good
resource "tofukit_feature" "structured_json_logging" {
  name = "structured-json-logging"
}

# Avoid
resource "tofukit_feature" "log" {
  name = "log"
}
```

### 2. Reference URIs in Constraints

```hcl
resource "tofukit_feature" "api_endpoints" {
  name = "rest-api"

  constraints = [
    "Must use ${tofukit_feature.authentication.link} for protected endpoints",
    "Must integrate with ${tofukit_feature.database.link} for persistence"
  ]
}
```

### 3. Document Dependencies Explicitly

```hcl
resource "tofukit_project" "microservice" {
  name = "user-service"

  requirements = [{
    name = "Dependencies"
    instructions = [{
      prompt = <<-EOT
        This service depends on:
        - ${tofukit_feature.database.link} - PostgreSQL connection
        - ${tofukit_feature.caching.link} - Redis caching layer
        - ${tofukit_feature.messaging.link} - RabbitMQ message queue

        Ensure all dependencies are initialized before starting the service.
      EOT
    }]
  }]
}
```

### 4. Combine with Terraform Features

```hcl
# Use locals for reusable URI sets
locals {
  core_features = [
    tofukit_feature.logging.link,
    tofukit_feature.config.link,
    tofukit_feature.error_handling.link
  ]
}

resource "tofukit_project" "service_a" {
  name = "service-a"

  requirements = [{
    name = "Core Features"
    instructions = [{
      prompt = "Integrate core features: ${join(", ", local.core_features)}"
    }]
  }]
}
```

## URI Validation

URIs are validated at multiple stages:

1. **Regex Pattern Matching** - URIs must match `tofukit://[a-z]+(/[a-z]+)?/[a-zA-Z0-9_-]+`
2. **Resource Existence** - Referenced resources must exist in the global registry
3. **Type Validation** - Resource type must be recognized (feature, file, stack, kit/*)

Invalid URIs will cause warnings but won't block project execution - Claude will proceed without the resource metadata.
