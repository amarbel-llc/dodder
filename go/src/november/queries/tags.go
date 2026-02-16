package queries

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
	"code.linenisgreat.com/dodder/go/src/foxtrot/tag_paths"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

// TODO move implicit tags here
type Tags struct {
	changes collections_slice.Slice[string]
	tags    tag_paths.TagsWithParentsAndTypes
}

func (tags *Tags) HasChanges() bool {
	return !tags.changes.IsEmpty()
}

func (tags *Tags) AddTag(e *tag_paths.Tag) (err error) {
	tags.changes.Append(fmt.Sprintf("added %q", e))

	if err = tags.tags.Add(e, nil); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (tags *Tags) RemoveDormantTag(e *tag_paths.Tag) (err error) {
	tags.changes.Append(fmt.Sprintf("removed %q", e))

	if err = tags.tags.Remove(e); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (tags *Tags) ContainsSku(object *sku.Transacted) bool {
	for _, tag := range tags.tags {
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

func (tags *Tags) Load(envRepo env_repo.Env) (err error) {
	var file *os.File

	path := envRepo.FileTags()

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

	if _, err = tags.ReadFrom(bufferedReader); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (tags *Tags) Flush(
	envRepo env_repo.Env,
	printerHeader interfaces.FuncIter[string],
	dryRun bool,
) (err error) {
	if tags.changes.IsEmpty() {
		ui.Log().Print("no tags changes")
		return err
	}

	if dryRun {
		ui.Log().Print("no tags flush, dry run")
		return err
	}

	if err = printerHeader("writing dormant tags"); err != nil {
		err = errors.Wrap(err)
		return err
	}

	path := envRepo.FileTags()

	var file *os.File

	if file, err = files.OpenExclusiveWriteOnlyTruncate(path); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	bufferedWriter, repool := pool.GetBufferedWriter(file)
	defer repool()

	defer errors.DeferredFlusher(&err, bufferedWriter)

	if _, err = tags.WriteTo(bufferedWriter); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = printerHeader("wrote dormant"); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (tags *Tags) ReadFrom(bufferedReader *bufio.Reader) (n int64, err error) {
	tags.tags.GetSlice().Reset()
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

	tags.tags.GetSlice().Grow(int(count))

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

		if cs, _, err = catgut.MakeFromReader(bufferedReader, int(l)); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		tags.tags.GetSlice().Append(tag_paths.TagWithParentsAndTypes{
			Tag: cs,
		})
	}

	return n, err
}

func (tags Tags) WriteTo(w io.Writer) (n int64, err error) {
	count := uint16(tags.tags.Len())

	var n1 int
	var n2 int64
	n1, err = ohio.WriteUint16(w, count)
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	for _, tag := range tags.tags {
		l := uint16(tag.Len())

		n1, err = ohio.WriteUint16(w, l)
		n += int64(n1)
		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		n2, err = tag.WriteTo(w)
		n += n2

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}
