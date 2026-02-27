package doddish

import (
	"io"
	"unicode/utf8"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type SeqRuneScanner struct {
	Seq
	token_index int
	byte_index  int
}

func (scanner *SeqRuneScanner) Reset() {
	scanner.token_index = 0
	scanner.byte_index = 0
}

func (scanner *SeqRuneScanner) IsFull() bool {
	if scanner == nil {
		return false
	}

	return scanner.token_index == 0 && scanner.byte_index == 0
}

func (scanner *SeqRuneScanner) ReadRune() (r rune, size int, err error) {
	if scanner.token_index == scanner.Len() {
		err = io.EOF
		return r, size, err
	}

	token := scanner.At(scanner.token_index)
	contents := token.Contents[scanner.byte_index:]

	r, size = utf8.DecodeRune(contents)

	if r == utf8.RuneError && size > 0 {
		err = errors.ErrorWithStackf("invalid utf8 byte sequence: %q", contents)
		return r, size, err
	} else if r == utf8.RuneError {
		// should never happen
		err = errors.ErrorWithStackf("tried to read past end of tokens: %q", contents)
		return r, size, err
	}

	scanner.byte_index += size

	if len(contents) == size {
		scanner.byte_index = 0
		scanner.token_index += 1
	}

	return r, size, err
}

func (scanner *SeqRuneScanner) UnreadRune() (err error) {
	if scanner.token_index == 0 && scanner.byte_index == 0 {
		err = errors.ErrorWithStackf("seq rune scanner fully unread")
		return err
	}

	token := scanner.At(scanner.token_index - 1)
	bytes := token.Contents[:scanner.byte_index+1]

	_, size := utf8.DecodeLastRune(bytes)
	scanner.byte_index -= size

	if size == len(bytes) {
		scanner.token_index -= 1
		scanner.byte_index = len(scanner.At(scanner.token_index).Contents) - 1
	}

	return err
}
