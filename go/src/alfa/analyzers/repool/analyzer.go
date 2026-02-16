// Package repool defines an Analyzer that checks for failure to call
// a FuncRepool function returned by pool.GetWithRepool.
//
// # Analyzer repool
//
// repool: check FuncRepool returned by GetWithRepool is called
//
// The repool function returned by GetWithRepool must be called exactly
// once when the caller is done with the pooled object, or the object
// will leak. Discarding the repool function with a blank identifier
// is reported unless suppressed with a //repool:owned comment.
package repool

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/cfg"
)

const funcRepoolTypeName = "FuncRepool"
const funcRepoolPkgSuffix = "interfaces"

var Analyzer = &analysis.Analyzer{
	Name: "repool",
	Doc:  "check FuncRepool returned by GetWithRepool is called",
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
		ctrlflow.Analyzer,
	},
}

func run(pass *analysis.Pass) (any, error) {
	if !importsFuncRepool(pass.Pkg) {
		return nil, nil
	}

	ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	cfgs := pass.ResultOf[ctrlflow.Analyzer].(*ctrlflow.CFGs)

	nodeTypes := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	ins.Preorder(nodeTypes, func(n ast.Node) {
		runFunc(pass, cfgs, n)
	})

	return nil, nil
}

func runFunc(pass *analysis.Pass, cfgs *ctrlflow.CFGs, node ast.Node) {
	var body *ast.BlockStmt

	switch n := node.(type) {
	case *ast.FuncDecl:
		body = n.Body
	case *ast.FuncLit:
		body = n.Body
	}

	if body == nil {
		return
	}

	ast.Inspect(body, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.FuncLit:
			return false // don't descend into nested function literals

		case *ast.AssignStmt:
			checkAssign(pass, cfgs, node, stmt)

		case *ast.ValueSpec:
			checkValueSpec(pass, cfgs, node, stmt)
		}

		return true
	})
}

func checkAssign(pass *analysis.Pass, cfgs *ctrlflow.CFGs, funcNode ast.Node, stmt *ast.AssignStmt) {
	if len(stmt.Rhs) != 1 {
		return
	}

	call, ok := stmt.Rhs[0].(*ast.CallExpr)
	if !ok {
		return
	}

	idx := repoolResultIndex(pass, call)
	if idx < 0 || idx >= len(stmt.Lhs) {
		return
	}

	id, ok := stmt.Lhs[idx].(*ast.Ident)
	if !ok {
		return
	}

	if id.Name == "_" {
		if hasRepoolOwnedComment(pass, stmt) {
			return
		}

		pass.ReportRangef(id,
			"the repool function returned by %s should be called, not discarded, to avoid a pool leak",
			callName(call))
		return
	}

	v, ok := pass.TypesInfo.Uses[id].(*types.Var)
	if !ok {
		v, ok = pass.TypesInfo.Defs[id].(*types.Var)
		if !ok {
			return
		}
	}

	checkVarUsedOnAllPaths(pass, cfgs, funcNode, stmt, v)
}

func checkValueSpec(pass *analysis.Pass, cfgs *ctrlflow.CFGs, funcNode ast.Node, spec *ast.ValueSpec) {
	if len(spec.Values) != 1 {
		return
	}

	call, ok := spec.Values[0].(*ast.CallExpr)
	if !ok {
		return
	}

	idx := repoolResultIndex(pass, call)
	if idx < 0 || idx >= len(spec.Names) {
		return
	}

	id := spec.Names[idx]

	if id.Name == "_" {
		if hasRepoolOwnedComment(pass, spec) {
			return
		}

		pass.ReportRangef(id,
			"the repool function returned by %s should be called, not discarded, to avoid a pool leak",
			callName(call))
		return
	}

	v, ok := pass.TypesInfo.Defs[id].(*types.Var)
	if !ok {
		return
	}

	checkVarUsedOnAllPaths(pass, cfgs, funcNode, spec, v)
}

func checkVarUsedOnAllPaths(
	pass *analysis.Pass,
	cfgs *ctrlflow.CFGs,
	funcNode ast.Node,
	defStmt ast.Node,
	v *types.Var,
) {
	var g *cfg.CFG

	switch fn := funcNode.(type) {
	case *ast.FuncDecl:
		g = cfgs.FuncDecl(fn)
	case *ast.FuncLit:
		g = cfgs.FuncLit(fn)
	}

	if g == nil {
		return
	}

	// Find the defining block and remaining nodes after the definition.
	var defblock *cfg.Block
	var rest []ast.Node

outer:
	for _, b := range g.Blocks {
		for i, n := range b.Nodes {
			if n == defStmt {
				defblock = b
				rest = b.Nodes[i+1:]
				break outer
			}
		}
	}

	if defblock == nil {
		return
	}

	// Is v used in the rest of its defining block?
	if usesVar(pass, v, rest) {
		return
	}

	// Does the defining block have no successors (implicit return)?
	if len(defblock.Succs) == 0 {
		pass.ReportRangef(defStmt,
			"the repool function is not called on all paths (possible pool leak)")
		return
	}

	// Search depth-first for a path to return without using v.
	seen := make(map[*cfg.Block]bool)
	if ret := searchUnused(pass, v, defblock.Succs, seen); ret != nil {
		pass.ReportRangef(defStmt,
			"the repool function is not called on all paths (possible pool leak)")
	}
}

func searchUnused(pass *analysis.Pass, v *types.Var, blocks []*cfg.Block, seen map[*cfg.Block]bool) *cfg.Block {
	for _, b := range blocks {
		if seen[b] {
			continue
		}
		seen[b] = true

		if blockUsesVar(pass, v, b) {
			continue
		}

		// Block doesn't use v. If it's a terminal block (no successors),
		// we found a path to exit without calling the repool function.
		if len(b.Succs) == 0 {
			return b
		}

		if found := searchUnused(pass, v, b.Succs, seen); found != nil {
			return found
		}
	}

	return nil
}

func usesVar(pass *analysis.Pass, v *types.Var, stmts []ast.Node) bool {
	for _, stmt := range stmts {
		found := false
		ast.Inspect(stmt, func(n ast.Node) bool {
			if found {
				return false
			}
			if id, ok := n.(*ast.Ident); ok {
				if pass.TypesInfo.Uses[id] == v {
					found = true
				}
			}
			return !found
		})
		if found {
			return true
		}
	}
	return false
}

func blockUsesVar(pass *analysis.Pass, v *types.Var, b *cfg.Block) bool {
	return usesVar(pass, v, b.Nodes)
}

// repoolResultIndex returns the index within the result tuple that is
// a FuncRepool type, or -1 if the call doesn't return one.
func repoolResultIndex(pass *analysis.Pass, call *ast.CallExpr) int {
	t := pass.TypesInfo.TypeOf(call)
	if t == nil {
		return -1
	}

	tuple, ok := t.(*types.Tuple)
	if !ok {
		// Single return value.
		if isFuncRepoolType(t) {
			return 0
		}
		return -1
	}

	for i := range tuple.Len() {
		if isFuncRepoolType(tuple.At(i).Type()) {
			return i
		}
	}

	return -1
}

func isFuncRepoolType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj.Name() != funcRepoolTypeName {
		return false
	}

	pkg := obj.Pkg()
	if pkg == nil {
		return false
	}

	return strings.HasSuffix(pkg.Path(), funcRepoolPkgSuffix)
}

func importsFuncRepool(pkg *types.Package) bool {
	for _, imp := range pkg.Imports() {
		if strings.HasSuffix(imp.Path(), funcRepoolPkgSuffix) {
			return true
		}
	}
	return false
}

func hasRepoolOwnedComment(pass *analysis.Pass, node ast.Node) bool {
	pos := pass.Fset.Position(node.Pos())

	for _, cg := range pass.Files {
		for _, c := range cg.Comments {
			for _, comment := range c.List {
				cpos := pass.Fset.Position(comment.Pos())
				if cpos.Line == pos.Line && strings.Contains(comment.Text, "//repool:owned") {
					return true
				}
			}
		}
	}

	return false
}

func callName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fn.Sel.Name
	case *ast.Ident:
		return fn.Name
	}

	return pass_fset_position(call.Pos())
}

func pass_fset_position(_ token.Pos) string {
	return "<unknown>"
}
