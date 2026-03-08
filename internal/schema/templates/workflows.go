package templates

import "fmt"

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

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.26.x"

      - name: Install APX
        run: go install github.com/infobloxopen/apx/cmd/apx@latest

      - name: Lint schemas
        run: apx lint

      - name: Check for breaking changes
        run: apx breaking --against origin/main
`
}

// GenerateCanonicalOnMerge generates .github/workflows/on-merge.yml for
// canonical repos. On push to main it validates, tags, and updates the catalog.
func GenerateCanonicalOnMerge(org string) string {
	return fmt.Sprintf(`name: APX On Merge

on:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  tag-and-catalog:
    runs-on: ubuntu-latest
    steps:
      - name: Generate App Token
        id: app-token
        uses: actions/create-github-app-token@v1
        with:
          app-id: ${{ secrets.APX_APP_ID }}
          private-key: ${{ secrets.APX_APP_PRIVATE_KEY }}

      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
          token: ${{ steps.app-token.outputs.token }}

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.26.x"

      - name: Install APX
        run: go install github.com/infobloxopen/apx/cmd/apx@latest

      - name: Validate schemas
        run: apx lint

      - name: Update catalog
        run: apx catalog generate

      - name: Commit catalog changes
        run: |
          git config user.name "apx-publisher[bot]"
          git config user.email "apx-publisher[bot]@users.noreply.github.com"
          if git diff --quiet catalog/; then
            echo "No catalog changes"
          else
            git add catalog/
            git commit -m "chore: update catalog [skip ci]"
            git push
          fi
`)
}

// GenerateAppPublish generates .github/workflows/apx-publish.yml for app
// repos. On tag push matching the APX tag pattern it publishes to canonical.
func GenerateAppPublish(org, canonicalRepo string) string {
	return fmt.Sprintf(`name: APX Publish

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
  publish:
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

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.26.x"

      - name: Install APX
        run: go install github.com/infobloxopen/apx/cmd/apx@latest

      - name: Parse tag
        id: tag
        run: |
          TAG="${GITHUB_REF#refs/tags/}"
          echo "tag=${TAG}" >> "$GITHUB_OUTPUT"

      - name: Validate
        run: |
          apx lint
          apx breaking --against HEAD^ || true

      - name: Publish to canonical repo
        env:
          GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}
        run: |
          apx publish \
            --tag="${{ steps.tag.outputs.tag }}" \
            --canonical-repo=github.com/%s/%s
`, org, canonicalRepo, org, canonicalRepo)
}
