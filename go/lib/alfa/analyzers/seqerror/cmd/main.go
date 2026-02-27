package main

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/analyzers/seqerror"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(seqerror.Analyzer)
}
