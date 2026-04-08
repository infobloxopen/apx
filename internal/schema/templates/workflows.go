package templates

import (
	"fmt"
	"strings"
)

// GenerateCanonicalCI generates .github/workflows/ci.yml for canonical repos.
// This runs lint and breaking-change checks on every PR.
func GenerateCanonicalCI() string {
	return `name: APX Schema CI

on:
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install APX
        uses: infobloxopen/apx@main

      - name: Lint schemas
        run: apx lint

      - name: Check for breaking changes
        run: apx breaking --against origin/main

      - name: Check third-party protos are in sync
        run: |
          ./scripts/sync-third-party.sh --check-only 2>&1 | tee /tmp/sync-check.txt
          if grep -qE '^\s+[+~-] ' /tmp/sync-check.txt; then
            echo "::error::Third-party protos are out of sync. Run ./scripts/sync-third-party.sh and commit the results."
            exit 1
          fi
`
}

// GenerateCanonicalOnMerge generates .github/workflows/on-merge.yml for
// canonical repos. On push to main it validates, generates catalog data,
// builds a Docker image with OCI labels, pushes to GHCR, and attests the build.
func GenerateCanonicalOnMerge(org string) string {
	// OCI image references must be lowercase; GitHub orgs may have mixed case.
	imageOrg := strings.ToLower(org)
	return fmt.Sprintf(`name: APX On Merge

on:
  push:
    branches: [main]

permissions:
  contents: read
  packages: write
  id-token: write
  attestations: write

jobs:
  catalog:
    runs-on: ubuntu-latest
    env:
      IMAGE: ghcr.io/%s/${{ github.event.repository.name }}-catalog
    steps:
      - name: Generate App Token
        id: app-token
        uses: actions/create-github-app-token@v1
        with:
          app-id: ${{ secrets.APX_APP_ID }}
          private-key: ${{ secrets.APX_APP_PRIVATE_KEY }}
          owner: ${{ github.repository_owner }}

      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
          token: ${{ steps.app-token.outputs.token }}

      - name: Install APX
        uses: infobloxopen/apx@main

      - name: Validate schemas
        run: apx lint

      - name: Generate catalog data
        run: apx catalog generate --output catalog/catalog.yaml

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ steps.app-token.outputs.token }}

      - name: Build catalog image
        run: |
          docker build \
            --build-arg CREATED="$(date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ)" \
            --build-arg REVISION="${{ github.sha }}" \
            --build-arg SOURCE="https://github.com/${{ github.repository }}" \
            --build-arg VERSION="${{ github.sha }}" \
            -t "$IMAGE:latest" \
            -t "$IMAGE:sha-${GITHUB_SHA::7}" \
            catalog/

      - name: Push catalog image
        id: push
        run: |
          docker push "$IMAGE:latest"
          docker push "$IMAGE:sha-${GITHUB_SHA::7}"
          DIGEST=$(docker inspect --format='{{index .RepoDigests 0}}' "$IMAGE:latest" | cut -d@ -f2)
          echo "digest=$DIGEST" >> "$GITHUB_OUTPUT"

      - name: Attest build provenance
        uses: actions/attest-build-provenance@v2
        with:
          subject-name: ${{ env.IMAGE }}
          subject-digest: ${{ steps.push.outputs.digest }}
          push-to-registry: true

      - name: Generate SBOM
        uses: anchore/sbom-action@v0
        with:
          image: ${{ env.IMAGE }}:latest
          output-file: sbom.spdx.json

      - name: Attest SBOM
        uses: actions/attest-sbom@v2
        with:
          subject-name: ${{ env.IMAGE }}
          subject-digest: ${{ steps.push.outputs.digest }}
          sbom-path: sbom.spdx.json
          push-to-registry: true
`, imageOrg)
}

// GenerateAppRelease generates .github/workflows/apx-release.yml for app
// repos. On tag push matching the APX tag pattern it releases to canonical.
func GenerateAppRelease(org, canonicalRepo string) string {
	return fmt.Sprintf(`name: APX Release

on:
  push:
    tags:
      - "proto/**/v[0-9]*"
      - "openapi/**/v[0-9]*"
      - "avro/**/v[0-9]*"
      - "jsonschema/**/v[0-9]*"
      - "parquet/**/v[0-9]*"

permissions:
  contents: read

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Generate App Token
        id: app-token
        uses: actions/create-github-app-token@v1
        with:
          app-id: ${{ secrets.APX_APP_ID }}
          private-key: ${{ secrets.APX_APP_PRIVATE_KEY }}
          owner: %s
          repositories: %s

      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install APX
        uses: infobloxopen/apx@main

      - name: Parse tag
        id: tag
        run: |
          TAG="${GITHUB_REF#refs/tags/}"
          echo "tag=${TAG}" >> "$GITHUB_OUTPUT"

      - name: Validate
        run: |
          apx lint
          apx breaking --against HEAD^ || true

      - name: Release to canonical repo
        env:
          GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}
        run: |
          apx release submit \
            --tag="${{ steps.tag.outputs.tag }}" \
            --canonical-repo=github.com/%s/%s
`, org, canonicalRepo, org, canonicalRepo)
}
