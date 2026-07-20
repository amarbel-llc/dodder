package command

import "code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"

type CommandComponentReader interface {
	GetCLIFlags() []string
}

type CommandComponent interface {
	CommandComponentReader
	interfaces.CommandComponentWriter
}
