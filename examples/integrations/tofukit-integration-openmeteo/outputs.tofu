output "integration" {
  description = "The complete Open-Meteo integration resource"
  value       = tofukit_integration.open_meteo
}

output "link" {
  description = "URI link to the Open-Meteo integration (tofukit://integration/open-meteo)"
  value       = tofukit_integration.open_meteo.link
}

output "id" {
  description = "Integration resource ID"
  value       = tofukit_integration.open_meteo.id
}

output "name" {
  description = "Integration name"
  value       = tofukit_integration.open_meteo.name
}
