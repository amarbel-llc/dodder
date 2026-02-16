package stream_index

import (
	"bufio"
	"os"
	"slices"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

type reindexCursor struct {
	tai            ids.Tai
	objectIdString string
	cursor         ohio.Cursor
}

type pageAdditionsFileBacked struct {
	index *Index

	file           *os.File
	bufferedWriter *bufio.Writer
	encoder        binaryEncoder
	offset         int64

	objectIds     map[string]struct{}
	objectCursors []reindexCursor

	decoder binaryDecoder
}

func (fb *pageAdditionsFileBacked) initialize(index *Index) (err error) {
	fb.index = index
	fb.objectIds = make(map[string]struct{})
	fb.decoder = makeBinary(ids.SigilHistory)

	if fb.file, err = index.envRepo.GetTempLocal().FileTemp(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var repool interfaces.FuncRepool
	fb.bufferedWriter, repool = pool.GetBufferedWriter(fb.file)
	_ = repool // bufferedWriter will be flushed and file closed in Close()

	return err
}

func (fb *pageAdditionsFileBacked) add(object *sku.Transacted) {
	objectClone, _ := object.CloneTransacted()

	fb.objectIds[object.ObjectId.String()] = struct{}{}

	var cursor ohio.Cursor
	cursor.Offset = fb.offset

	n, err := fb.encoder.writeFormat(
		fb.bufferedWriter,
		objectWithSigil{Transacted: object},
	)
	if err != nil {
		panic(errors.Wrap(err))
	}

	cursor.ContentLength = n
	fb.offset += n

	fb.objectCursors = append(fb.objectCursors, reindexCursor{
		tai:            object.GetTai(),
		objectIdString: object.ObjectId.String(),
		cursor:         cursor,
	})

	seqProbeIds := object.AllProbeIds(
		fb.index.index.GetHashType(),
		fb.index.defaultObjectDigestMarklFormatId,
	)

	additionProbes := fb.index.probeIndex.additionProbes

	for probeId := range seqProbeIds {
		idBytes := probeId.Id.GetBytes()
		additionProbes.Set(string(idBytes), objectClone)
	}
}

func (fb *pageAdditionsFileBacked) hasChanges() bool {
	return fb.Len() > 0
}

func (fb *pageAdditionsFileBacked) Len() int {
	return len(fb.objectCursors)
}

func (fb *pageAdditionsFileBacked) Reset() {
	fb.objectCursors = fb.objectCursors[:0]
	fb.objectIds = make(map[string]struct{})
	fb.offset = 0
}

func (fb *pageAdditionsFileBacked) All() interfaces.Seq[*sku.Transacted] {
	return func(yield func(*sku.Transacted) bool) {
		if len(fb.objectCursors) == 0 {
			return
		}

		if err := fb.bufferedWriter.Flush(); err != nil {
			panic(errors.Wrap(err))
		}

		sorted := make([]reindexCursor, len(fb.objectCursors))
		copy(sorted, fb.objectCursors)

		slices.SortFunc(sorted, func(a, b reindexCursor) int {
			if result := a.tai.SortCompare(b.tai); !result.IsEqual() {
				if result.IsLess() {
					return -1
				}
				return 1
			}

			if a.objectIdString < b.objectIdString {
				return -1
			} else if a.objectIdString > b.objectIdString {
				return 1
			}
			return 0
		})

		for _, rc := range sorted {
			object, objectRepool := sku.GetTransactedPool().GetWithRepool()

			var objectWithCS objectWithCursorAndSigil
			objectWithCS.Transacted = object
			objectWithCS.Cursor = rc.cursor

			if _, err := fb.decoder.readFormatExactly(
				fb.file,
				&objectWithCS,
			); err != nil {
				objectRepool()
				panic(errors.Wrap(err))
			}

			if !yield(object) {
				objectRepool()
				return
			}
		}
	}
}

func (fb *pageAdditionsFileBacked) containsObjectId(objectIdString string) bool {
	_, ok := fb.objectIds[objectIdString]
	return ok
}

func (fb *pageAdditionsFileBacked) Close() (err error) {
	if fb.bufferedWriter != nil {
		if flushErr := fb.bufferedWriter.Flush(); flushErr != nil {
			err = errors.Wrap(flushErr)
		}
	}

	if fb.file != nil {
		name := fb.file.Name()

		if closeErr := fb.file.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr)
		}

		if removeErr := os.Remove(name); removeErr != nil && err == nil {
			if !errors.IsNotExist(removeErr) {
				err = errors.Wrap(removeErr)
			}
		}
	}

	return err
}
