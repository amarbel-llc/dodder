package log_remote_inventory_lists

import (
	"bufio"
	"encoding/gob"
	"io"
	"os"
	"path/filepath"
	"sync"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/files"
	"code.linenisgreat.com/dodder/go/lib/charlie/tridex"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
	"code.linenisgreat.com/dodder/go/internal/hotel/file_lock"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
)

// TODO consider moving this or refactoring it as it's currently not really used
type v0 struct {
	once      sync.Once
	path      string
	lockSmith interfaces.LockSmith
	file      *os.File
	values    interfaces.TridexMutable
}

func (log *v0) Flush() (err error) {
	if _, err = log.file.Seek(0, io.SeekStart); err != nil {
		err = errors.Wrap(err)
		return err
	}

	bufferedWriter := bufio.NewWriter(log.file)

	enc := gob.NewEncoder(bufferedWriter)

	if err = enc.Encode(log.values); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = bufferedWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = log.file.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = log.lockSmith.Unlock(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return nil
}

func (log *v0) initialize(ctx errors.Context, env env_repo.Env) {
	gob.Register(tridex.Make())

	log.values = tridex.Make()

	dir := env.DirCacheRemoteInventoryListsLog()

	log.path = filepath.Join(dir, "log-v0")
	log.lockSmith = file_lock.New(
		env,
		filepath.Join(dir, "log-v0.lock"),
		"log_remote_inventory_lists",
	)

	if err := log.lockSmith.Lock(); err != nil {
		ctx.Cancel(err)
		return
	}

	{
		var err error

		if log.file, err = files.TryOrMakeDirIfNecessary(
			log.path,
			files.OpenCreate,
		); err != nil {
			ctx.Cancel(err)
			return
		}
	}
}

func (log *v0) Append(entry Entry) (err error) {
	if err = log.readIfNecessary(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var key string

	if key, err = log.Key(entry); err != nil {
		err = errors.Wrap(err)
		return err
	}

	log.values.Add(key)

	return err
}

func (log *v0) Key(entry Entry) (key string, err error) {
	if entry.EntryType == nil {
		err = errors.ErrorWithStackf("nil entry type")
		return key, err
	}

	// TODO determine via config
	digest, repool := markl.FormatHashSha256.GetMarklIdFromStringFormat(
		"%s%s%s%s",
		entry.EntryType,
		entry.PublicKey.StringWithFormat(),
		entry.GetObjectId(),
		entry.GetBlobDigest(),
	)

	defer repool()

	// TODO determine via config, and switch to digest.String()
	key = markl.FormatBytesAsHex(digest)

	return key, err
}

func (log *v0) Exists(entry Entry) (err error) {
	if err = log.readIfNecessary(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var key string

	if key, err = log.Key(entry); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if !log.values.ContainsExpansion(key) {
		return errors.MakeErrNotFoundString(key)
	}

	return err
}

func (log *v0) readIfNecessary() (err error) {
	log.once.Do(
		func() {
			bufferedReader := bufio.NewReader(log.file)

			dec := gob.NewDecoder(bufferedReader)

			if err = dec.Decode(log.values); err != nil {
				if errors.IsEOF(err) {
					err = nil
				} else {
					err = errors.Wrap(err)
				}

				return
			}
		},
	)

	return err
}
