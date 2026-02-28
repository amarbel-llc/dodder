package object_id_provider

import (
	"path"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
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

func (hf *Provider) Left() provider {
	return hf.yin
}

func (hf *Provider) Right() provider {
	return hf.yang
}
