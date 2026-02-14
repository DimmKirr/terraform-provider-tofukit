# TofuKit Provider for OpenTofu

This is a custom provider for OpenTofu that implements the TofuKit component system.

The goal is to provide a way to define and manage components of a software system generated via LLM in a declarative way.

## Documentation

Full documentation for all resources and data sources is available in the [docs](./docs/) directory:

- **Resources**: See [docs/resources/](./docs/resources/) for detailed schema and usage examples
- **Data Sources**: See [docs/data-sources/](./docs/data-sources/) for query capabilities
- **Provider Configuration**: See [docs/index.md](./docs/index.md) for provider setup

To regenerate documentation after schema changes:

```bash
task docs
```
