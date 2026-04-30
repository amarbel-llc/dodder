package repo_actions

import (
	"code.linenisgreat.com/dodder/go/internal/charlie/haustoria"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/object_metadata_fmt_hyphence"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/lima/orgie"
	"code.linenisgreat.com/dodder/go/internal/lima/store_workspace"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
)

func MakeCheckout(r *local_working_copy.Repo) Checkout {
	return Checkout{repo: r}
}

func MakeCheckin(r *local_working_copy.Repo) Checkin {
	return Checkin{repo: r}
}

func MakeDiff(
	r *local_working_copy.Repo,
	formatterFamily object_metadata_fmt_hyphence.FormatterFamily,
) Diff {
	return Diff{
		repo:            r,
		FormatterFamily: formatterFamily,
	}
}

func MakeCreateFromPaths(
	r *local_working_copy.Repo,
	textParser object_metadata_fmt_hyphence.Parser,
) CreateFromPaths {
	return CreateFromPaths{
		repo:       r,
		TextParser: textParser,
	}
}

func MakeCreateFromShas(r *local_working_copy.Repo) CreateFromShas {
	return CreateFromShas{repo: r}
}

func MakeOrganize(
	r *local_working_copy.Repo,
	metadata orgie.Metadata,
) Organize {
	return Organize{
		repo:     r,
		Metadata: metadata,
	}
}

func MakeOrganize2(
	r *local_working_copy.Repo,
	metadata orgie.Metadata,
) Organize2 {
	return Organize2{
		repo:     r,
		Metadata: metadata,
	}
}

func MakeUpdateObject(r *local_working_copy.Repo) UpdateObject {
	return UpdateObject{repo: r}
}

func MakeWriteNewZettels(r *local_working_copy.Repo) WriteNewZettels {
	return WriteNewZettels{repo: r}
}

func MakeExecLua(r *local_working_copy.Repo) ExecLua {
	return ExecLua{repo: r}
}

func MakeOpenEditor(r *local_working_copy.Repo) OpenEditor {
	return OpenEditor{repo: r}
}

func MakeEachBlob(
	r *local_working_copy.Repo,
	utility string,
) EachBlob {
	return EachBlob{
		repo:    r,
		Utility: utility,
	}
}

func MakeCreateOrganizeFile(
	r *local_working_copy.Repo,
	options orgie.Options,
) CreateOrganizeFile {
	return CreateOrganizeFile{
		repo:    r,
		Options: options,
	}
}

func MakeReadOrganizeFile(r *local_working_copy.Repo) ReadOrganizeFile {
	return ReadOrganizeFile{repo: r}
}

func MakeCheckinHaustoria(
	r *local_working_copy.Repo,
	h haustoria.Haustoria,
	storeLike store_workspace.StoreLike,
	query *queries.Query,
) CheckinHaustoria {
	return CheckinHaustoria{
		repo:      r,
		Haustoria: h,
		StoreLike: storeLike,
		Query:     query,
	}
}

func MakeNewHaustoria(
	r *local_working_copy.Repo,
	h haustoria.Haustoria,
	textParser object_metadata_fmt_hyphence.Parser,
	proto sku.Proto,
) NewHaustoria {
	return NewHaustoria{
		repo:       r,
		Haustoria:  h,
		TextParser: textParser,
		Proto:      proto,
	}
}
