package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// seamFuncs are the clock-injecting entry points on the rate limiter. They exist
// so the unit tests can drive refill from a fixed instant; production must reach
// the limiter through the no-argument wrappers instead.
var seamFuncs = map[string]struct{}{
	"allowAt":          {},
	"newRateLimiterAt": {},
}

// seamWrappers are the two functions allowed to call into the seam: they are the
// wrappers that supply time.Now(), so they are how production reaches it.
var seamWrappers = map[string]struct{}{
	"allow":          {},
	"newRateLimiter": {},
}

// TestClockSeamIsTestOnly enforces what the doc comments on allowAt and
// newRateLimiterAt promise. Both are package-scoped, so nothing stops another
// file in package server from handing the limiter an instant — and an instant a
// caller chooses is a throttle a caller can loosen. This test is the constraint
// the compiler cannot express: outside _test.go files, the seam is reachable
// only from its own wrappers.
//
// It parses rather than greps so that a mention in a comment or a string does
// not fail it, and so a new production file is covered the day it is added.
func TestClockSeamIsTestOnly(t *testing.T) {
	t.Parallel()

	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("failed to list package sources: %v", err)
	}

	var checked int

	for _, path := range sources {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		checked++

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", path, err)
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if _, sanctioned := seamWrappers[fn.Name.Name]; sanctioned {
				continue
			}

			// Bodies only. A declaration mentions its own name, and the seam
			// declaring itself is not a call into it.
			if fn.Body == nil {
				continue
			}

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				ident, ok := n.(*ast.Ident)
				if !ok {
					return true
				}

				if _, isSeam := seamFuncs[ident.Name]; isSeam {
					t.Errorf("%s: %s calls the clock seam %s; production must go through the "+
						"no-argument wrapper so the throttle stays the configured one",
						path, fn.Name.Name, ident.Name)
				}

				return true
			})
		}
	}

	// A glob that quietly matched nothing would make this test vacuous.
	if checked == 0 {
		t.Fatal("found no non-test sources to check; the seam is unguarded")
	}
}
