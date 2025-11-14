output "link" {
  description = "URI link to the OpenTofu tool resource (tofukit://tool/opentofu)"
  value       = tofukit_tool.opentofu.link
}

output "id" {
  description = "Tool resource ID"
  value       = tofukit_tool.opentofu.id
}

output "name" {
  description = "Tool name"
  value       = tofukit_tool.opentofu.name
}
