package main

import (
	"os"

	"code.linenisgreat.com/dodder/go/internal/delta/store_version"
	"code.linenisgreat.com/dodder/go/internal/zulu/commands_dodder"
)

func main() {
	store_version.VCurrent = store_version.VNext
	store_version.VNext = store_version.VNull
	utility := commands_dodder.GetUtility("dodder")
	utility.Run(os.Args)
}
