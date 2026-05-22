package command

import "github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"

type CommandComponentReader interface {
	GetCLIFlags() []string
}

type CommandComponent interface {
	CommandComponentReader
	interfaces.CommandComponentWriter
}
