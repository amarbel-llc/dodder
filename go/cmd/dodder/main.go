package main

import (
	"os"

	"code.linenisgreat.com/dodder/go/internal/yankee/commands_dodder"
)

func main() {
	utility := commands_dodder.GetUtility("dodder")
	utility.Run(os.Args)
}
