#!/usr/bin/env bash
# Builds the autoscaler image from the repository checkout and pushes it, authenticating
# to Artifact Registry with the Application Default Credentials in the environment.
set -euo pipefail

fail() { echo "ERROR: $*" >&2; exit 1; }

: "${CONTAINER_CLI:?}" "${REGISTRY_HOST:?}" "${IMAGE:?}" "${SOURCE_ROOT:?}" "${DOCKERFILE:?}" "${SOURCE_HASH:?}"

command -v "$CONTAINER_CLI" >/dev/null || fail "'$CONTAINER_CLI' not found on PATH (set var.container_cli to docker or podman)"
command -v gcloud >/dev/null || fail "gcloud not found on PATH (needed to mint a token from the ADC)"
[ -f "$SOURCE_ROOT/$DOCKERFILE" ] || fail "$SOURCE_ROOT/$DOCKERFILE does not exist"

token=$(gcloud auth application-default print-access-token) \
  || fail "cannot get an access token from the Application Default Credentials (GOOGLE_APPLICATION_CREDENTIALS)"

# Isolated credential store so the token never touches the user's registry logins.
auth_dir=$(mktemp -d)
trap 'rm -rf "$auth_dir"' EXIT
export REGISTRY_AUTH_FILE="$auth_dir/auth.json" DOCKER_CONFIG="$auth_dir"

printf '%s' "$token" | "$CONTAINER_CLI" login --username oauth2accesstoken --password-stdin "$REGISTRY_HOST" \
  || fail "login to $REGISTRY_HOST failed"

"$CONTAINER_CLI" build --platform linux/amd64 \
  --label "source-hash=$SOURCE_HASH" \
  -f "$SOURCE_ROOT/$DOCKERFILE" -t "$IMAGE" "$SOURCE_ROOT" \
  || fail "build of $IMAGE failed"

"$CONTAINER_CLI" push "$IMAGE" \
  || fail "push of $IMAGE failed: the repository must exist, and the identity behind the ADC must be able to write to it (the DR registry grants writers to the gcp-service-accounts and gcp-organization-admins groups of the DR org domain)"
