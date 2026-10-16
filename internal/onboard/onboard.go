package onboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Options holds all the configuration for onboarding an existing service.
type Options struct {
	APIID         string // e.g. "proto/infoblox.dev/host-activation/v1"
	CanonicalRepo string // e.g. "github.com/Infoblox-CTO/apis"
	SchemaDir     string // directory to run buf build on (e.g. "simpleproto")
	ProtoFile     string // specific proto file for atlas-gentool (e.g. "pkg/pb/service.proto")
	SchemaFile    string // path to schema file for non-FDS formats (e.g. "api/kafka-metric-message.schema.json")
	Swagger       string // path to swagger file for openapi module (e.g. "pkg/pb/service.swagger.json")
	Lifecycle     string // default: "beta"
	Audience      string // default: "public"
	FDSOutput     string // default: "api-v1.binpb"
	DefaultBranch string // repo default branch (e.g. "main", "master"); used in push trigger
	GoPrivate     string // GOPRIVATE pattern for private module access (e.g. "github.com/Infoblox-CTO/*")
	Org           string
	Repo          string
}

// Result reports what was created/modified/skipped.
type Result struct {
	Created  []string
	Modified []string
	Skipped  []string
}

// Scaffolder generates the files needed to onboard an existing service.
type Scaffolder struct {
	opts Options
}

// New creates a new onboard scaffolder.
func New(opts Options) *Scaffolder {
	return &Scaffolder{opts: opts}
}

// Generate writes files to baseDir (typically ".").
func (s *Scaffolder) Generate(baseDir string) (*Result, error) {
	result := &Result{}

	if err := s.generateApxYaml(baseDir, result); err != nil {
		return nil, fmt.Errorf("apx.yaml: %w", err)
	}
	if err := s.generatePublishManifest(baseDir, result); err != nil {
		return nil, fmt.Errorf(".apx-publish.yaml: %w", err)
	}
	if err := s.patchMakefile(baseDir, result); err != nil {
		return nil, fmt.Errorf("Makefile: %w", err)
	}
	if err := s.generateWorkflows(baseDir, result); err != nil {
		return nil, fmt.Errorf("workflows: %w", err)
	}
	if err := s.appendGitignore(baseDir, result); err != nil {
		return nil, fmt.Errorf(".gitignore: %w", err)
	}

	return result, nil
}

func (s *Scaffolder) generateApxYaml(baseDir string, result *Result) error {
	path := filepath.Join(baseDir, "apx.yaml")
	if _, err := os.Stat(path); err == nil {
		result.Skipped = append(result.Skipped, "apx.yaml (already exists)")
		return nil
	}

	content := fmt.Sprintf(`version: 1
org: %s
repo: %s
module_roots:
  - build/apx-modules
release:
  tag_format: '{subdir}/v{version}'
  ci_only: true
tools:
  spectral:
    version: v6.11.0
  oasdiff:
    version: v1.9.6
execution:
  mode: local
`, s.opts.Org, s.opts.Repo)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	result.Created = append(result.Created, "apx.yaml")
	return nil
}

func (s *Scaffolder) generatePublishManifest(baseDir string, result *Result) error {
	path := filepath.Join(baseDir, ".apx-publish.yaml")

	format := inferFormat(s.opts.APIID)
	usesFDS := IsFDSFormat(s.opts.APIID)

	var modulesBlock strings.Builder

	fmt.Fprintf(&modulesBlock, "  - id: %s\n", s.opts.APIID)
	if usesFDS {
		fmt.Fprintf(&modulesBlock, "    fds: %s\n", s.opts.FDSOutput)
	} else if s.opts.SchemaFile != "" {
		// Field name matches the format prefix (e.g. "jsonschema:" for jsonschema/ modules).
		fmt.Fprintf(&modulesBlock, "    %s: %s\n", inferFormat(s.opts.APIID), filepath.ToSlash(s.opts.SchemaFile))
	}
	fmt.Fprintf(&modulesBlock, "    lifecycle: %s\n", s.opts.Lifecycle)
	fmt.Fprintf(&modulesBlock, "    audience: %s\n", s.opts.Audience)
	if format == "proto" {
		fmt.Fprintf(&modulesBlock, "    verify_clients: off    # proto FDS is not an OpenAPI spec (see infobloxopen/apx#43)\n")
	}

	// OpenAPI companion module (only meaningful alongside a proto/ FDS module)
	if s.opts.Swagger != "" && usesFDS {
		openapiID := openAPIIDFrom(s.opts.APIID)
		fmt.Fprintf(&modulesBlock, "  - id: %s\n", openapiID)
		fmt.Fprintf(&modulesBlock, "    swagger: %s\n", filepath.ToSlash(s.opts.Swagger))
		fmt.Fprintf(&modulesBlock, "    fds: %s\n", s.opts.FDSOutput)
		fmt.Fprintf(&modulesBlock, "    lifecycle: %s\n", s.opts.Lifecycle)
		fmt.Fprintf(&modulesBlock, "    audience: %s\n", s.opts.Audience)
	}

	defaultBranch := s.opts.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	var header strings.Builder
	fmt.Fprintf(&header, "canonical_repo: %s\n", s.opts.CanonicalRepo)
	fmt.Fprintf(&header, "forge: github\n")
	if usesFDS {
		fmt.Fprintf(&header, "fds_target: fds\n")
	}
	fmt.Fprintf(&header, "version_bump: minor\n\nbranch_targets:\n  %s: main\n\nmodules:\n", defaultBranch)

	content := header.String() + modulesBlock.String()

	existed := false
	if _, err := os.Stat(path); err == nil {
		existed = true
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	if existed {
		result.Modified = append(result.Modified, ".apx-publish.yaml")
	} else {
		result.Created = append(result.Created, ".apx-publish.yaml")
	}
	return nil
}

func (s *Scaffolder) patchMakefile(baseDir string, result *Result) error {
	if !IsFDSFormat(s.opts.APIID) {
		result.Skipped = append(result.Skipped, "Makefile fds target (not needed for "+inferFormat(s.opts.APIID)+" format)")
		return nil
	}

	path := filepath.Join(baseDir, "Makefile")

	tool := s.detectTool(baseDir)
	target := s.makeFDSTarget(tool)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(target), 0644); err != nil {
				return err
			}
			result.Created = append(result.Created, "Makefile")
			return nil
		}
		return err
	}

	content := string(data)
	if strings.Contains(content, "\nfds:") || strings.HasPrefix(content, "fds:") {
		result.Skipped = append(result.Skipped, "Makefile fds target (already exists)")
		return nil
	}

	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += "\n" + target
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	result.Modified = append(result.Modified, "Makefile (fds target added)")
	return nil
}

type fdsTool int

const (
	toolBuf fdsTool = iota
	toolAtlasGentool
	toolUnknown
)

func (s *Scaffolder) detectTool(baseDir string) fdsTool {
	// Prefer buf if buf.yaml is present in schema-dir or repo root.
	searchDirs := []string{baseDir}
	if s.opts.SchemaDir != "" {
		searchDirs = append(searchDirs, filepath.Join(baseDir, s.opts.SchemaDir))
	}
	for _, dir := range searchDirs {
		for _, name := range []string{"buf.yaml", "buf.yml"} {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				return toolBuf
			}
		}
	}

	// Fall back to atlas-gentool if Makefile* files reference it.
	for _, name := range []string{"Makefile", "Makefile.vars", "Makefile.common"} {
		data, err := os.ReadFile(filepath.Join(baseDir, name))
		if err != nil {
			continue
		}
		content := string(data)
		if strings.Contains(content, "atlas-gentool") || strings.Contains(content, "DOCKER_GENERATOR") {
			return toolAtlasGentool
		}
	}

	return toolUnknown
}

func (s *Scaffolder) makeFDSTarget(tool fdsTool) string {
	switch tool {
	case toolBuf:
		dir := "."
		if s.opts.SchemaDir != "" {
			dir = s.opts.SchemaDir
		}
		return fmt.Sprintf(".PHONY: fds\nfds:\n\tbuf build %s --as-file-descriptor-set -o %s\n",
			dir, s.opts.FDSOutput)

	case toolAtlasGentool:
		protoFile := s.opts.ProtoFile
		if protoFile == "" {
			protoFile = "path/to/service.proto  # TODO: set the correct proto file path"
		}
		// atlas-gentool resolves proto file paths relative to /go/src inside the
		// container, so the proto argument uses $(PROJECT_ROOT)/... not $(SRCROOT_IN_CONTAINER)/...
		// -I=$(PROJECT_ROOT)/vendor covers imports vendored alongside the service.
		// GENERATOR, SRCROOT_IN_CONTAINER, and PROJECT_ROOT are defined in Makefile.vars.
		return fmt.Sprintf(`.PHONY: fds
fds:
	$(GENERATOR) \
		-I=$(PROJECT_ROOT)/vendor \
		--include_imports \
		--descriptor_set_out=$(SRCROOT_IN_CONTAINER)/%s \
		$(PROJECT_ROOT)/%s
`, s.opts.FDSOutput, filepath.ToSlash(protoFile))

	default:
		return fmt.Sprintf(`.PHONY: fds
fds:
	# TODO: generate FileDescriptorSet — edit this target to match your build tool.
	# See: https://infobloxopen.github.io/apx/getting-started/brownfield-onboarding/
	@echo "fds target not configured" && exit 1
`)
	}
}

func (s *Scaffolder) generateWorkflows(baseDir string, result *Result) error {
	dir := filepath.Join(baseDir, ".github", "workflows")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create .github/workflows: %w", err)
	}

	defaultBranch := s.opts.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	schemaGlob := schemaPathGlob(s.opts.SchemaDir, s.opts.ProtoFile, s.opts.SchemaFile, s.opts.APIID)

	// Atlas-gentool repos with a go.mod need an inline workflow so that
	// go mod vendor can run with private-repo auth before make fds.
	// Non-FDS formats (jsonschema, avro, parquet, crd) never need this.
	needsVendorSetup := IsFDSFormat(s.opts.APIID) &&
		s.detectTool(baseDir) == toolAtlasGentool &&
		fileExists(filepath.Join(baseDir, "go.mod"))
	goPrivate := s.opts.GoPrivate

	var pr, publish string
	if needsVendorSetup {
		pr = generateInlinePRWorkflow(schemaGlob, goPrivate)
		publish = generateInlinePublishWorkflow(schemaGlob, defaultBranch, goPrivate)
	} else {
		pr = generatePRWorkflow(schemaGlob)
		publish = generatePublishWorkflow(schemaGlob, defaultBranch)
	}

	prPath := filepath.Join(dir, "api-catalog-pr.yml")
	if err := os.WriteFile(prPath, []byte(pr), 0644); err != nil {
		return err
	}
	result.Created = append(result.Created, ".github/workflows/api-catalog-pr.yml")

	publishPath := filepath.Join(dir, "api-catalog-publish.yml")
	if err := os.WriteFile(publishPath, []byte(publish), 0644); err != nil {
		return err
	}
	result.Created = append(result.Created, ".github/workflows/api-catalog-publish.yml")

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *Scaffolder) appendGitignore(baseDir string, result *Result) error {
	path := filepath.Join(baseDir, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	content := string(data)
	entries := []struct{ line, comment string }{
		{"build/apx-modules/", "# APX publish-on-change artifacts"},
		{".apx-release.yaml", "# APX release manifest (ephemeral)"},
	}
	if IsFDSFormat(s.opts.APIID) {
		entries = append([]struct{ line, comment string }{
			{s.opts.FDSOutput, "# APX FDS artifact (generated by make fds)"},
		}, entries...)
	}

	added := false
	for _, e := range entries {
		if !strings.Contains(content, e.line) {
			if !strings.HasSuffix(content, "\n") && content != "" {
				content += "\n"
			}
			content += e.comment + "\n" + e.line + "\n"
			added = true
		}
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	if added {
		result.Modified = append(result.Modified, ".gitignore")
	} else {
		result.Skipped = append(result.Skipped, ".gitignore (entries already present)")
	}
	return nil
}

// InferAPIIDFromPath attempts to derive a canonical API ID from a local path.
// Given "canary-scaffold/apis/proto/infoblox.dev/svc/v1", returns
// "proto/infoblox.dev/svc/v1". Returns empty string if no known format is found.
func InferAPIIDFromPath(path string) string {
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")
	known := map[string]bool{"proto": true, "openapi": true, "avro": true, "jsonschema": true, "parquet": true, "crd": true}
	for i, p := range parts {
		if known[p] {
			return strings.Join(parts[i:], "/")
		}
	}
	return ""
}

// DetectSwagger looks for a swagger/OpenAPI file in dir and returns the first match.
func DetectSwagger(dir string) string {
	if dir == "" {
		return ""
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".swagger.json") || strings.HasSuffix(name, ".swagger.yaml") ||
			strings.HasSuffix(name, ".openapi.yaml") || strings.HasSuffix(name, ".openapi.json") {
			return filepath.Join(dir, name)
		}
	}
	return ""
}

func inferFormat(apiID string) string {
	parts := strings.SplitN(apiID, "/", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// IsFDSFormat returns true for formats that compile a FileDescriptorSet (proto, openapi).
// Other formats (jsonschema, avro, parquet, crd) reference their schema file directly.
func IsFDSFormat(apiID string) bool {
	f := inferFormat(apiID)
	return f == "proto" || f == "openapi"
}

// openAPIIDFrom returns an openapi/... ID derived from a proto/... API ID.
// "proto/infoblox.dev/host-activation/v1" → "openapi/infoblox.dev/host-activation/v1"
func openAPIIDFrom(protoID string) string {
	parts := strings.SplitN(protoID, "/", 2)
	if len(parts) == 2 {
		return "openapi/" + parts[1]
	}
	return "openapi/" + protoID
}

func schemaPathGlob(schemaDir, protoFile, schemaFile, apiID string) string {
	if schemaFile != "" {
		return filepath.ToSlash(schemaFile)
	}
	if schemaDir != "" {
		return filepath.ToSlash(schemaDir) + "/**"
	}
	if protoFile != "" {
		return filepath.ToSlash(filepath.Dir(protoFile)) + "/**"
	}
	// Format-aware fallback glob
	switch inferFormat(apiID) {
	case "jsonschema":
		return "**/*.schema.json"
	case "avro":
		return "**/*.avsc"
	case "crd":
		return "api/**"
	default:
		return "**/*.proto"
	}
}

// generateInlinePRWorkflow produces a fully-inlined PR check workflow for
// repos that need go mod vendor with private-repo auth before make fds.
//
// An inline workflow (not a reusable workflow call) is required here because:
//   - The vendor step must run BEFORE make fds, which the reusable workflow
//     calls internally via apx-publish.sh
//   - ~/.netrc with GITPAT is kept active for the whole run so that
//     vendor-atlas (a Makefile prerequisite of make fds) can also authenticate
//   - The canonical catalog is cloned explicitly so apx-publish.sh can be
//     invoked with --canonical-dir
func generateInlinePRWorkflow(schemaGlob, goPrivate string) string {
	return fmt.Sprintf(`name: API catalog — PR check
on:
  pull_request:
    paths:
      - '%s'
      - '.apx-publish.yaml'
      - 'apx.yaml'
      - 'Makefile'
permissions:
  contents: read
  pull-requests: write
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Configure private Go module auth
        run: printf 'machine github.com login x-oauth-basic password %%s\n' "${GITPAT}" > "$HOME/.netrc" && chmod 600 "$HOME/.netrc"
        env:
          GITPAT: ${{ secrets.GITPAT }}
      - name: Vendor Go dependencies
        run: go mod vendor
        env:
          GOPRIVATE: %s
          GONOSUMDB: %s
      - name: Provision apx toolchain
        run: |
          cid="$(docker create ghcr.io/infobloxopen/apx-toolchain:v1)"
          trap 'docker rm -f "$cid" >/dev/null 2>&1 || true' EXIT
          dest="$HOME/.apx-toolchain/bin"
          mkdir -p "$dest"
          docker cp "$cid:/usr/local/bin/." "$dest/"
          echo "$dest" >> "$GITHUB_PATH"
      - name: Load apx publish script
        uses: actions/checkout@v4
        with:
          repository: infobloxopen/apx-action
          ref: v1
          path: _apx_action
      - name: Read canonical repo
        id: cfg
        run: |
          repo="$(yq -r '.canonical_repo' .apx-publish.yaml)"
          hp="${repo#http*://}"; hp="${hp%%.git}"; hp="${hp#github.com/}"
          echo "owner=${hp%%%%/*}" >> "$GITHUB_OUTPUT"
          echo "repo_name=${hp##*/}" >> "$GITHUB_OUTPUT"
      - name: Generate App Token
        id: app-token
        uses: actions/create-github-app-token@v3
        with:
          app-id: ${{ secrets.APX_SUBMIT_APP_ID }}
          private-key: ${{ secrets.APX_SUBMIT_APP_PRIVATE_KEY }}
          owner: ${{ steps.cfg.outputs.owner }}
          repositories: ${{ steps.cfg.outputs.repo_name }}
      - name: Clone canonical catalog
        run: git clone --quiet "https://x-access-token:${GH_TOKEN}@github.com/${{ steps.cfg.outputs.owner }}/${{ steps.cfg.outputs.repo_name }}.git" .canonical
        env:
          GH_TOKEN: ${{ steps.app-token.outputs.token }}
      - name: Check API catalog
        run: |
          git config --global user.name "apx-submit[bot]"
          git config --global user.email "apx-submit[bot]@users.noreply.github.com"
          chmod +x _apx_action/scripts/apx-publish.sh
          _apx_action/scripts/apx-publish.sh --mode check --canonical-dir .canonical --breaking-gate auto --verify-clients fail --supersede on
        env:
          GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}
`, schemaGlob, goPrivate, goPrivate)
}

// generateInlinePublishWorkflow produces a fully-inlined publish workflow for
// repos that need go mod vendor with private-repo auth before make fds.
func generateInlinePublishWorkflow(schemaGlob, defaultBranch, goPrivate string) string {
	return fmt.Sprintf(`name: API catalog — publish on change
on:
  push:
    branches: [%s]
    paths:
      - '%s'
      - '.apx-publish.yaml'
      - 'apx.yaml'
permissions:
  contents: read
  issues: write
jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Configure private Go module auth
        run: printf 'machine github.com login x-oauth-basic password %%s\n' "${GITPAT}" > "$HOME/.netrc" && chmod 600 "$HOME/.netrc"
        env:
          GITPAT: ${{ secrets.GITPAT }}
      - name: Vendor Go dependencies
        run: go mod vendor
        env:
          GOPRIVATE: %s
          GONOSUMDB: %s
      - name: Provision apx toolchain
        run: |
          cid="$(docker create ghcr.io/infobloxopen/apx-toolchain:v1)"
          trap 'docker rm -f "$cid" >/dev/null 2>&1 || true' EXIT
          dest="$HOME/.apx-toolchain/bin"
          mkdir -p "$dest"
          docker cp "$cid:/usr/local/bin/." "$dest/"
          echo "$dest" >> "$GITHUB_PATH"
      - name: Load apx publish script
        uses: actions/checkout@v4
        with:
          repository: infobloxopen/apx-action
          ref: v1
          path: _apx_action
      - name: Read canonical repo
        id: cfg
        run: |
          repo="$(yq -r '.canonical_repo' .apx-publish.yaml)"
          hp="${repo#http*://}"; hp="${hp%%.git}"; hp="${hp#github.com/}"
          echo "owner=${hp%%%%/*}" >> "$GITHUB_OUTPUT"
          echo "repo_name=${hp##*/}" >> "$GITHUB_OUTPUT"
      - name: Generate App Token
        id: app-token
        uses: actions/create-github-app-token@v3
        with:
          app-id: ${{ secrets.APX_SUBMIT_APP_ID }}
          private-key: ${{ secrets.APX_SUBMIT_APP_PRIVATE_KEY }}
          owner: ${{ steps.cfg.outputs.owner }}
          repositories: ${{ steps.cfg.outputs.repo_name }}
      - name: Clone canonical catalog
        run: git clone --quiet "https://x-access-token:${GH_TOKEN}@github.com/${{ steps.cfg.outputs.owner }}/${{ steps.cfg.outputs.repo_name }}.git" .canonical
        env:
          GH_TOKEN: ${{ steps.app-token.outputs.token }}
      - name: Publish to API catalog
        run: |
          git config --global user.name "apx-submit[bot]"
          git config --global user.email "apx-submit[bot]@users.noreply.github.com"
          git config --global "http.https://github.com/.extraheader" \
            "AUTHORIZATION: Basic $(printf 'x-access-token:%%s' "${GITHUB_TOKEN}" | base64 | tr -d '\n')"
          chmod +x _apx_action/scripts/apx-publish.sh
          _apx_action/scripts/apx-publish.sh --mode publish --canonical-dir .canonical --breaking-gate auto --verify-clients fail --supersede on
        env:
          GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}
`, defaultBranch, schemaGlob, goPrivate, goPrivate)
}

func generatePRWorkflow(schemaGlob string) string {
	return fmt.Sprintf(`name: API catalog — PR check
on:
  pull_request:
    paths:
      - '%s'
      - '.apx-publish.yaml'
      - 'apx.yaml'
      - 'Makefile'
permissions:
  contents: read
  pull-requests: write
jobs:
  check:
    uses: infobloxopen/apx-action/.github/workflows/apx-publish.yml@v1
    with:
      mode: check
      apx-action-ref: v1
      breaking-gate: auto
    secrets:
      apx-app-id:          ${{ secrets.APX_SUBMIT_APP_ID }}
      apx-app-private-key: ${{ secrets.APX_SUBMIT_APP_PRIVATE_KEY }}
`, schemaGlob)
}

func generatePublishWorkflow(schemaGlob, defaultBranch string) string {
	return fmt.Sprintf(`name: API catalog — publish on change
on:
  push:
    branches: [%s]
    paths:
      - '%s'
      - '.apx-publish.yaml'
      - 'apx.yaml'
permissions:
  contents: read
  issues: write
jobs:
  publish:
    uses: infobloxopen/apx-action/.github/workflows/apx-publish.yml@v1
    with:
      mode: publish
      apx-action-ref: v1
    secrets:
      apx-app-id:          ${{ secrets.APX_SUBMIT_APP_ID }}
      apx-app-private-key: ${{ secrets.APX_SUBMIT_APP_PRIVATE_KEY }}
`, defaultBranch, schemaGlob)
}
