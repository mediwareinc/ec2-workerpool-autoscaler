output "image" {
  description = "Image reference by tag"
  value       = local.image
}

output "image_digest_ref" {
  description = "Image reference pinned to the pushed digest"
  value       = data.google_artifact_registry_docker_image.built.self_link
}

output "source_hash" {
  description = "Hash of the source files the image was built from"
  value       = local.source_hash
}
