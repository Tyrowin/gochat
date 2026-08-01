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

// TestClockSeamIsTestOnly enforces what the doc comments on allowAt and
// newRateLimiterAt promise. Both are package-scoped, so nothing stops another
// file in package server from handing the limiter an instant — and an instant a
// caller chooses is a throttle a caller can loosen. This test is the constraint
// the compiler cannot express: outside _test.go files, the seam is named only by
// its own two wrappers.
//
// It parses rather than greps, so a mention in a comment or a string does not
// fail it and a new production file is covered the day it is added. It walks
// whole files rather than function bodies, so a package-level var calling the
// seam is caught as readily as a call inside a function.
//
// Scope, stated plainly: this guards the seam functions. It is not a guarantee
// that no production code can arrange a stale baseline by other means — the
// rateLimiter struct and its fields are package-scoped too, so a composite
// literal or a direct write to last would sidestep this check. That door is
// older than the seam and unchanged by it; closing it is a separate job.
func TestClockSeamIsTestOnly(t *testing.T) {
	t.Parallel()

	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("failed to list package sources: %v", err)
	}

	fset := token.NewFileSet()

	var checked int

	for _, path := range sources {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		checked++

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", path, err)
		}

		// Nodes the walk below must not descend into or report: the bodies of
		// the two sanctioned wrappers, which exist precisely to call the seam,
		// and the seam declarations' own names, since declaring a function is
		// not calling it.
		exempt := map[ast.Node]bool{}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if isSeamWrapper(fn) && fn.Body != nil {
				exempt[fn.Body] = true
			}

			if _, isSeam := seamFuncs[fn.Name.Name]; isSeam {
				exempt[fn.Name] = true
			}
		}

		ast.Inspect(file, func(n ast.Node) bool {
			if n == nil || exempt[n] {
				return false
			}

			ident, ok := n.(*ast.Ident)
			if !ok {
				return true
			}

			if _, isSeam := seamFuncs[ident.Name]; isSeam {
				t.Errorf("%s: names the clock seam %s; production must go through the "+
					"no-argument wrapper so the throttle stays the configured one",
					fset.Position(ident.Pos()), ident.Name)
			}

			return true
		})
	}

	// A glob that quietly matched nothing would make this test vacuous.
	if checked == 0 {
		t.Fatal("found no non-test sources to check; the seam is unguarded")
	}
}

// isSeamWrapper reports whether fn is one of the two functions allowed to name
// the seam: the rateLimiter.allow method, and the package-level newRateLimiter.
// The receiver is checked rather than the name alone, so an unrelated method
// that happens to be called allow does not inherit the exemption.
func isSeamWrapper(fn *ast.FuncDecl) bool {
	switch fn.Name.Name {
	case "newRateLimiter":
		return fn.Recv == nil
	case "allow":
		return receiverTypeName(fn) == "rateLimiter"
	default:
		return false
	}
}

// receiverTypeName returns the name of fn's receiver type, without any pointer,
// or "" when fn is not a method.
func receiverTypeName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}

	expr := fn.Recv.List[0].Type
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}

	ident, ok := expr.(*ast.Ident)
	if !ok {
		return ""
	}

	return ident.Name
}
