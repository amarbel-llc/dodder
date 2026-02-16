package blob_stores

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/markl_io"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
)

type remoteSftp struct {
	ctx       interfaces.ActiveContext
	uiPrinter ui.Printer
	once      sync.Once

	buckets []int

	config blob_store_configs.ConfigSFTPRemotePath

	multiHash       bool
	defaultHashType markl.FormatHash

	// TODO populate blobIOWrapper with env_repo.FileNameBlobStoreConfig at
	// `config.GetRemotePath()`
	blobIOWrapper        domain_interfaces.BlobIOWrapper
	sshClientInitializer func() (*ssh.Client, error)
	sshClient            *ssh.Client
	sftpClient           *sftp.Client

	// TODO extract below into separate struct
	blobCacheLock sync.RWMutex
	blobCache     map[string]struct{}
}

var _ domain_interfaces.BlobStore = &remoteSftp{}

func makeSftpStore(
	ctx interfaces.ActiveContext,
	uiPrinter ui.Printer,
	config blob_store_configs.ConfigSFTPRemotePath,
	sshClientInitializer func() (*ssh.Client, error),
) (blobStore *remoteSftp, err error) {
	var defaultHashType markl.FormatHash

	if defaultHashType, err = markl.GetFormatHashOrError(
		blob_store_configs.DefaultHashTypeId,
	); err != nil {
		err = errors.Wrap(err)
		return blobStore, err
	}

	blobStore = &remoteSftp{
		ctx:                  ctx,
		defaultHashType:      defaultHashType,
		uiPrinter:            uiPrinter,
		buckets:              defaultBuckets,
		config:               config,
		blobCache:            make(map[string]struct{}),
		sshClientInitializer: sshClientInitializer,
	}

	return blobStore, err
}

func (blobStore *remoteSftp) GetBlobStoreConfig() blob_store_configs.Config {
	return blobStore.config
}

func (blobStore *remoteSftp) GetDefaultHashType() domain_interfaces.FormatHash {
	return blobStore.defaultHashType
}

func (blobStore *remoteSftp) close() (err error) {
	if blobStore.sftpClient != nil {
		if err = blobStore.sftpClient.Close(); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return nil
}

func (blobStore *remoteSftp) initializeOnce() {
	blobStore.once.Do(func() {
		if err := blobStore.initialize(); err != nil {
			err = errors.Wrap(err)
			blobStore.ctx.Cancel(err)
		}
	})
}

func (blobStore *remoteSftp) initialize() (err error) {
	if blobStore.sshClient, err = blobStore.sshClientInitializer(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if blobStore.sftpClient, err = sftp.NewClient(blobStore.sshClient); err != nil {
		err = errors.Wrapf(err, "failed to create SFTP client")
		return err
	}

	blobStore.ctx.After(errors.MakeFuncContextFromFuncErr(blobStore.close))

	remotePath := blobStore.config.GetRemotePath()
	// TODO read remote blob store config (including hash buckets)

	// Create directory tree if it doesn't exist
	parts := strings.Split(remotePath, "/")
	var currentPath string

	for _, part := range parts {
		if part == "" {
			continue
		}

		if currentPath == "" && !strings.HasPrefix(remotePath, "/") {
			currentPath = part
		} else {
			currentPath = path.Join(currentPath, part)
		}

		blobStore.uiPrinter.Printf("checking directory %q...", currentPath)
		if _, err = blobStore.sftpClient.Stat(currentPath); err != nil {
			// TODO check error
			blobStore.uiPrinter.Printf("creating directory %q...", currentPath)
			if err = blobStore.sftpClient.Mkdir(currentPath); err != nil {
				// Directory might exist, continue
				err = nil
			}
		}
	}

	return err
}

func (blobStore *remoteSftp) GetBlobStoreDescription() string {
	return "remote sftp hash bucketed"
}

func (blobStore *remoteSftp) GetBlobIOWrapper() domain_interfaces.BlobIOWrapper {
	blobStore.initializeOnce()
	return blobStore.blobIOWrapper
}

func (blobStore *remoteSftp) GetLocalBlobStore() domain_interfaces.BlobStore {
	return blobStore
}

func (blobStore *remoteSftp) makeEnvDirConfig() env_dir.Config {
	return env_dir.DefaultConfig
	// return env_dir.MakeConfig(
	// 	blobStore.blobIOWrapper.GetBlobCompression(),
	// 	blobStore.blobIOWrapper.GetBlobEncryption(),
	// )
}

func (blobStore *remoteSftp) remotePathForMerkleId(
	merkleId domain_interfaces.MarklId,
) string {
	return env_dir.MakeHashBucketPathFromMerkleId(
		merkleId,
		blobStore.buckets,
		blobStore.multiHash,
		strings.TrimPrefix(blobStore.config.GetRemotePath(), "/"),
	)
}

func (blobStore *remoteSftp) HasBlob(
	merkleId domain_interfaces.MarklId,
) (ok bool) {
	blobStore.initializeOnce()

	if merkleId.IsNull() {
		ok = true
		return ok
	}

	blobStore.blobCacheLock.RLock()

	if _, ok = blobStore.blobCache[string(merkleId.GetBytes())]; ok {
		blobStore.blobCacheLock.RUnlock()
		return ok
	}

	blobStore.blobCacheLock.RUnlock()

	remotePath := blobStore.remotePathForMerkleId(merkleId)

	if _, err := blobStore.sftpClient.Stat(remotePath); err == nil {
		blobStore.blobCacheLock.Lock()
		blobStore.blobCache[string(merkleId.GetBytes())] = struct{}{}
		blobStore.blobCacheLock.Unlock()
		ok = true
	}

	return ok
}

func (blobStore *remoteSftp) AllBlobs() interfaces.SeqError[domain_interfaces.MarklId] {
	blobStore.initializeOnce()

	return func(yield func(domain_interfaces.MarklId, error) bool) {
		basePath := strings.TrimPrefix(blobStore.config.GetRemotePath(), "/")

		// Walk through the two-level directory structure (Git-like bucketing)
		walker := blobStore.sftpClient.Walk(basePath)

		digest, repool := blobStore.defaultHashType.GetBlobId()
		defer repool()

		for walker.Step() {
			if err := walker.Err(); err != nil {
				if !yield(nil, errors.Wrapf(err, "BasePath: %q", basePath)) {
					return
				}

				continue
			}

			info := walker.Stat()

			if info.IsDir() {
				continue
			}

			currentPath := walker.Path()

			{
				var err error

				if currentPath, err = filepath.Rel(basePath, currentPath); err != nil {
					if !yield(
						nil,
						errors.Wrapf(err, "BasePath: %q", basePath),
					) {
						return
					}
				}
			}

			if err := markl.SetHexStringFromAbsolutePath(
				digest,
				currentPath,
				basePath,
			); err != nil {
				if !yield(nil, errors.Wrap(err)) {
					return
				}

				continue
			}

			blobStore.blobCacheLock.Lock()
			blobStore.blobCache[string(digest.GetBytes())] = struct{}{}
			blobStore.blobCacheLock.Unlock()

			if !yield(digest, nil) {
				return
			}
		}
	}
}

func (blobStore *remoteSftp) MakeBlobWriter(
	marklHashType domain_interfaces.FormatHash,
) (blobWriter domain_interfaces.BlobWriter, err error) {
	blobStore.initializeOnce()

	// TODO use hash type
	mover := &sftpMover{
		store:  blobStore,
		config: blobStore.makeEnvDirConfig(),
	}

	hash, hashRepool := blobStore.defaultHashType.Get()
	defer hashRepool()

	if err = mover.initialize(hash); err != nil {
		err = errors.Wrap(err)
		return blobWriter, err
	}

	blobWriter = mover

	return blobWriter, err
}

func (blobStore *remoteSftp) MakeBlobReader(
	digest domain_interfaces.MarklId,
) (readCloser domain_interfaces.BlobReader, err error) {
	blobStore.initializeOnce()

	if digest.IsNull() {
		hash, hashRepool := blobStore.defaultHashType.Get()
		defer hashRepool()
		readCloser = markl_io.MakeNopReadCloser(
			hash,
			ohio.NopCloser(bytes.NewReader(nil)),
		)
		return readCloser, err
	}

	remotePath := blobStore.remotePathForMerkleId(digest)

	var remoteFile *sftp.File

	if remoteFile, err = blobStore.sftpClient.Open(remotePath); err != nil {
		if os.IsNotExist(err) {
			clonedDigest, _ := markl.Clone(digest)
			err = env_dir.ErrBlobMissing{
				BlobId: clonedDigest,
				Path:   remotePath,
			}
		} else {
			err = errors.Wrap(err)
		}
		return readCloser, err
	}

	blobStore.blobCacheLock.Lock()
	blobStore.blobCache[string(digest.GetBytes())] = struct{}{}
	blobStore.blobCacheLock.Unlock()

	// Create streaming reader that handles decompression/decryption
	config := blobStore.makeEnvDirConfig()
	streamingReader := &sftpStreamingReader{
		file:   remoteFile,
		config: config,
	}

	readerHash, readerHashRepool := blobStore.defaultHashType.Get()
	defer readerHashRepool()

	if readCloser, err = streamingReader.createReader(
		readerHash,
	); err != nil {
		remoteFile.Close()
		err = errors.Wrap(err)
		return readCloser, err
	}

	return readCloser, err
}

// sftpMover implements interfaces.Mover and interfaces.ShaWriteCloser
// TODO explore using env_dir.Mover generically instead of this
type sftpMover struct {
	hash     domain_interfaces.Hash
	store    *remoteSftp
	config   env_dir.Config
	tempFile *sftp.File
	tempPath string
	writer   *sftpWriter
	closed   bool
}

func (mover *sftpMover) initialize(hash domain_interfaces.Hash) (err error) {
	mover.hash = hash

	// Create a temporary file on the remote server
	var tempNameBytes [16]byte
	if _, err = rand.Read(tempNameBytes[:]); err != nil {
		err = errors.Wrap(err)
		return err
	}

	tempName := fmt.Sprintf("tmp_%x", tempNameBytes)
	mover.tempPath = path.Join(mover.store.config.GetRemotePath(), tempName)

	if mover.tempFile, err = mover.store.sftpClient.Create(
		mover.tempPath,
	); err != nil {
		ui.Debug().Printf("unable to create temp file: %q", mover.tempPath)
		err = errors.Wrapf(err, "unable to create temp file: %q", mover.tempPath)
		return err
	}

	// Create the streaming writer with compression/encryption

	if mover.writer, err = newSftpWriter(
		mover.config,
		mover.tempFile,
		hash,
	); err != nil {
		mover.tempFile.Close()
		mover.store.sftpClient.Remove(mover.tempPath)
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (mover *sftpMover) Write(p []byte) (n int, err error) {
	if mover.writer == nil {
		err = errors.ErrorWithStackf("writer not initialized")
		return n, err
	}

	return mover.writer.Write(p)
}

func (mover *sftpMover) ReadFrom(r io.Reader) (n int64, err error) {
	if mover.writer == nil {
		err = errors.ErrorWithStackf("writer not initialized")
		return n, err
	}

	return mover.writer.ReadFrom(r)
}

func (mover *sftpMover) Close() (err error) {
	if mover.closed {
		return nil
	}

	mover.closed = true

	// Ensure cleanup happens
	// TODO capture errors using errors.Deferred*
	defer func() {
		if mover.tempFile != nil {
			mover.tempFile.Close()
		}
		if mover.tempPath != "" {
			mover.store.sftpClient.Remove(mover.tempPath)
		}
	}()

	if mover.writer == nil {
		// No data was written
		return nil
	}

	// Close the writer to finalize compression/encryption and digest
	// calculation
	if err = mover.writer.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Close the temp file
	if err = mover.tempFile.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Get the calculated digest and determine final path
	blobDigest := mover.writer.GetDigest()
	finalPath := mover.store.remotePathForMerkleId(blobDigest)

	// Ensure the target directory exists (Git-like bucketing)
	finalDir := path.Dir(finalPath)
	if err = mover.store.sftpClient.MkdirAll(finalDir); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Atomically move temp file to final location
	if err = mover.store.sftpClient.Rename(mover.tempPath, finalPath); err != nil {
		// Check if file already exists
		if _, statErr := mover.store.sftpClient.Stat(finalPath); statErr == nil {
			// File already exists, this is OK - just remove temp file
			mover.store.sftpClient.Remove(mover.tempPath)
			err = nil
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	mover.store.blobCacheLock.Lock()
	mover.store.blobCache[string(blobDigest.GetBytes())] = struct{}{}
	mover.store.blobCacheLock.Unlock()

	// Clear temp path so cleanup doesn't try to remove it
	mover.tempPath = ""

	return err
}

func (mover *sftpMover) GetMarklId() domain_interfaces.MarklId {
	if mover.writer == nil {
		return mover.GetMarklId()
	}

	return mover.writer.GetDigest()
}

// sftpWriter implements the streaming writer with compression/encryption
type sftpWriter struct {
	hash            domain_interfaces.Hash
	tee             io.Writer
	wCompress, wAge io.WriteCloser
	wBuf            *bufio.Writer
}

func newSftpWriter(
	config env_dir.Config,
	ioWriter io.Writer,
	hash domain_interfaces.Hash,
) (writer *sftpWriter, err error) {
	writer = &sftpWriter{}

	writer.wBuf = bufio.NewWriter(ioWriter)

	if writer.wAge, err = config.GetBlobEncryption().WrapWriter(writer.wBuf); err != nil {
		err = errors.Wrap(err)
		return writer, err
	}

	writer.hash = hash

	if writer.wCompress, err = config.GetBlobCompression().WrapWriter(writer.wAge); err != nil {
		err = errors.Wrap(err)
		return writer, err
	}

	writer.tee = io.MultiWriter(writer.hash, writer.wCompress)

	return writer, err
}

func (writer *sftpWriter) ReadFrom(r io.Reader) (n int64, err error) {
	if n, err = io.Copy(writer.tee, r); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (writer *sftpWriter) Write(p []byte) (n int, err error) {
	return writer.tee.Write(p)
}

func (writer *sftpWriter) Close() (err error) {
	if writer.wCompress != nil {
		if err = writer.wCompress.Close(); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if writer.wAge != nil {
		if err = writer.wAge.Close(); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if writer.wBuf != nil {
		if err = writer.wBuf.Flush(); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (writer *sftpWriter) GetDigest() domain_interfaces.MarklId {
	id, _ := writer.hash.GetMarklId()
	return id
}

// sftpStreamingReader handles decompression/decryption while reading from SFTP
// TODO combine with sftpReader
type sftpStreamingReader struct {
	file   *sftp.File
	config env_dir.Config
}

func (reader *sftpStreamingReader) createReader(
	hash domain_interfaces.Hash,
) (readCloser domain_interfaces.BlobReader, err error) {
	// Create streaming reader with decompression/decryption
	sftpReader := &sftpReader{
		file:   reader.file,
		config: reader.config,
	}

	if err = sftpReader.initialize(hash); err != nil {
		err = errors.Wrap(err)
		return readCloser, err
	}

	readCloser = sftpReader

	return readCloser, err
}

// sftpReader implements streaming decompression/decryption for SFTP
type sftpReader struct {
	file      *sftp.File
	config    env_dir.Config
	hash      domain_interfaces.Hash
	decrypter io.Reader
	expander  io.ReadCloser
	tee       io.Reader
}

func (reader *sftpReader) initialize(hash domain_interfaces.Hash) (err error) {
	// Set up decryption
	if reader.decrypter, err = reader.config.GetBlobEncryption().WrapReader(reader.file); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Set up decompression
	if reader.expander, err = reader.config.GetBlobCompression().WrapReader(reader.decrypter); err != nil {
		err = errors.Wrap(err)
		return err
	}

	reader.hash = hash
	reader.tee = io.TeeReader(reader.expander, reader.hash)

	return err
}

func (reader *sftpReader) Read(p []byte) (n int, err error) {
	return reader.tee.Read(p)
}

func (reader *sftpReader) WriteTo(w io.Writer) (n int64, err error) {
	return io.Copy(w, reader.tee)
}

func (reader *sftpReader) Seek(
	offset int64,
	whence int,
) (actual int64, err error) {
	seeker, ok := reader.decrypter.(io.Seeker)

	if !ok {
		err = errors.ErrorWithStackf("seeking not supported")
		return actual, err
	}

	return seeker.Seek(offset, whence)
}

func (reader *sftpReader) ReadAt(p []byte, off int64) (n int, err error) {
	readerAt, ok := reader.decrypter.(io.ReaderAt)

	if !ok {
		err = errors.ErrorWithStackf("reading at not supported")
		return n, err
	}

	return readerAt.ReadAt(p, off)
}

func (reader *sftpReader) Close() error {
	// TODO capture both errors using errors.Join
	err1 := reader.expander.Close()
	err2 := reader.file.Close()

	if err1 != nil {
		return err1
	}
	return err2
}

func (reader *sftpReader) GetMarklId() domain_interfaces.MarklId {
	id, _ := reader.hash.GetMarklId()
	return id
}
