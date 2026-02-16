package dormant_index

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
	"code.linenisgreat.com/dodder/go/src/foxtrot/tag_paths"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

type Index struct {
	changes []string
	tags    tag_paths.TagsWithParentsAndTypes
}

func (index *Index) GetChanges() (out []string) {
	out = make([]string, len(index.changes))
	copy(out, index.changes)

	return out
}

func (index *Index) HasChanges() bool {
	return len(index.changes) > 0
}

func (index *Index) AddDormantTag(e *tag_paths.Tag) (err error) {
	index.changes = append(index.changes, fmt.Sprintf("added %q", e))

	if err = index.tags.Add(e, nil); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (index *Index) RemoveDormantTag(e *tag_paths.Tag) (err error) {
	index.changes = append(index.changes, fmt.Sprintf("removed %q", e))

	if err = index.tags.Remove(e); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (index *Index) ContainsSku(object *sku.Transacted) bool {
	for _, tag := range index.tags {
		if tag.Len() == 0 {
			panic("empty dormant tag")
		}

		all := object.GetMetadata().GetIndex().GetTagPaths().All

		if _, ok := all.ContainsTag(tag.Tag); ok {
			return true
		}
	}

	return false
}

func (index *Index) Load(envRepo env_repo.Env) (err error) {
	var file *os.File

	path := envRepo.FileCacheDormant()

	if file, err = files.Open(path); err != nil {
		if errors.IsNotExist(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
		}

		return err
	}

	defer errors.DeferredCloser(&err, file)

	bufferedReader, repool := pool.GetBufferedReader(file)
	defer repool()

	if _, err = index.ReadFrom(bufferedReader); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (index *Index) Flush(
	envRepo env_repo.Env,
	printerHeader interfaces.FuncIter[string],
	dryRun bool,
) (err error) {
	if len(index.changes) == 0 {
		ui.Log().Print("no dormant changes")
		return err
	}

	if dryRun {
		ui.Log().Print("no dormant flush, dry run")
		return err
	}

	if err = printerHeader("writing dormant"); err != nil {
		err = errors.Wrap(err)
		return err
	}

	path := envRepo.FileCacheDormant()

	var file *os.File

	if file, err = files.OpenExclusiveWriteOnlyTruncate(path); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	bufferedWriter, repool := pool.GetBufferedWriter(file)
	defer repool()

	defer errors.DeferredFlusher(&err, bufferedWriter)

	if _, err = index.WriteTo(bufferedWriter); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = printerHeader("wrote dormant"); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (index *Index) ReadFrom(bufferedReader *bufio.Reader) (n int64, err error) {
	index.tags.GetSlice().Reset()
	var count uint16

	var n1 int64
	count, n1, err = ohio.ReadUint16(bufferedReader)
	n += n1
	// n += int64(n1)
	if err != nil {
		if errors.IsEOF(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
		}

		return n, err
	}

	index.tags = slices.Grow(index.tags, int(count))

	for i := uint16(0); i < count; i++ {
		var l uint16

		var n1 int64
		l, n1, err = ohio.ReadUint16(bufferedReader)
		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		var cs *catgut.String

		var csRepool interfaces.FuncRepool
		if cs, csRepool, err = catgut.MakeFromReader(bufferedReader, int(l)); err != nil {
			_ = csRepool
			err = errors.Wrap(err)
			return n, err
		}

		index.tags = append(index.tags, tag_paths.TagWithParentsAndTypes{
			Tag: cs,
		})
	}

	return n, err
}

func (index Index) WriteTo(writer io.Writer) (n int64, err error) {
	count := uint16(index.tags.Len())

	var n1 int
	var n2 int64
	n1, err = ohio.WriteUint16(writer, count)
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	for _, tags := range index.tags {
		count := uint16(tags.Len())

		n1, err = ohio.WriteUint16(writer, count)
		n += int64(n1)
		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		n2, err = tags.WriteTo(writer)
		n += n2

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}
