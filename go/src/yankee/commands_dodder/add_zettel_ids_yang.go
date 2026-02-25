package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/src/foxtrot/object_id_log"
	"code.linenisgreat.com/dodder/go/src/foxtrot/object_id_provider"
)

func init() {
	utility.AddCmd("add-zettel-ids-yang", &AddZettelIds{
		side:         object_id_log.SideYang,
		flatFileName: object_id_provider.FilePathZettelIdYang,
	})
}
