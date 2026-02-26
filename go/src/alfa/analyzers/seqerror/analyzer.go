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

		if id, ok := rangeStmt.Value.(*ast.Ident); ok && id.Name == "_" {
			if !hasErrCheckedComment(pass, rangeStmt) {
				pass.ReportRangef(id, "error from iter.Seq2 range is discarded; must be checked (or add //seq:err-checked)")
			}
		}
	})

	return nil, nil
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
