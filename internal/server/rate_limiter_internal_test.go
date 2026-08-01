// Package server: this file guards the rate limiter's clock seam. It is the one
// check in the package about the package's own shape rather than its behaviour,
// which is why it holds no limiter tests itself — refill, the cap at capacity, a
// frozen clock and a rewound one are pinned in hub_internal_test.go alongside
// the other unexported hot paths.
package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// seamFuncs are the clock-injecting entry points on the rate limiter, listed
// here for detection.
//
// Detection matches the identifier alone, deliberately unlike [isSeamWrapper]
// below, which checks the receiver too. Each is loose in its safe direction: a
// name matched too widely here costs a false failure, which a reader sees and
// can rule out in a moment, whereas a wrapper exempted too widely would wave a
// real caller through in silence.
var seamFuncs = map[string]struct{}{
	"allowAt":          {},
	"newRateLimiterAt": {},
}

// TestClockSeamIsTestOnly enforces what the doc comment on [rateLimiter.allowAt]
// promises: outside _test.go files, the seam is named only by its own two
// wrappers. Both seam functions are package-scoped, so nothing else stops a file
// in package server from handing the limiter an instant — and an instant a
// caller chooses is a throttle a caller can loosen. Why the seam is shaped this
// way is in docs/architecture/overview.md; this is the check that keeps it that
// way.
//
// It parses rather than greps, so a mention in a comment or a string does not
// fail it, and a new production file is covered the day it is added. It walks
// every top-level declaration rather than function bodies alone, so a
// package-level var calling the seam is caught as readily as a call inside a
// function.
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

		for _, decl := range file.Decls {
			checkDeclForSeam(t, fset, decl)
		}
	}

	// A glob that quietly matched nothing would make this test vacuous.
	if checked == 0 {
		t.Fatal("found no non-test sources to check; the seam is unguarded")
	}
}

// checkDeclForSeam reports every mention of the seam within one top-level
// declaration, naming the declaration it was found in so a failure says where to
// look without opening the file.
//
// Two kinds of declaration are handled specially: the wrappers exist precisely
// to call the seam, so they are skipped whole, and the seam's own declarations
// are walked everywhere except their names, since declaring a function is not
// calling it.
func checkDeclForSeam(t *testing.T, fset *token.FileSet, decl ast.Decl) {
	t.Helper()

	fn, isFunc := decl.(*ast.FuncDecl)
	if !isFunc {
		checkNodeForSeam(t, fset, "package-level declaration", decl)
		return
	}

	if isSeamWrapper(fn) {
		return
	}

	where := funcLabel(fn)

	if _, isSeam := seamFuncs[fn.Name.Name]; isSeam {
		checkNodeForSeam(t, fset, where, fn.Type)
		if fn.Body != nil {
			checkNodeForSeam(t, fset, where, fn.Body)
		}
		return
	}

	checkNodeForSeam(t, fset, where, decl)
}

// checkNodeForSeam fails the test once per identifier under node that names the
// seam.
func checkNodeForSeam(t *testing.T, fset *token.FileSet, where string, node ast.Node) {
	t.Helper()

	ast.Inspect(node, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		if _, isSeam := seamFuncs[ident.Name]; isSeam {
			t.Errorf("%s: %s names the clock seam %s; production must go through the "+
				"no-argument wrapper so the throttle stays the configured one",
				fset.Position(ident.Pos()), where, ident.Name)
		}

		return true
	})
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

// funcLabel names fn the way a reader would go looking for it: "func allow" for
// a plain function, "func (rateLimiter) allow" for a method.
func funcLabel(fn *ast.FuncDecl) string {
	if recv := receiverTypeName(fn); recv != "" {
		return "func (" + recv + ") " + fn.Name.Name
	}

	return "func " + fn.Name.Name
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
