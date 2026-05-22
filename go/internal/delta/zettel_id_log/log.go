package zettel_id_log

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	charlie_zil "code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_log"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"github.com/amarbel-llc/madder/go/pkgs/hyphence"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/files"
)

type Log struct {
	Path string
}

// AppendEntry appends `entry` to the log on disk. The first call against
// an empty (or nonexistent) file writes a single hyphence header
// (`---\n! zettel_id_log-vN\n---\n`) followed by the entry body.
// Subsequent calls append just a body, separated from prior content by a
// blank line.
//
// Background: an earlier implementation wrapped every entry in a fresh
// hyphence TypedBlob and emitted the full header per call, producing
// stacked hyphence docs on disk (amarbel-llc/dodder#212). v15 had not
// shipped to users, so the legacy reader path was dropped along with
// the fixture regen.
func (l Log) AppendEntry(entry Entry) (err error) {
	var file *os.File

	if file, err = files.OpenFile(
		l.Path,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0o666,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	var stat os.FileInfo

	if stat, err = file.Stat(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if stat.Size() == 0 {
		// The type constant already carries a leading `!`
		// (e.g. `!zettel_id_log-v1`). Hyphence's type-header line is
		// `! <type-without-prefix>`, so strip the prefix before
		// printing — otherwise we'd emit `! !zettel_id_log-v1`.
		typeName := strings.TrimPrefix(
			ids.TypeZettelIdLogVCurrent,
			"!",
		)

		if _, err = fmt.Fprintf(
			file,
			"%s\n! %s\n%s\n\n",
			hyphence.Boundary,
			typeName,
			hyphence.Boundary,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		// Blank-line separator between bodies.
		if _, err = io.WriteString(file, "\n"); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	var body []byte

	if body, err = encodeEntryBody(entry); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if _, err = file.Write(body); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// encodeEntryBody returns the body-only encoding of an entry — the TOML
// key/value block, with no surrounding hyphence boundaries or type
// header. Mirrors what `charlie_zil.V1Document.Encode` returns directly.
func encodeEntryBody(entry Entry) ([]byte, error) {
	doc, err := charlie_zil.DecodeV1(nil)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	switch v := entry.(type) {
	case *V1:
		*doc.Data() = *v
	case V1:
		*doc.Data() = v
	default:
		return nil, errors.Errorf("unsupported entry type %T", entry)
	}

	body, err := doc.Encode()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return body, nil
}

func (l Log) ReadAllEntries() (entries []Entry, err error) {
	var file *os.File

	if file, err = files.Open(l.Path); err != nil {
		if errors.IsNotExist(err) {
			err = nil
			return entries, err
		}

		err = errors.Wrap(err)
		return entries, err
	}

	defer errors.DeferredCloser(&err, file)

	bufferedReader := bufio.NewReader(file)

	bodies, err := segmentBodies(bufferedReader)
	if err != nil {
		err = errors.Wrap(err)
		return entries, err
	}

	for _, body := range bodies {
		doc, err := charlie_zil.DecodeV1([]byte(body))
		if err != nil {
			return entries, errors.Wrap(err)
		}

		v := *doc.Data()
		entries = append(entries, v)
	}

	return entries, err
}

// segmentBodies reads a zettel_id_log file and returns the entry bodies
// (TOML key/value blocks) in order.
//
// File shape: a single hyphence header at the top
// (`---\n! zettel_id_log-v1\n---\n`), then bodies separated by blank
// lines. The header is skipped; bodies are collected line-by-line and
// flushed on blank-line boundaries.
func segmentBodies(reader *bufio.Reader) (bodies []string, err error) {
	var current strings.Builder

	flush := func() {
		if current.Len() == 0 {
			return
		}
		bodies = append(bodies, current.String())
		current.Reset()
	}

	// Skip the leading header block: a `---` line, then arbitrary
	// header content, then a closing `---` line. Tolerate a missing
	// header (e.g. a zero-length log).
	sawOpeningBoundary := false
	pastHeader := false

	for line, errIter := range ohio.MakeLineSeqFromReader(reader) {
		if errIter != nil {
			err = errIter
			return bodies, err
		}

		trimmedRight := strings.TrimSuffix(line, "\n")

		if !pastHeader {
			if trimmedRight == hyphence.Boundary {
				if sawOpeningBoundary {
					pastHeader = true
				} else {
					sawOpeningBoundary = true
				}
				continue
			}

			if !sawOpeningBoundary {
				// Header missing or empty log; just treat the
				// content as bodies from here on.
				pastHeader = true
			} else {
				// Header content (e.g. `! zettel_id_log-v1`).
				continue
			}
		}

		if trimmedRight == "" {
			flush()
			continue
		}

		current.WriteString(line)
	}

	flush()

	return bodies, err
}
