package main

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var routeRegistrationRE = regexp.MustCompile(`api\.HandleFunc\("([^"]*)"[^\n]*\.Methods\("([A-Z]+)"\)`)

func canonicalRoute(method, path string) string {
	if path == "/api/v1/" {
		path = "/api/v1"
	}
	return strings.ToUpper(method) + " " + path
}

func registeredOperations(t *testing.T) map[string]struct{} {
	t.Helper()
	data, err := os.ReadFile("routes.go")
	if err != nil {
		t.Fatal(err)
	}
	ops := map[string]struct{}{}
	for _, match := range routeRegistrationRE.FindAllStringSubmatch(string(data), -1) {
		path := "/api/v1" + match[1]
		ops[canonicalRoute(match[2], path)] = struct{}{}
	}
	// Health is intentionally outside the authenticated /api/v1 subrouter.
	ops["GET /health"] = struct{}{}
	return ops
}

func openAPIOperations(t *testing.T) map[string]struct{} {
	t.Helper()
	path := filepath.Join("..", "..", "OPENAPI.yaml")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	ops := map[string]struct{}{}
	var currentPath string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "  /") && strings.HasSuffix(strings.TrimSpace(line), ":") {
			currentPath = strings.TrimSuffix(strings.TrimSpace(line), ":")
			continue
		}
		if currentPath == "" || !strings.HasPrefix(line, "    ") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		method := strings.TrimSuffix(trimmed, ":")
		switch method {
		case "get", "post", "put", "patch", "delete", "head", "options":
			ops[canonicalRoute(strings.ToUpper(method), currentPath)] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return ops
}

func diffOperations(a, b map[string]struct{}) []string {
	var out []string
	for op := range a {
		if _, ok := b[op]; !ok {
			out = append(out, op)
		}
	}
	sort.Strings(out)
	return out
}

func TestOpenAPIExactlyMatchesProductionRouteRegistry(t *testing.T) {
	registered := registeredOperations(t)
	documented := openAPIOperations(t)
	if missing := diffOperations(registered, documented); len(missing) > 0 {
		t.Fatalf("registered routes missing from OpenAPI: %v", missing)
	}
	if stale := diffOperations(documented, registered); len(stale) > 0 {
		t.Fatalf("OpenAPI operations without registered route: %v", stale)
	}
	if got := len(documented); got != 72 {
		t.Fatalf("expected 72 documented operations, got %d", got)
	}
}
