package ui

import (
	"os"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

var Null null

func init() {
	var err error
	Null.file, err = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	errors.PanicIfError(err)
}

type null struct {
	file *os.File
}

var _ Printer = null{}

func (null) Write(b []byte) (n int, err error) {
	n = len(b)
	return n, err
}

func (printer null) GetFile() *os.File {
	return printer.file
}

func (null) IsTty() bool {
	return false
}

func (printer null) Caller(_ int) Printer {
	return printer
}

func (null) PrintDebug(_ ...any) (err error) {
	return err
}

func (null) Print(_ ...any) (err error) {
	return err
}

func (null) Printf(_ string, _ ...any) (err error) {
	return err
}
