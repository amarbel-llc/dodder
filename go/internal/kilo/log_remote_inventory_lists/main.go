package log_remote_inventory_lists

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

type EntryType interface {
	entryType()
}

//go:generate stringer -type=entryType
type entryType byte

func (entryType) entryType() {}

const (
	EntryTypeSent = entryType(iota)
	EntryTypeReceived
)

type Entry struct {
	EntryType
	PublicKey domain_interfaces.MarklId
	*sku.Transacted
}

type Log interface {
	errors.Flusher
	initialize(errors.Context, env_repo.Env)
	Key(Entry) (string, error)
	Append(Entry) error
	Exists(Entry) error
}

func Make(ctx errors.Context, envRepo env_repo.Env) (log Log) {
	log = &v0{}

	log.initialize(ctx, envRepo)
	ctx.After(errors.MakeFuncContextFromFuncErr(log.Flush))

	return log
}
