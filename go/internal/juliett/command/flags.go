package command

import "code.linenisgreat.com/dodder/go/internal/_/interfaces"

type CommandComponentReader interface {
	GetCLIFlags() []string
}

type CommandComponent interface {
	CommandComponentReader
	interfaces.CommandComponentWriter
}
