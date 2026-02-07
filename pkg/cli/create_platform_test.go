package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"gotest.tools/v3/assert"
)

// Regression test for ENGPROV-118: --restore flag must not be silently ignored
// when using the platform driver. This test verifies that CreatePlatform
// references options.Restore and calls the Restore function.
func TestCreatePlatform_RestoreNotIgnored_ENGPROV118(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "create_platform.go", nil, 0)
	assert.NilError(t, err)

	// Find the CreatePlatform function
	var createPlatformFunc *ast.FuncDecl
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "CreatePlatform" {
			createPlatformFunc = fn
			break
		}
	}
	assert.Assert(t, createPlatformFunc != nil, "CreatePlatform function not found")

	// Verify that CreatePlatform references options.Restore
	foundRestoreCheck := false
	// Verify that CreatePlatform calls the Restore function
	foundRestoreCall := false

	ast.Inspect(createPlatformFunc, func(n ast.Node) bool {
		// Check for options.Restore selector
		sel, ok := n.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == "Restore" {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "options" {
				foundRestoreCheck = true
			}
		}

		// Check for a call to Restore()
		call, ok := n.(*ast.CallExpr)
		if ok {
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "Restore" {
				foundRestoreCall = true
			}
		}

		return true
	})

	assert.Assert(t, foundRestoreCheck,
		"CreatePlatform must check options.Restore (ENGPROV-118: --restore flag was silently ignored with platform driver)")
	assert.Assert(t, foundRestoreCall,
		"CreatePlatform must call Restore() when options.Restore is set (ENGPROV-118: --restore flag was silently ignored with platform driver)")
}
