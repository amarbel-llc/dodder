package sku

import (
	"bufio"
	"io"
	"testing"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

type spyBlobWriter struct {
	closeCount int
}

func (writer *spyBlobWriter) Write(bites []byte) (int, error) {
	return len(bites), nil
}

func (writer *spyBlobWriter) ReadFrom(reader io.Reader) (int64, error) {
	return io.Copy(io.Discard, reader)
}

func (writer *spyBlobWriter) Close() error {
	writer.closeCount++
	return nil
}

func (writer *spyBlobWriter) GetMarklId() mad_domain_interfaces.MarklId {
	return nil
}

type spyBlobWriterFactory struct {
	writers []*spyBlobWriter
}

func (factory *spyBlobWriterFactory) make() (mad_domain_interfaces.BlobWriter, error) {
	writer := &spyBlobWriter{}
	factory.writers = append(factory.writers, writer)
	return writer, nil
}

type fakeListCoder struct{}

func (fakeListCoder) EncodeTo(
	object *Transacted,
	writer *bufio.Writer,
) (int64, error) {
	n, err := writer.WriteString("object\n")
	return int64(n), err
}

func (fakeListCoder) DecodeFrom(
	object *Transacted,
	reader *bufio.Reader,
) (int64, error) {
	return 0, nil
}

func TestWorkingListEmptyCloseOpensNoBlobWriter(t1 *testing.T) {
	ui.RunTestContext(t1, testWorkingListEmptyCloseOpensNoBlobWriter)
}

// A working list that never receives an Add must never open a blob
// writer: writers create their temp file at construction, so an
// eagerly-opened writer on a list discarded without Close leaks an
// empty temp file (issue #366).
func testWorkingListEmptyCloseOpensNoBlobWriter(t *ui.TestContext) {
	factory := &spyBlobWriterFactory{}

	list := MakeWorkingList(fakeListCoder{}, factory.make, nil)

	t.AssertEqual(0, len(factory.writers))

	t.AssertNoError(list.Close())

	t.AssertEqual(0, len(factory.writers))
}

func TestWorkingListAddOpensBlobWriterOnce(t1 *testing.T) {
	ui.RunTestContext(t1, testWorkingListAddOpensBlobWriterOnce)
}

func testWorkingListAddOpensBlobWriterOnce(t *ui.TestContext) {
	factory := &spyBlobWriterFactory{}

	list := MakeWorkingList(fakeListCoder{}, factory.make, nil)

	object, repool := GetTransactedPool().GetWithRepool()
	defer repool()

	t.AssertNoError(list.Add(object))
	t.AssertNoError(list.Add(object))

	t.AssertEqual(1, len(factory.writers))
	t.AssertEqual(2, list.Len())

	t.AssertNoError(list.Close())

	t.AssertEqual(1, len(factory.writers))
	t.AssertEqual(1, factory.writers[0].closeCount)
}
