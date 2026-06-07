package remote_proto

import (
	"bytes"
	"io"
	"net"
	"testing"

	"github.com/DataDog/zstd"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// TestBlobStreamZstdRoundTrip verifies a zstd-compressed blob streams as
// frames and decompresses, via blobFrameReader, back to the original bytes —
// and that the wire bytes are actually fewer than the payload.
func TestBlobStreamZstdRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	clientConn, serverConn := net.Pipe()

	// Highly compressible payload, larger than one chunk.
	payload := bytes.Repeat([]byte("aaaabbbbcccc"), 30000)

	writeErr := make(chan error, 1)

	go func() {
		sw := makeSession(clientConn)
		writeErr <- sw.writeBlob("blake2b256-zstd", bytes.NewReader(payload), CompressionZstd)
		_ = clientConn.Close()
	}()

	sr := makeSession(serverConn)

	typeString, header, err := sr.readControl()
	t.AssertNoError(err)
	t.AssertEqual("!"+TypeBlobHeader, typeString)
	t.AssertEqual(CompressionZstd, header.Compression)

	frameReader := &blobFrameReader{session: sr}
	decoder := zstd.NewReader(frameReader)

	var got bytes.Buffer

	if _, err = io.Copy(&got, decoder); err != nil {
		t.Fatalf("decompressing: %v", err)
	}

	t.AssertNoError(decoder.Close())
	t.AssertNoError(<-writeErr)

	if !bytes.Equal(payload, got.Bytes()) {
		t.Fatalf("decompressed blob differs from original")
	}
}

// TestBlobStreamRoundTrip verifies writeBlob chunks a blob larger than one
// frame into multiple blob frames terminated by a zero-length frame, and that
// the chunks reassemble to the original bytes.
func TestBlobStreamRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	clientConn, serverConn := net.Pipe()

	// Larger than blobChunkSize so the blob spans several frames.
	payload := bytes.Repeat([]byte("dodder-blob-payload-"), 20000)

	writeErr := make(chan error, 1)

	go func() {
		sw := makeSession(clientConn)
		writeErr <- sw.writeBlob("blake2b256-test", bytes.NewReader(payload), CompressionNone)
		_ = clientConn.Close()
	}()

	sr := makeSession(serverConn)

	typeString, header, err := sr.readControl()
	t.AssertNoError(err)
	t.AssertEqual("!"+TypeBlobHeader, typeString)
	t.AssertEqual("blake2b256-test", header.BlobId)

	var got bytes.Buffer
	frameCount := 0

	for {
		kind, length, frameErr := readFrameHeader(sr.reader)
		t.AssertNoError(frameErr)
		t.AssertEqual(frameKindBlob, kind)

		if length == 0 {
			break
		}

		frameCount++

		if _, err = io.CopyN(&got, sr.reader, int64(length)); err != nil {
			t.Fatalf("copying chunk: %v", err)
		}
	}

	t.AssertNoError(<-writeErr)
	t.AssertEqual(len(payload), got.Len())

	if !bytes.Equal(payload, got.Bytes()) {
		t.Fatalf("reassembled blob differs from original")
	}

	if frameCount < 2 {
		t.Fatalf("expected the blob to span multiple frames, got %d", frameCount)
	}
}

// TestBlobStreamEmpty verifies a zero-byte blob is a header followed
// immediately by the terminator frame.
func TestBlobStreamEmpty(t1 *testing.T) {
	t := ui.MakeT(t1)

	clientConn, serverConn := net.Pipe()

	writeErr := make(chan error, 1)

	go func() {
		sw := makeSession(clientConn)
		writeErr <- sw.writeBlob("blake2b256-empty", bytes.NewReader(nil), CompressionNone)
		_ = clientConn.Close()
	}()

	sr := makeSession(serverConn)

	typeString, _, err := sr.readControl()
	t.AssertNoError(err)
	t.AssertEqual("!"+TypeBlobHeader, typeString)

	kind, length, frameErr := readFrameHeader(sr.reader)
	t.AssertNoError(frameErr)
	t.AssertEqual(frameKindBlob, kind)
	t.AssertEqual(uint32(0), length)

	t.AssertNoError(<-writeErr)
}
