// Package seqerror defines an Analyzer that checks the error value
// from iter.Seq2[T, error] range statements is not discarded or
// left unchecked.
//
// # Analyzer seqerror
//
// seqerror: check error from iter.Seq2 range is not discarded
//
// When ranging over an iter.Seq2[T, error], the error must be checked
// (typically via an if err != nil block with a scope-exiting statement)
// or propagated (passed to a yield function or other call). Discarding
// the error with a blank identifier or omitting it entirely is reported
// unless suppressed with a //seq:err-checked comment.
package seqerror

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "seqerror",
	Doc:      "check error from iter.Seq2 range is not discarded",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (any, error) {
	ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeTypes := []ast.Node{(*ast.RangeStmt)(nil)}

	ins.Preorder(nodeTypes, func(n ast.Node) {
		rangeStmt := n.(*ast.RangeStmt)

		typ := pass.TypesInfo.TypeOf(rangeStmt.X)
		if typ == nil {
			return
		}

		if !isSeq2Error(typ) {
			return
		}

		if rangeStmt.Value == nil {
			if !hasErrCheckedComment(pass, rangeStmt) {
				pass.ReportRangef(rangeStmt, "error from iter.Seq2 range is discarded; must be checked (or add //seq:err-checked)")
			}
			return
		}

		id, ok := rangeStmt.Value.(*ast.Ident)
		if !ok {
			return
		}

		if id.Name == "_" {
			if !hasErrCheckedComment(pass, rangeStmt) {
				pass.ReportRangef(id, "error from iter.Seq2 range is discarded; must be checked (or add //seq:err-checked)")
			}
			return
		}

		if hasErrCheckedComment(pass, rangeStmt) {
			return
		}

		v := resolveErrorVar(id, pass.TypesInfo)
		if v == nil {
			return
		}

		if bodyHasCallPassingVar(rangeStmt.Body.List, v, pass.TypesInfo) {
			return
		}

		checkNamedErrorUsage(pass, rangeStmt, id, v)
	})

	return nil, nil
}

func checkNamedErrorUsage(
	pass *analysis.Pass,
	rangeStmt *ast.RangeStmt,
	id *ast.Ident,
	v *types.Var,
) {
	for _, stmt := range rangeStmt.Body.List {
		ifStmt, ok := stmt.(*ast.IfStmt)
		if !ok {
			continue
		}

		if !exprReferencesVar(ifStmt.Cond, v, pass.TypesInfo) {
			continue
		}

		if !ifBodyHasQualifyingUsage(ifStmt.Body, v, pass.TypesInfo) {
			pass.ReportRangef(
				id,
				"error variable %q is checked but not handled; if-body must return, break, continue, or propagate the error",
				id.Name,
			)
		}

		return
	}

	pass.ReportRangef(
		id,
		"error variable %q from iter.Seq2 range is never checked or propagated",
		id.Name,
	)
}

func isSeq2Error(t types.Type) bool {
	sig, ok := t.Underlying().(*types.Signature)
	if !ok {
		return false
	}

	if sig.Params().Len() != 1 {
		return false
	}

	innerSig, ok := sig.Params().At(0).Type().Underlying().(*types.Signature)
	if !ok {
		return false
	}

	if innerSig.Params().Len() != 2 || innerSig.Results().Len() != 1 {
		return false
	}

	if !types.Identical(innerSig.Results().At(0).Type(), types.Typ[types.Bool]) {
		return false
	}

	errorType := types.Universe.Lookup("error").Type()
	return types.Identical(innerSig.Params().At(1).Type(), errorType)
}

func hasErrCheckedComment(pass *analysis.Pass, node ast.Node) bool {
	pos := pass.Fset.Position(node.Pos())

	for _, f := range pass.Files {
		for _, cg := range f.Comments {
			for _, comment := range cg.List {
				cpos := pass.Fset.Position(comment.Pos())
				if cpos.Line == pos.Line && strings.Contains(comment.Text, "//seq:err-checked") {
					return true
				}
			}
		}
	}

	return false
}

func resolveErrorVar(id *ast.Ident, info *types.Info) *types.Var {
	if obj, ok := info.Defs[id]; ok && obj != nil {
		if v, ok := obj.(*types.Var); ok {
			return v
		}
	}

	if obj, ok := info.Uses[id]; ok {
		if v, ok := obj.(*types.Var); ok {
			return v
		}
	}

	return nil
}

func exprReferencesVar(expr ast.Expr, v *types.Var, info *types.Info) bool {
	found := false

	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}

		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		if info.Uses[ident] == v {
			found = true
			return false
		}

		return true
	})

	return found
}

func bodyHasCallPassingVar(stmts []ast.Stmt, v *types.Var, info *types.Info) bool {
	found := false

	for _, stmt := range stmts {
		ast.Inspect(stmt, func(n ast.Node) bool {
			if found {
				return false
			}

			if _, ok := n.(*ast.FuncLit); ok {
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			for _, arg := range call.Args {
				if exprReferencesVar(arg, v, info) {
					found = true
					return false
				}
			}

			return true
		})

		if found {
			return true
		}
	}

	return false
}

func ifBodyHasQualifyingUsage(body *ast.BlockStmt, v *types.Var, info *types.Info) bool {
	for _, stmt := range body.List {
		switch s := stmt.(type) {
		case *ast.ReturnStmt:
			return true

		case *ast.BranchStmt:
			return true

		case *ast.ExprStmt:
			call, ok := s.X.(*ast.CallExpr)
			if !ok {
				continue
			}

			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
				return true
			}

			for _, arg := range call.Args {
				if exprReferencesVar(arg, v, info) {
					return true
				}
			}
		}
	}

	return false
}
