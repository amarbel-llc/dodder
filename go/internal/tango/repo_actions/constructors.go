package repo_actions

import (
	"code.linenisgreat.com/dodder/go/internal/charlie/haustoria"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/object_metadata_fmt_hyphence"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/lima/organize_text"
	"code.linenisgreat.com/dodder/go/internal/lima/store_workspace"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
)

func MakeCheckout(repo *local_working_copy.Repo) Checkout {
	return Checkout{Repo: repo}
}

func MakeCheckin(repo *local_working_copy.Repo) Checkin {
	return Checkin{Repo: repo}
}

func MakeDiff(
	repo *local_working_copy.Repo,
	formatterFamily object_metadata_fmt_hyphence.FormatterFamily,
) Diff {
	return Diff{
		Repo:            repo,
		FormatterFamily: formatterFamily,
	}
}

func MakeCreateFromPaths(
	repo *local_working_copy.Repo,
	textParser object_metadata_fmt_hyphence.Parser,
) CreateFromPaths {
	return CreateFromPaths{
		Repo:       repo,
		TextParser: textParser,
	}
}

func MakeCreateFromShas(repo *local_working_copy.Repo) CreateFromShas {
	return CreateFromShas{Repo: repo}
}

func MakeOrganize(
	repo *local_working_copy.Repo,
	metadata organize_text.Metadata,
) Organize {
	return Organize{
		Repo:     repo,
		Metadata: metadata,
	}
}

func MakeOrganize2(
	repo *local_working_copy.Repo,
	metadata organize_text.Metadata,
) Organize2 {
	return Organize2{
		Repo:     repo,
		Metadata: metadata,
	}
}

func MakeUpdateObject(repo *local_working_copy.Repo) UpdateObject {
	return UpdateObject{Repo: repo}
}

func MakeWriteNewZettels(repo *local_working_copy.Repo) WriteNewZettels {
	return WriteNewZettels{Repo: repo}
}

func MakeExecLua(repo *local_working_copy.Repo) ExecLua {
	return ExecLua{Repo: repo}
}

func MakeOpenEditor(repo *local_working_copy.Repo) OpenEditor {
	return OpenEditor{Repo: repo}
}

func MakeEachBlob(
	repo *local_working_copy.Repo,
	utility string,
) EachBlob {
	return EachBlob{
		Repo:    repo,
		Utility: utility,
	}
}

func MakeCreateOrganizeFile(
	repo *local_working_copy.Repo,
	options organize_text.Options,
) CreateOrganizeFile {
	return CreateOrganizeFile{
		Repo:    repo,
		Options: options,
	}
}

func MakeReadOrganizeFile(repo *local_working_copy.Repo) ReadOrganizeFile {
	return ReadOrganizeFile{Repo: repo}
}

func MakeCheckinHaustoria(
	repo *local_working_copy.Repo,
	h haustoria.Haustoria,
	storeLike store_workspace.StoreLike,
	query *queries.Query,
) CheckinHaustoria {
	return CheckinHaustoria{
		Repo:      repo,
		Haustoria: h,
		StoreLike: storeLike,
		Query:     query,
	}
}

func MakeNewHaustoria(
	repo *local_working_copy.Repo,
	h haustoria.Haustoria,
	textParser object_metadata_fmt_hyphence.Parser,
	proto sku.Proto,
) NewHaustoria {
	return NewHaustoria{
		Repo:       repo,
		Haustoria:  h,
		TextParser: textParser,
		Proto:      proto,
	}
}
