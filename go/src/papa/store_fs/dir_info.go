package store_fs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter_set"
	"code.linenisgreat.com/dodder/go/src/charlie/collections_value"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/delta/file_extensions"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/foxtrot/fd"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/oscar/store_workspace"
)

type itemWithError struct {
	error
	*sku.FSItem
}

// TODO support globs and ignores
type dirInfo struct {
	root          string
	rootProcessed bool

	fileExtensions file_extensions.Config
	envRepo        env_repo.Env
	storeSupplies  store_workspace.Supplies

	probablyCheckedOut      fsItemData
	definitelyNotCheckedOut fsItemData

	errors interfaces.SetMutable[itemWithError]
}

type (
	dirEntryKey = string
	dirInfoMap  map[dirEntryKey]*sku.FSItem
)

func makeObjectsWithDir(
	fileExtensions file_extensions.Config,
	envRepo env_repo.Env,
) (info dirInfo) {
	info.fileExtensions = fileExtensions
	info.envRepo = envRepo
	info.probablyCheckedOut = makeFSItemData()
	info.definitelyNotCheckedOut = makeFSItemData()
	info.errors = collections_value.MakeMutableValueSet[itemWithError](nil)

	return info
}

//  __        __    _ _    _
//  \ \      / /_ _| | | _(_)_ __   __ _
//   \ \ /\ / / _` | | |/ / | '_ \ / _` |
//    \ V  V / (_| | |   <| | | | | (_| |
//     \_/\_/ \__,_|_|_|\_\_|_| |_|\__, |
//                                 |___/

func (dirInfo *dirInfo) walkDir(
	cache dirInfoMap,
	dir string,
	pattern string,
) (err error) {
	if err = filepath.WalkDir(
		dir,
		func(path string, dirEntry fs.DirEntry, in error) (err error) {
			if in != nil {
				err = errors.Wrap(in)
				return err
			}

			if path == dirInfo.root {
				return err
			}

			if dirEntry.Type()&fs.ModeSymlink != 0 {
				if path, err = filepath.EvalSymlinks(path); err != nil {
					err = nil
					return err
					// err = errors.Wrap(err)
					// return
				}

				var fi fs.FileInfo

				if fi, err = os.Lstat(path); err != nil {
					err = errors.Wrap(err)
					return err
				}

				dirEntry = fs.FileInfoToDirEntry(fi)
			}

			if strings.HasPrefix(filepath.Base(path), ".") {
				if dirEntry.IsDir() {
					err = filepath.SkipDir
				}

				return err
			}

			if pattern != "" {
				var matched bool

				if matched, err = filepath.Match(pattern, path); err != nil {
					err = errors.Wrap(err)
					return err
				}

				if !matched {
					return err
				}
			}

			if dirEntry.IsDir() {
				fileWorkspace := filepath.Join(path, env_repo.FileWorkspace)

				if files.Exists(fileWorkspace) {
					err = filepath.SkipDir
				}

				return err
			}

			if _, _, err = dirInfo.addPathAndDirEntry(cache, path, dirEntry); err != nil {
				err = errors.Wrapf(err, "DirEntry: %s", dirEntry)
				return err
			}

			return err
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (dirInfo *dirInfo) addPathAndDirEntry(
	cache dirInfoMap,
	path string,
	dirEntry fs.DirEntry,
) (key string, fds *sku.FSItem, err error) {
	if dirEntry.IsDir() {
		return key, fds, err
	}

	var fdee *fd.FD

	if fdee, err = fd.MakeFromPathAndDirEntry(
		path,
		dirEntry,
		dirInfo.envRepo.GetDefaultBlobStore(),
	); err != nil {
		err = errors.Wrap(err)
		return key, fds, err
	}

	if key, fds, err = dirInfo.addFD(cache, fdee); err != nil {
		err = errors.Wrap(err)
		return key, fds, err
	}

	return key, fds, err
}

func (dirInfo *dirInfo) keyForFD(fdee *fd.FD) (key string, err error) {
	if fdee.ExtSansDot() == dirInfo.fileExtensions.Config {
		key = "konfig"
		return key, err
	}

	path := fdee.GetPath()

	if path == "" {
		err = errors.ErrorWithStackf("empty path")
		return key, err
	}

	if !fd.DoesDirectoryContainPath(dirInfo.root, path) {
		err = errors.Errorf("fdee (%q) not in root (%q): rel %q", fdee, dirInfo.root, path)
		return
	}

	var rel string

	if rel, err = filepath.Rel(dirInfo.root, path); err != nil {
		err = errors.Wrap(err)
		return key, err
	}

	if rel == "" {
		err = errors.ErrorWithStackf("empty rel path")
		return key, err
	}

	// calculating the relative path is definitely necessary, just not sure why
	key = dirInfo.keyForObjectIdString(rel)

	if key == "" {
		err = errors.ErrorWithStackf("empty key for rel path: %q", rel)
		return key, err
	}

	return key, err
}

// TODO determine if this method should key zettel id's with the parent dir too
func (dirInfo *dirInfo) keyForObjectIdString(
	objectIdString string,
) (key string) {
	var ok bool

	key, _, ok = strings.Cut(objectIdString, ".")

	if !ok {
		key = objectIdString
	} else if key == "" {
		key = fd.FileNameSansExt(objectIdString)
	}

	return key
}

func (dirInfo *dirInfo) addFD(
	cache dirInfoMap,
	fileDescriptor *fd.FD,
) (key string, fds *sku.FSItem, err error) {
	if fileDescriptor.IsDir() {
		return key, fds, err
	}

	if key, err = dirInfo.keyForFD(fileDescriptor); err != nil {
		err = errors.Wrap(err)
		return key, fds, err
	}

	if cache == nil {
		fds = &sku.FSItem{
			FDs: collections_value.MakeMutableValueSet[*fd.FD](nil),
		}

		fds.FDs.Add(fileDescriptor)
	} else {
		fds = cache[key]

		if fds == nil {
			fds = &sku.FSItem{
				FDs: collections_value.MakeMutableValueSet[*fd.FD](nil),
			}
		}

		fds.FDs.Add(fileDescriptor)
		cache[key] = fds
	}

	return key, fds, err
}

//   ____                              _
//  |  _ \ _ __ ___   ___ ___  ___ ___(_)_ __   __ _
//  | |_) | '__/ _ \ / __/ _ \/ __/ __| | '_ \ / _` |
//  |  __/| | | (_) | (_|  __/\__ \__ \ | | | | (_| |
//  |_|   |_|  \___/ \___\___||___/___/_|_| |_|\__, |
//                                             |___/

func (dirInfo *dirInfo) processDir(
	path string,
) (results []*sku.FSItem, err error) {
	cache := make(dirInfoMap)

	results = make([]*sku.FSItem, 0)

	if err = dirInfo.walkDir(cache, path, ""); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	for objectIdString, fds := range cache {
		var someResult []*sku.FSItem

		if someResult, err = dirInfo.processFDSet(objectIdString, fds); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		results = append(results, someResult...)
	}

	return results, err
}

func (dirInfo *dirInfo) processFDPattern(
	objectIdString string,
	pattern string,
	dir string,
) (fds []*sku.FSItem, err error) {
	cache := make(dirInfoMap)

	if err = dirInfo.walkDir(cache, dir, pattern); err != nil {
		err = errors.Wrap(err)
		return fds, err
	}

	item := cache[objectIdString]

	if item == nil {
		return fds, err
	}

	if fds, err = dirInfo.processFDSet(
		objectIdString,
		item,
	); err != nil {
		err = errors.Wrapf(
			err,
			"FD: %q, ObjectIdString: %q",
			item.Debug(),
			objectIdString,
		)
		return fds, err
	}

	return fds, err
}

func (dirInfo *dirInfo) processFD(
	fdee *fd.FD,
) (objectIdString string, fds []*sku.FSItem, err error) {
	cache := make(dirInfoMap)

	if objectIdString, err = dirInfo.keyForFD(fdee); err != nil {
		err = errors.Wrap(err)
		return objectIdString, fds, err
	}

	dir := filepath.Dir(fdee.GetPath())
	pattern := filepath.Join(dir, fmt.Sprintf("%s*", fdee.FileNameSansExt()))

	if err = dirInfo.walkDir(cache, dir, pattern); err != nil {
		err = errors.Wrap(err)
		return objectIdString, fds, err
	}

	item := cache[objectIdString]

	if item == nil {
		err = errors.ErrorWithStackf(
			"failed to write FSItem to cache. Cache: %s, Pattern: %s, ObjectId: %s, Dir: %s",
			cache,
			pattern,
			objectIdString,
			dir,
		)

		return objectIdString, fds, err
	}

	if fds, err = dirInfo.processFDSet(
		objectIdString,
		item,
	); err != nil {
		err = errors.Wrapf(
			err,
			"FD: %q, ObjectIdString: %q",
			item.Debug(),
			objectIdString,
		)
		return objectIdString, fds, err
	}

	return objectIdString, fds, err
}

func (dirInfo *dirInfo) processRootDir() (err error) {
	if dirInfo.rootProcessed {
		return err
	}

	if _, err = dirInfo.processDir(dirInfo.root); err != nil {
		err = errors.Wrap(err)
		return err
	}

	dirInfo.rootProcessed = true

	return err
}

func (dirInfo *dirInfo) processFDsOnItem(
	item *sku.FSItem,
) (blobCount, objectCount int, err error) {
	for fd := range item.FDs.All() {
		ext := fd.ExtSansDot()

		switch ext {
		case dirInfo.fileExtensions.Zettel:
			item.ExternalObjectId.SetGenre(genres.Zettel)

		case dirInfo.fileExtensions.Type:
			item.ExternalObjectId.SetGenre(genres.Type)

		case dirInfo.fileExtensions.Tag:
			item.ExternalObjectId.SetGenre(genres.Tag)

		case dirInfo.fileExtensions.Repo:
			item.ExternalObjectId.SetGenre(genres.Repo)

		case dirInfo.fileExtensions.Lockfile:
			item.Lockfile.ResetWith(fd)

		case dirInfo.fileExtensions.Conflict:
			item.Conflict.ResetWith(fd)
			continue

		default: // blobs
			item.Blob.ResetWith(fd)
			blobCount++
			continue
		}

		item.Object.ResetWith(fd)
		objectCount++
	}

	return blobCount, objectCount, err
}

func (dirInfo *dirInfo) processFDSet(
	objectIdString string,
	item *sku.FSItem,
) (results []*sku.FSItem, err error) {
	var recognizedGenre genres.Genre

	{
		recognized, recognizedRepool := sku.GetTransactedPool().GetWithRepool()
		defer recognizedRepool()

		var objectId ids.ObjectId

		if err = objectId.Set(objectIdString); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		if err = dirInfo.storeSupplies.ReadOneInto(
			&objectId,
			recognized,
		); err != nil {
			if errors.IsErrNotFound(err) {
				err = nil
			} else {
				err = errors.Wrapf(err, "ObjectId: %q", objectIdString)
				return results, err
			}
		} else {
			recognizedGenre = genres.Must(recognized.GetGenre())
		}
	}

	var blobCount, objectCount int

	if blobCount, objectCount, err = dirInfo.processFDsOnItem(item); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	if item.ExternalObjectId.GetGenre() != genres.Unknown {
		if blobCount > 1 {
			err = errors.ErrorWithStackf(
				"several blobs matching object id %q: %q",
				objectIdString,
				item.FDs,
			)
		} else if objectCount > 1 {
			err = errors.ErrorWithStackf(
				"found more than one object: %q",
				item.FDs,
			)
		}

		if err != nil {
			if err = dirInfo.errors.Add(itemWithError{FSItem: item, error: err}); err != nil {
				err = errors.Wrap(err)
				return results, err
			}
		}
	}

	if item.ExternalObjectId.GetGenre() == genres.Unknown {
		item.ExternalObjectId.SetGenre(recognizedGenre)
	}

	if item.ExternalObjectId.GetGenre() == genres.Unknown {
		if results, err = dirInfo.addOneOrMoreBlobs(
			item,
		); err != nil {
			err = errors.Wrap(err)
			return results, err
		}
	} else {
		if err = dirInfo.addOneObject(
			objectIdString,
			item,
		); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		results = []*sku.FSItem{item}
	}

	return results, err
}

func (dirInfo *dirInfo) addOneUntracked(
	fdee *fd.FD,
) (result *sku.FSItem, err error) {
	result = &sku.FSItem{
		FDs: collections_value.MakeMutableValueSet[*fd.FD](nil),
	}

	result.Blob.ResetWith(fdee)

	if err = result.FDs.Add(&result.Blob); err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	if err = result.ExternalObjectId.SetBlob(
		dirInfo.envRepo.Rel(fdee.GetPath()),
	); err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	if err = dirInfo.definitelyNotCheckedOut.Add(result); err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	digest := fdee.GetDigest()

	if digest.IsNull() {
		return result, err
	}

	// TODO try reading as object

	// TODO add sha cache
	key := digest.GetBytes()
	existing, ok := dirInfo.definitelyNotCheckedOut.digests[string(key)]

	if !ok {
		existing = collections_value.MakeMutableValueSet[*sku.FSItem](nil)
	}

	if err = existing.Add(result); err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	dirInfo.definitelyNotCheckedOut.digests[string(key)] = existing

	return result, err
}

func (dirInfo *dirInfo) addOneOrMoreBlobs(
	fds *sku.FSItem,
) (results []*sku.FSItem, err error) {
	if fds.FDs.Len() == 1 {
		var fdsOne *sku.FSItem

		if fdsOne, err = dirInfo.addOneUntracked(
			quiter_set.Any(fds.FDs),
		); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		fdsOne.ExternalObjectId.SetGenre(genres.Blob)
		results = []*sku.FSItem{fdsOne}

		return results, err
	}

	for range fds.FDs.All() {
		var fdsOne *sku.FSItem

		if fdsOne, err = dirInfo.addOneUntracked(
			quiter_set.Any(fds.FDs),
		); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		results = append(results, fdsOne)
	}

	return results, err
}

func (dirInfo *dirInfo) addOneObject(
	objectIdString string,
	item *sku.FSItem,
) (err error) {
	if err = item.ExternalObjectId.Set(objectIdString); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = dirInfo.probablyCheckedOut.Add(item); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

//   ___ _                 _   _
//  |_ _| |_ ___ _ __ __ _| |_(_) ___  _ __
//   | || __/ _ \ '__/ _` | __| |/ _ \| '_ \
//   | || ||  __/ | | (_| | |_| | (_) | | | |
//  |___|\__\___|_|  \__,_|\__|_|\___/|_| |_|
//

// TODO switch to seq.Iter2
func (dirInfo *dirInfo) All(
	output interfaces.FuncIter[*sku.FSItem],
) (err error) {
	waitGroup := errors.MakeWaitGroupParallel()

	quiter.ErrorWaitGroupApply(waitGroup, dirInfo.probablyCheckedOut, output)
	quiter.ErrorWaitGroupApply(waitGroup, dirInfo.definitelyNotCheckedOut, output)

	return waitGroup.GetError()
}
