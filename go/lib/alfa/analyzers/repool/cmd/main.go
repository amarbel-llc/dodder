package main

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/analyzers/repool"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(repool.Analyzer)
}
