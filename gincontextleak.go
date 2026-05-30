package gincontextleak

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name: "gincontextleak",
	Doc: `reports calls that pass *gin.Context to a context.Context parameter.

gin.Context is not safe to use concurrently. Passing it to functions
that expect context.Context (and may retain it for background work or
goroutines) can cause data races because the underlying *gin.Context
is reused by the Gin engine for subsequent requests.

The suggested fix replaces the argument with <arg>.Request.Context(),
which returns the standard library context.Context associated with the
request and is safe to use.`,
	URL:      "https://github.com/haruyama480/gincontextleak",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	ins.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)

		typ := pass.TypesInfo.TypeOf(call.Fun)
		if typ == nil {
			return
		}
		sig, ok := typ.Underlying().(*types.Signature)
		if !ok {
			return
		}

		for i, arg := range call.Args {
			argTyp := pass.TypesInfo.TypeOf(arg)
			if argTyp == nil {
				continue
			}
			if !isGinContextPtr(argTyp) {
				continue
			}
			paramTyp := paramTypeForArg(sig, i)
			if paramTyp == nil {
				continue
			}
			if !isContextType(paramTyp) {
				continue
			}

			// Report the problematic argument.
			pass.Report(analysis.Diagnostic{
				Pos:     arg.Pos(),
				End:     arg.End(),
				Message: "do not pass *gin.Context where context.Context is expected: *gin.Context is not goroutine-safe and may cause data races (see gin-gonic/gin#4117); use c.Request.Context() instead",
				SuggestedFixes: []analysis.SuggestedFix{{
					Message: "Replace with .Request.Context() to obtain a goroutine-safe context.Context",
					TextEdits: []analysis.TextEdit{
						makeRequestContextEdit(pass, arg),
					},
				}},
			})
		}
	})

	return nil, nil
}

func isGinContextPtr(t types.Type) bool {
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}
	if named.Obj().Name() != "Context" {
		return false
	}
	pkg := named.Obj().Pkg()
	return pkg != nil && pkg.Path() == "github.com/gin-gonic/gin"
}

func isContextType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	if named.Obj().Name() != "Context" {
		return false
	}
	pkg := named.Obj().Pkg()
	return pkg != nil && pkg.Path() == "context"
}

func paramTypeForArg(sig *types.Signature, idx int) types.Type {
	if sig == nil {
		return nil
	}
	params := sig.Params()
	n := params.Len()
	if n == 0 {
		return nil
	}
	if sig.Variadic() && idx >= n-1 {
		last := params.At(n - 1).Type()
		if s, ok := last.(*types.Slice); ok {
			return s.Elem()
		}
		return last
	}
	if idx < n {
		return params.At(idx).Type()
	}
	return nil
}

func makeRequestContextEdit(pass *analysis.Pass, arg ast.Expr) analysis.TextEdit {
	// Build the replacement expression: <arg>.Request.Context()
	reqSel := &ast.SelectorExpr{
		X:   arg,
		Sel: ast.NewIdent("Request"),
	}
	ctxSel := &ast.SelectorExpr{
		X:   reqSel,
		Sel: ast.NewIdent("Context"),
	}
	call := &ast.CallExpr{
		Fun: ctxSel,
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, pass.Fset, call); err != nil {
		// format.Node should not fail for this simple selector/call tree.
		// Return a placeholder so the edit is still valid (user can adjust).
		return analysis.TextEdit{
			Pos:     arg.Pos(),
			End:     arg.End(),
			NewText: []byte("/* gincontextleak: fix failed */"),
		}
	}

	return analysis.TextEdit{
		Pos:     arg.Pos(),
		End:     arg.End(),
		NewText: buf.Bytes(),
	}
}
