package zettel_id_provider

import (
	"path"
	"sync"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/directory_layout"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/foxtrot/object_id_log"
)

const (
	FilePathZettelIdYin  = "Yin"
	FilePathZettelIdYang = "Yang"
)

type Provider struct {
	sync.Locker
	yin  provider
	yang provider
}

// BlobResolver fetches a blob by its MarklId and returns the newline-delimited
// words it contains.
type BlobResolver func(markl.Id) ([]string, error)

func New(ps directory_layout.RepoMutable) (f *Provider, err error) {
	providerPathYin := path.Join(ps.DirObjectId(), FilePathZettelIdYin)
	providerPathYang := path.Join(ps.DirObjectId(), FilePathZettelIdYang)

	f = &Provider{
		Locker: &sync.Mutex{},
	}

	if f.yin, err = newProvider(providerPathYin); err != nil {
		err = errors.Wrap(err)
		return f, err
	}

	if f.yang, err = newProvider(providerPathYang); err != nil {
		err = errors.Wrap(err)
		return f, err
	}

	return f, err
}

// NewFromLog builds a Provider by replaying the object ID log. Each log entry
// references a blob containing delta words; resolveBlob fetches those words.
// When the log does not exist or is empty, it falls back to reading flat files
// via New.
func NewFromLog(
	directoryLayout directory_layout.RepoMutable,
	resolveBlob BlobResolver,
) (f *Provider, err error) {
	log := object_id_log.Log{Path: directoryLayout.FileObjectIdLog()}

	var entries []object_id_log.Entry

	if entries, err = log.ReadAllEntries(); err != nil {
		err = errors.Wrap(err)
		return f, err
	}

	if len(entries) == 0 {
		return New(directoryLayout)
	}

	f = &Provider{
		Locker: &sync.Mutex{},
	}

	for _, entry := range entries {
		var words []string

		if words, err = resolveBlob(entry.GetMarklId()); err != nil {
			err = errors.Wrapf(err, "resolving blob for log entry")
			return f, err
		}

		switch entry.GetSide() {
		case object_id_log.SideYin:
			f.yin = append(f.yin, words...)

		case object_id_log.SideYang:
			f.yang = append(f.yang, words...)

		default:
			err = errors.ErrorWithStackf("unknown side: %d", entry.GetSide())
			return f, err
		}
	}

	return f, err
}

func (hf *Provider) Left() provider {
	return hf.yin
}

func (hf *Provider) Right() provider {
	return hf.yang
}
