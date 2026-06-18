package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequiredRolesFailsOnMissingMutatingAnnotation(t *testing.T) {
	output, err := runRequiredRoles(t, `
openapi: 3.0.3
info:
  title: Test
  version: "1.0"
security:
  - BearerAuth: []
paths:
  /teams:
    post:
      operationId: createTeam
      security:
        - BearerAuth: []
      responses:
        '204':
          description: OK
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
`)
	if err == nil {
		t.Fatalf("expected generator failure, got success")
	}
	if !strings.Contains(output, "POST /teams: missing x-required-role") {
		t.Fatalf("expected missing annotation error, got %s", output)
	}
}

func TestRequiredRolesAcceptsAuthenticatedWithoutRoleGate(t *testing.T) {
	output, err := runRequiredRoles(t, `
openapi: 3.0.3
info:
  title: Test
  version: "1.0"
security:
  - BearerAuth: []
paths:
  /teams:
    post:
      operationId: createTeam
      x-required-role: authenticated
      security:
        - BearerAuth: []
      responses:
        '204':
          description: OK
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
`)
	if err != nil {
		t.Fatalf("expected generator success, got %v: %s", err, output)
	}
	if strings.Contains(output, "POST /teams") {
		t.Fatalf("authenticated operation must not be emitted as a role gate: %s", output)
	}
}

func runRequiredRoles(t *testing.T, spec string) (string, error) {
	t.Helper()

	dir := t.TempDir()
	input := filepath.Join(dir, "openapi.yaml")
	output := filepath.Join(dir, "roles.gen.go")
	if err := os.WriteFile(input, []byte(strings.TrimSpace(spec)+"\n"), 0o644); err != nil {
		t.Fatalf("write input spec: %v", err)
	}

	cmd := exec.CommandContext(
		t.Context(),
		"go",
		"run",
		"./openapi-required-roles.go",
		"-input",
		input,
		"-output",
		output,
		"-package",
		"httpserver",
	)
	combined, err := cmd.CombinedOutput()
	if err != nil {
		return string(combined), err
	}

	generated, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read generated roles: %v", err)
	}
	return string(generated), nil
}
