package main

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/analyzers/seqerror"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(seqerror.Analyzer)
}
