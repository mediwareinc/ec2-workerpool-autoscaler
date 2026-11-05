locals {
  source_root   = abspath("${path.module}/../..")
  registry_host = "${var.location}-docker.pkg.dev"
  image_repo    = "${local.registry_host}/${var.project_id}/${var.repository_id}/${var.image_name}"
  image         = "${local.image_repo}:${var.image_tag}"

  # Everything the Dockerfile compiles; tests are excluded as in .dockerignore.
  source_files = sort([
    for f in setunion(fileset(local.source_root, "cmd/**"), fileset(local.source_root, "internal/**"), ["go.mod", "go.sum"]) : f
    if !endswith(f, "_test.go")
  ])
  source_hash = sha256(join("\n", [for f in local.source_files : "${f} ${filesha256("${local.source_root}/${f}")}"]))
}

resource "google_project_service" "artifactregistry" {
  project            = var.project_id
  service            = "artifactregistry.googleapis.com"
  disable_on_destroy = false
}

resource "terraform_data" "build" {
  triggers_replace = [local.source_hash, local.image, var.dockerfile]

  provisioner "local-exec" {
    command = "bash ${path.module}/build-push.sh"
    environment = {
      CONTAINER_CLI = var.container_cli
      REGISTRY_HOST = local.registry_host
      IMAGE         = local.image
      SOURCE_ROOT   = local.source_root
      DOCKERFILE    = var.dockerfile
      SOURCE_HASH   = local.source_hash
    }
  }

  depends_on = [google_project_service.artifactregistry]
}

data "google_artifact_registry_docker_image" "built" {
  project       = var.project_id
  location      = var.location
  repository_id = var.repository_id
  image_name    = "${var.image_name}:${var.image_tag}"

  depends_on = [terraform_data.build]
}
