package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const apiConfigWithProvider = `export function provideApiConfiguration(rootUrl: string) {
  var config = new ApiConfiguration();
  config.rootUrl = rootUrl;
  return { provide: ApiConfiguration, useValue: config };
}
export class ApiConfiguration { rootUrl: string = ''; }
`

func TestEnsureProviderExported_AddsExport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api-configuration.ts", apiConfigWithProvider)
	writeFile(t, dir, "index.ts", "export { ApiConfiguration } from './api-configuration';\n")

	if err := ensureProviderExported(dir); err != nil {
		t.Fatalf("ensureProviderExported: %v", err)
	}
	got := readFile(t, dir, "index.ts")
	if !strings.Contains(got, "export { provideApiConfiguration } from './api-configuration';") {
		t.Fatalf("barrel missing provideApiConfiguration re-export:\n%s", got)
	}
}

func TestEnsureProviderExported_Idempotent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api-configuration.ts", apiConfigWithProvider)
	writeFile(t, dir, "index.ts",
		"export { ApiConfiguration } from './api-configuration';\n"+
			"export { provideApiConfiguration } from './api-configuration';\n")

	if err := ensureProviderExported(dir); err != nil {
		t.Fatalf("ensureProviderExported: %v", err)
	}
	got := readFile(t, dir, "index.ts")
	if n := strings.Count(got, "export { provideApiConfiguration }"); n != 1 {
		t.Fatalf("expected exactly one provideApiConfiguration export, got %d:\n%s", n, got)
	}
}

func TestEnsureProviderExported_NoHelper_NoOp(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api-configuration.ts", "export class ApiConfiguration { rootUrl = ''; }\n")
	orig := "export { ApiConfiguration } from './api-configuration';\n"
	writeFile(t, dir, "index.ts", orig)

	if err := ensureProviderExported(dir); err != nil {
		t.Fatalf("ensureProviderExported: %v", err)
	}
	if got := readFile(t, dir, "index.ts"); got != orig {
		t.Fatalf("expected barrel unchanged when helper absent, got:\n%s", got)
	}
}

func TestEnsureProviderExported_NoBarrel_NoError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api-configuration.ts", apiConfigWithProvider)
	if err := ensureProviderExported(dir); err != nil {
		t.Fatalf("expected no error when barrel absent, got: %v", err)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(b)
}
