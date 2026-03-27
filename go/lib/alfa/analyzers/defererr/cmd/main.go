package main

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/analyzers/defererr"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(defererr.Analyzer)
}
