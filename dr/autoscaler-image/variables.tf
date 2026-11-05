variable "project_id" {
  description = "Project holding the container registry"
  type        = string
}

variable "location" {
  description = "Artifact Registry location of the repository"
  type        = string
  default     = "us"
}

variable "repository_id" {
  description = "Docker Artifact Registry repository to push to"
  type        = string
  default     = "images"
}

variable "image_name" {
  description = "Image name within the repository"
  type        = string
  default     = "spacelift-workerpool-autoscaler"
}

variable "image_tag" {
  description = "Tag the image is pushed under"
  type        = string
  default     = "dr"
}

variable "dockerfile" {
  description = "Dockerfile path, relative to the repository root (the build context)"
  type        = string
  default     = "cmd/cloudrun/Dockerfile"
}

variable "container_cli" {
  description = "Docker-compatible CLI used to build and push (docker or podman)"
  type        = string
  default     = "docker"
}
