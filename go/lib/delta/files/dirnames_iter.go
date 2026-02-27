package files

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

// TODO migrate all dir walking to this package

type WalkDirEntry struct {
	Path    string
	RelPath string
	os.DirEntry
}

type WalkDirEntryIgnoreFunc func(WalkDirEntry) bool

func WalkDirIgnoreFuncHidden(dirEntry WalkDirEntry) bool {
	if strings.HasPrefix(dirEntry.RelPath, ".") {
		return true
	}

	return false
}

func WalkDir(
	base string,
) interfaces.SeqError[WalkDirEntry] {
	return func(yield func(WalkDirEntry, error) bool) {
		if err := filepath.WalkDir(
			base,
			func(path string, dirEntry os.DirEntry, in error) (out error) {
				if in != nil {
					out = in
					return out
				}

				entry := WalkDirEntry{
					Path:     path,
					DirEntry: dirEntry,
				}

				if entry.RelPath, out = filepath.Rel(base, path); out != nil {
					out = errors.Wrap(out)
					return out
				}

				if entry.RelPath == "." {
					return out
				}

				if !yield(entry, nil) {
					out = fs.SkipAll
					return out
				}

				return out
			},
		); err != nil {
			yield(WalkDirEntry{}, errors.Wrap(err))
			return
		}
	}
}

func DirNames2(p string) interfaces.SeqError[os.DirEntry] {
	return func(yield func(os.DirEntry, error) bool) {
		var names []os.DirEntry

		{
			var err error

			if names, err = ReadDir(p); err != nil {
				yield(nil, errors.Wrap(err))
				return
			}
		}

		for _, dirEntry := range names {
			if !yield(dirEntry, nil) {
				return
			}
		}
	}
}

func DirEntries(
	dirPath string,
) (dirEntries collections_slice.Slice[os.DirEntry], err error) {
	if dirEntries, err = ReadDir(dirPath); err != nil {
		err = errors.Wrap(err)
		return dirEntries, err
	}

	return dirEntries, err
}

func DirNames(dirPath string) (slice collections_slice.Slice[string], err error) {
	var names []os.DirEntry

	if names, err = ReadDir(dirPath); err != nil {
		err = errors.Wrap(err)
		return slice, err
	}

	for _, dirEntry := range names {
		slice.Append(path.Join(dirPath, dirEntry.Name()))
	}

	return slice, err
}

func DirNameWriterIgnoringHidden(
	seq interfaces.SeqError[string],
) interfaces.SeqError[string] {
	return func(yield func(string, error) bool) {
		for path, err := range seq {
			if err != nil {
				yield(path, err)
				return
			}

			b := filepath.Base(path)

			if strings.HasPrefix(b, ".") {
				return
			}

			if !yield(path, err) {
				return
			}
		}
	}
}

func DirNamesLevel2(
	dirPath string,
) interfaces.SeqError[string] {
	return func(yield func(string, error) bool) {
		var topLevel collections_slice.Slice[string]

		{
			var err error

			if topLevel, err = DirNames(dirPath); err != nil {
				yield("", err)
				return
			}
		}

		for topLevelDir := range topLevel.All() {
			var secondLevel collections_slice.Slice[string]

			{
				var err error

				if secondLevel, err = DirNames(topLevelDir); err != nil {
					yield("", err)
					return
				}
			}

			for secondLevelDir := range secondLevel.All() {
				if !yield(secondLevelDir, nil) {
					return
				}
			}
		}
	}
}
