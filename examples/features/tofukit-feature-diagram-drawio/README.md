# Draw.io Diagram Generation Feature

Reusable TofuKit feature that generates professional architecture diagrams in draw.io format from structured component specifications.

## Overview

This feature module accepts a structured description of your system architecture (services, databases, integrations, queues, infrastructure) and generates a visual diagram in draw.io XML format. The diagram is automatically exported to PNG for documentation purposes.

## Prerequisites

**draw.io CLI** must be installed on your system:

- **macOS**: `brew install --cask drawio`
- **Linux**: Download from [drawio-desktop releases](https://github.com/jgraph/drawio-desktop/releases)
- **Windows**: Download installer from [drawio-desktop releases](https://github.com/jgraph/drawio-desktop/releases)

Verify installation:
```bash
drawio --version
```

## Module Inputs

### `file_path` (optional)
Output path for diagram file (without `.drawio` extension).

- **Type**: `string`
- **Default**: `"architecture"`
- **Example**: `"docs/system-architecture"`

### `project` (required)
Structured object describing your system architecture, grouped by component type.

**Structure**:
```hcl
project = {
  infrastructure = [{ name, type, description }]  # Load balancers, proxies, CDN
  services       = [{ name, type, description }]  # Web servers, APIs, microservices
  databases      = [{ name, type, description }]  # SQL, NoSQL, caches, search engines
  integrations   = [{ name, type, description }]  # External APIs, SaaS, cloud services
  queues         = [{ name, type, description }]  # Message queues, pub/sub, streams
}
```

**Component Types**:

| Category | Types |
|----------|-------|
| `infrastructure` | `load_balancer`, `proxy`, `cdn`, `gateway` |
| `services` | `web_server`, `api`, `microservice` |
| `databases` | `sql`, `nosql`, `cache`, `search` |
| `integrations` | `rest_api`, `graphql_api`, `saas`, `cloud_service` |
| `queues` | `queue`, `pub_sub`, `stream` |

## Module Outputs

### `feature`
The `tofukit_feature` resource that can be referenced in projects.

## Usage Example

```hcl
module "architecture_diagram" {
  source = "../../stacks/tofukit-feature-diagram-drawio"

  file_path = "docs/system-architecture"

  project = {
    infrastructure = [
      { name = "Nginx Load Balancer", type = "load_balancer" }
    ]

    services = [
      { name = "Flask API", type = "web_server" },
      { name = "Auth Service", type = "microservice" }
    ]

    databases = [
      { name = "PostgreSQL", type = "sql" },
      { name = "Redis", type = "cache" }
    ]

    integrations = [
      { name = "Stripe API", type = "rest_api" },
      { name = "SendGrid", type = "saas" }
    ]

    queues = [
      { name = "RabbitMQ", type = "queue" }
    ]
  }
}

resource "tofukit_project" "my_app" {
  name = "my-application"

  features = {
    "diagram" = module.architecture_diagram.feature
  }
}
```

## Generated Files

After running `tofu apply`, the following files will be generated:

1. **`<file_path>.drawio`** - Draw.io XML diagram file (can be opened in draw.io desktop)
2. **`<file_path>.drawio.png`** - PNG export of the diagram (for documentation)

## How It Works

1. **Component Grouping**: Components are organized by type (infrastructure, services, databases, integrations, queues)
2. **JSON Serialization**: The `project` object is converted to JSON and passed to Claude in the prompt
3. **Diagram Generation**: Claude generates valid draw.io XML with appropriate shapes for each component type
4. **Layout**: Components are arranged in logical tiers (top to bottom: infrastructure → services → databases → integrations → queues)
5. **PNG Export**: draw.io CLI exports the diagram to PNG for easy embedding in documentation

## Diagram Conventions

- **Infrastructure**: Trapezoid or rectangle shapes (load balancers, proxies)
- **Services**: Rounded rectangles (web servers, APIs)
- **Databases**: Cylinder shapes (SQL, NoSQL, caches)
- **Integrations**: Cloud or hexagon shapes (external APIs, SaaS)
- **Queues**: Rectangles with queue icons (message queues, streams)
- **Arrows**: Show data flow and request/response patterns
- **Colors**: Professional blue/gray scheme

## Troubleshooting

### "drawio: command not found"

Install draw.io CLI:
```bash
# macOS
brew install --cask drawio

# Linux (Debian/Ubuntu)
wget https://github.com/jgraph/drawio-desktop/releases/download/v24.7.8/drawio-amd64-24.7.8.deb
sudo dpkg -i drawio-amd64-24.7.8.deb
```

### Diagram file exists but PNG export fails

Verify draw.io CLI has proper permissions and supports export:
```bash
drawio --version
drawio -x -f png --help
```

### Empty or invalid diagram

Check Claude debug output in `.debug/` directory (if `debug = true` in provider config) to see the generated XML.

## See Also

- [3-Tier App Example](../../projects/infra-3-tier-app/) - Simple usage example
- [Draw.io Documentation](https://www.drawio.com/doc/)
- [TofuKit Provider Documentation](../../../CLAUDE.md)
