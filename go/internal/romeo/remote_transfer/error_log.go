package remote_transfer

import (
	"fmt"
	"os"
	"time"

	"code.linenisgreat.com/dodder/go/internal/golf/sku"
)

type importErrorLog struct {
	path  string
	file  *os.File
	count int
}

func (l *importErrorLog) ensureOpen() error {
	if l.file != nil {
		return nil
	}

	l.path = fmt.Sprintf(
		"import-errors-%s.log",
		time.Now().Format("20060102-150405"),
	)

	var err error
	if l.file, err = os.Create(l.path); err != nil {
		return err
	}

	return nil
}

func (l *importErrorLog) LogError(object *sku.Transacted, err error) error {
	if openErr := l.ensureOpen(); openErr != nil {
		return openErr
	}

	_, writeErr := fmt.Fprintf(
		l.file,
		"%s\t%s\n",
		sku.String(object),
		err,
	)

	if writeErr != nil {
		return writeErr
	}

	l.count++

	return nil
}

func (l *importErrorLog) Count() int {
	return l.count
}

func (l *importErrorLog) Path() string {
	return l.path
}

func (l *importErrorLog) Close() error {
	if l.file == nil {
		return nil
	}

	return l.file.Close()
}
