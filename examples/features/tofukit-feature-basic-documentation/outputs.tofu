# Outputs for Basic Documentation Feature Module

output "feature" {
  description = "The basic documentation feature resource"
  value       = tofukit_feature.basic_documentation
}

output "feature_id" {
  description = "The ID of the basic documentation feature"
  value       = tofukit_feature.basic_documentation.id
}

output "feature_link" {
  description = "The URI link to reference this feature"
  value       = tofukit_feature.basic_documentation.link
}
