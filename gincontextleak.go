// Package gincontextleak provides a go/analysis analyzer that detects
// unsafe passing of *gin.Context to functions expecting context.Context.
//
// gin.Context is not goroutine-safe. Passing it where context.Context is
// expected can cause data races.
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

// Analyzer reports calls that pass *gin.Context to a context.Context parameter.
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

	// Per-package caches for type predicate results.
	// Within one analysis pass, the same semantic type reuses the same
	// types.Type value, so map lookup by identity is both correct and fast.
	ginCtxCache := make(map[types.Type]bool)
	ctxCache := make(map[types.Type]bool)

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
		if !signatureHasAnyContextParam(sig) {
			return
		}

		for i, arg := range call.Args {
			argTyp := pass.TypesInfo.TypeOf(arg)
			if argTyp == nil {
				continue
			}
			if !isGinContextPtrCached(argTyp, ginCtxCache) {
				continue
			}
			paramTyp := paramTypeForArg(sig, i)
			if paramTyp == nil {
				continue
			}
			if !isContextTypeCached(paramTyp, ctxCache) {
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
	t = unwrapAlias(t)
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	elem := unwrapAlias(ptr.Elem())
	named, ok := elem.(*types.Named)
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
	t = unwrapAlias(t)
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

// unwrapAlias repeatedly unwraps *types.Alias until a non-alias type is reached.
// This makes the predicates robust against type aliases (type Foo = *gin.Context),
// which became common after Go 1.23's improved alias support in go/types.
func unwrapAlias(t types.Type) types.Type {
	for {
		alias, ok := t.(*types.Alias)
		if !ok {
			return t
		}
		t = alias.Underlying()
	}
}

// Cached versions of the type predicates. The maps are local to a single
// analysis.Pass (one package), so using the types.Type value as key is safe.
func isGinContextPtrCached(t types.Type, cache map[types.Type]bool) bool {
	if res, ok := cache[t]; ok {
		return res
	}
	res := isGinContextPtr(t)
	cache[t] = res
	return res
}

func isContextTypeCached(t types.Type, cache map[types.Type]bool) bool {
	if res, ok := cache[t]; ok {
		return res
	}
	res := isContextType(t)
	cache[t] = res
	return res
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
	// Fast path for the common case: the argument is a simple identifier
	// such as `c` or `ctx`. This is by far the most frequent pattern in
	// real Gin handlers and completely avoids AST construction + format.Node.
	if ident, ok := arg.(*ast.Ident); ok {
		return analysis.TextEdit{
			Pos:     arg.Pos(),
			End:     arg.End(),
			NewText: []byte(ident.Name + ".Request.Context()"),
		}
	}

	// Fallback for complex expressions (selector exprs, calls, parenthesized
	// expressions, index expressions, etc.). We still need to produce valid
	// Go source, so we build a small AST and let go/format handle it.
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

// signatureHasAnyContextParam reports whether sig has at least one parameter
// whose type is context.Context (including the element type of a variadic
// ...context.Context parameter). It is used to skip call expressions early
// when the callee cannot possibly accept a context.Context argument.
func signatureHasAnyContextParam(sig *types.Signature) bool {
	if sig == nil {
		return false
	}
	params := sig.Params()
	n := params.Len()
	for i := 0; i < n; i++ {
		if isContextType(paramTypeForArg(sig, i)) {
			return true
		}
	}
	return false
}
