package object_metadata_fmt_hyphence

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
	"code.linenisgreat.com/dodder/go/lib/delta/script_config"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/format"
)

type formatterComponents Factory

func (factory formatterComponents) getWriteTypeAndSigFunc() funcWrite {
	if factory.AllowMissingTypeSig || true {
		return factory.writeTypeAndSigIfNecessary
	} else {
		return factory.writeTypeAndSig
	}
}

func (factory formatterComponents) writeComments(
	writer interfaces.WriterAndStringWriter,
	context FormatterContext,
) (n int64, err error) {
	n1 := 0

	for comment := range context.GetMetadata().GetIndex().GetComments() {
		n1, err = io.WriteString(writer, "% ")
		n += int64(n1)

		if err != nil {
			return n, err
		}

		n1, err = io.WriteString(writer, comment)
		n += int64(n1)

		if err != nil {
			return n, err
		}

		n1, err = io.WriteString(writer, "\n")
		n += int64(n1)

		if err != nil {
			return n, err
		}
	}

	return n, err
}

func (factory formatterComponents) writeBoundary(
	writer interfaces.WriterAndStringWriter,
	_ FormatterContext,
) (n int64, err error) {
	return ohio.WriteLine(writer, hyphence.Boundary)
}

func (factory formatterComponents) writeNewLine(
	writer interfaces.WriterAndStringWriter,
	_ FormatterContext,
) (n int64, err error) {
	return ohio.WriteLine(writer, "")
}

func (factory formatterComponents) writeCommonMetadataFormat(
	writer interfaces.WriterAndStringWriter,
	formatterContext FormatterContext,
) (n int64, err error) {
	lineWriter := format.NewLineWriter()

	metadata := formatterContext.GetMetadata()
	description := metadata.GetDescription()

	if description.String() != "" || !formatterContext.DoNotWriteEmptyDescription {
		reader, repool := pool.GetStringReader(description.String())
		defer repool()

		stringReader := bufio.NewReader(reader)

		for {
			var line string
			line, err = stringReader.ReadString('\n')
			isEOF := err == io.EOF

			if err != nil && !isEOF {
				err = errors.Wrap(err)
				return n, err
			}

			lineWriter.WriteLines(
				fmt.Sprintf("# %s", strings.TrimSpace(line)),
			)

			if isEOF {
				break
			}
		}
	}

	for _, tag := range quiter.SortedValues(metadata.AllTags()) {
		if ids.IsEmpty(tag) {
			continue
		}

		lineWriter.WriteFormat("- %s", tag)
	}

	if n, err = lineWriter.WriteTo(writer); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (factory formatterComponents) writeTypeAndSigIfNecessary(
	writer interfaces.WriterAndStringWriter,
	formatterContext FormatterContext,
) (n int64, err error) {
	metadata := formatterContext.GetMetadata()
	typeTuple := metadata.GetTypeLock()

	if typeTuple.GetKey().IsEmpty() {
		return n, err
	}

	if typeTuple.GetValue().IsEmpty() {
		return ohio.WriteLine(
			writer,
			fmt.Sprintf(
				"! %s",
				typeTuple.GetKey().ToType().StringSansOp(),
			),
		)
	}

	return factory.writeTypeAndSig(writer, formatterContext)
}

func (factory formatterComponents) writeTypeAndSig(
	writer interfaces.WriterAndStringWriter,
	formatterContext FormatterContext,
) (n int64, err error) {
	metadata := formatterContext.GetMetadata()
	typeTuple := metadata.GetTypeLock()

	if typeTuple.GetKey().IsEmpty() {
		return n, err
	}

	if typeTuple.GetValue().IsEmpty() {
		err = errors.Errorf("empty type signature for type: %q", typeTuple.GetKey())
		return n, err
	}

	return ohio.WriteLine(
		writer,
		fmt.Sprintf(
			"! %s@%s",
			typeTuple.GetKey().ToType().StringSansOp(),
			typeTuple.GetValue(),
		),
	)
}

func (factory formatterComponents) writeReferencedObjects(
	writer interfaces.WriterAndStringWriter,
	formatterContext FormatterContext,
) (n int64, err error) {
	metadata := formatterContext.GetMetadata()

	for ref := range metadata.AllReferencedObjects() {
		lockValue := metadata.GetReferencedObjectLock(ref).GetValue()

		var line string

		alias := metadata.GetReferenceAlias(ref)

		if alias != "" {
			if strings.ContainsAny(alias, " \t\"") {
				alias = fmt.Sprintf("%q", alias)
			}

			if lockValue.IsEmpty() {
				line = fmt.Sprintf("- %s < %s", alias, ref)
			} else {
				line = fmt.Sprintf("- %s < %s@%s", alias, ref, lockValue)
			}
		} else {
			if lockValue.IsEmpty() {
				line = fmt.Sprintf("- %s", ref)
			} else {
				line = fmt.Sprintf("- %s@%s", ref, lockValue)
			}
		}

		var n1 int64
		if n1, err = ohio.WriteLine(writer, line); err != nil {
			return n, err
		}
		n += n1
	}

	return n, err
}

func (factory formatterComponents) writeBlobReferences(
	writer interfaces.WriterAndStringWriter,
	formatterContext FormatterContext,
) (n int64, err error) {
	metadata := formatterContext.GetMetadata()

	for blobId := range metadata.AllBlobReferences() {
		var line string

		alias := metadata.GetBlobReferenceAlias(blobId)

		typeLock := metadata.GetBlobReferenceTypeLock(blobId)
		typeLockStr := markl.MakeLockCoderValueNotRequired(typeLock).String()

		if alias != "" {
			if strings.ContainsAny(alias, " \t\"") {
				alias = fmt.Sprintf("%q", alias)
			}

			line = fmt.Sprintf("- %s < @%s", alias, blobId)
		} else {
			line = fmt.Sprintf("- @%s", blobId)
		}

		if typeLockStr != "" {
			line = fmt.Sprintf("%s %s", line, typeLockStr)
		}

		var n1 int64
		if n1, err = ohio.WriteLine(writer, line); err != nil {
			return n, err
		}
		n += n1
	}

	return n, err
}

func (factory formatterComponents) writeBlobDigest(
	writer interfaces.WriterAndStringWriter,
	formatterContext FormatterContext,
) (n int64, err error) {
	metadata := formatterContext.GetMetadata()

	return ohio.WriteLine(
		writer,
		fmt.Sprintf(
			"@ %s",
			metadata.GetBlobDigest(),
		),
	)
}

func (factory formatterComponents) writeBlobPath(
	writer interfaces.WriterAndStringWriter,
	formatterContext FormatterContext,
) (n int64, err error) {
	var blobPath string

	metadata := formatterContext.GetMetadata()

	for field := range metadata.GetIndex().GetFields() {
		if strings.ToLower(field.Key) == "blob" {
			blobPath = field.Value
			break
		}
	}

	if blobPath != "" {
		blobPath = factory.EnvDir.RelToCwdOrSame(blobPath)
	} else {
		err = errors.ErrorWithStackf("path not found in fields")
		return n, err
	}

	return ohio.WriteLine(writer, fmt.Sprintf("@ %s", blobPath))
}

func (factory formatterComponents) writeBlob(
	writer interfaces.WriterAndStringWriter,
	formatterContext FormatterContext,
) (n int64, err error) {
	var blobReader domain_interfaces.BlobReader

	metadata := formatterContext.GetMetadata()

	if blobReader, err = factory.BlobStore.MakeBlobReader(
		metadata.GetBlobDigest(),
	); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if blobReader == nil {
		err = errors.ErrorWithStackf("blob reader is nil")
		return n, err
	}

	defer errors.DeferredCloser(&err, blobReader)

	if factory.BlobFormatter != nil {
		var writerTo io.WriterTo

		env := factory.EnvDir.MakeCommonEnv()

		if factory.BlobTreeDir != "" {
			env["DODDER_BLOB_TREE"] = factory.BlobTreeDir
		}

		if writerTo, err = script_config.MakeWriterToWithStdin(
			factory.BlobFormatter,
			env,
			blobReader,
		); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		if n, err = writerTo.WriteTo(writer); err != nil {
			var errExit *exec.ExitError

			if errors.As(err, &errExit) {
				err = MakeErrBlobFormatterFailed(errExit)
			}

			err = errors.Wrap(err)

			return n, err
		}
	} else {
		if n, err = io.Copy(writer, blobReader); err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}
