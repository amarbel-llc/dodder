package domain_interfaces

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
)

type (
	Genre interface {
		GenreGetter
		interfaces.Stringer
		IsConfig() bool
		IsNone() bool
		IsTag() bool
		IsType() bool
		IsZettel() bool
		GetGenreBitInt() byte
	}

	GenreGetter interface {
		GetGenre() Genre
	}

	ObjectId interface {
		GenreGetter
		interfaces.Stringer
		IsEmpty() bool
	}

	ExternalObjectId interface {
		ObjectId
		ExternalObjectIdGetter
	}

	ExternalObjectIdGetter interface {
		GetExternalObjectId() ExternalObjectId
	}

	RepoId interface {
		interfaces.Stringer
		GetRepoIdString() string
	}

	RepoIdGetter interface {
		GetRepoId() RepoId
	}

	Abbreviatable interface {
		interfaces.Stringer
	}

	FuncExpandString     func(string) (string, error)
	FuncAbbreviateString func(Abbreviatable) (string, error)

	Abbreviator struct {
		Expand     FuncExpandString
		Abbreviate FuncAbbreviateString
	}
)
