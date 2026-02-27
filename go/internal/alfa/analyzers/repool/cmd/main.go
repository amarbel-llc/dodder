package main

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/analyzers/repool"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(repool.Analyzer)
}
