package doddish

import (
	"bytes"
	"io"
	"unicode"
	"unicode/utf8"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type Scanner struct {
	RuneScanner io.RuneScanner

	tokenTypeProbably TokenType

	scanned       bytes.Buffer
	scannedOffset int
	seq           Seq

	err      error // TODO switch to pkgError
	unscan   *SeqRuneScanner
	n        int64
	lastRune rune
}

func MakeScanner(runeScanner io.RuneScanner) *Scanner {
	var scanner Scanner
	scanner.Reset(runeScanner)
	return &scanner
}

func (scanner *Scanner) Reset(runeScanner io.RuneScanner) {
	scanner.RuneScanner = runeScanner
	scanner.scanned.Reset()
	scanner.scannedOffset = 0
	scanner.tokenTypeProbably = TokenTypeIncomplete
	scanner.seq.Reset()
	scanner.err = nil
	scanner.unscan = nil
	scanner.n = 0
}

func (scanner *Scanner) ReadRune() (char rune, n int, err error) {
	if scanner.unscan != nil {
		char, n, err = scanner.unscan.ReadRune()

		if err == io.EOF {
			scanner.unscan = nil
			// pass
		} else if err != nil {
			err = errors.Wrap(err)
			return char, n, err
		} else {
			return char, n, err
		}
	}

	scanner.lastRune, n, err = scanner.RuneScanner.ReadRune()
	scanner.n += int64(n)

	return scanner.lastRune, n, err
}

// TODO remove unread entirely
func (scanner *Scanner) UnreadRune() (err error) {
	if scanner.unscan != nil {
		if err = scanner.unscan.UnreadRune(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}

	err = scanner.RuneScanner.UnreadRune()

	if err == nil {
		scanner.n -= int64(utf8.RuneLen(scanner.lastRune))
	}

	return err
}

func (scanner *Scanner) Unscan() {
	scanner.unscan = &SeqRuneScanner{Seq: scanner.seq}
}

func (scanner *Scanner) CanScan() (ok bool) {
	return scanner.unscan != nil || scanner.err == nil
}

func (scanner *Scanner) resetBeforeNextScan() {
	scanner.scanned.Reset()
	scanner.scannedOffset = 0
	scanner.tokenTypeProbably = TokenTypeIncomplete
	scanner.seq.Reset()
}

func (scanner *Scanner) ScanSkipSpace() (ok bool) {
	if !scanner.ConsumeSpacesOrErrorOnFalse() {
		return ok
	}

	ok = scanner.Scan()

	return ok
}

// TODO create sequence methods for Scan family
func (scanner *Scanner) Scan() (ok bool) {
	return scanner.scan(true)
}

func (scanner *Scanner) ScanDotAllowedInIdentifiers() (ok bool) {
	return scanner.scan(false)
}

func (scanner *Scanner) ScanDotAllowedInIdentifiersOrError() (Seq, Error) {
	if !scanner.ScanDotAllowedInIdentifiers() {
		return nil, ErrEmptySeq
	}

	if scanner.CanScan() {
		return nil, ErrMoreThanOneSeq
	}

	return scanner.GetSeq(), nil
}

func (scanner *Scanner) appendTokenWithTypeToSeq(tokenType TokenType) {
	if bites := scanner.scanned.Bytes()[scanner.scannedOffset:]; len(bites) > 0 {
		scanner.seq.Add(tokenType, bites)
		scanner.scannedOffset += len(bites)
	}
}

func (scanner *Scanner) scan(dotOperatorAsSplit bool) (hasSeq bool) {
	if scanner.unscan.IsFull() {
		hasSeq = true
		scanner.unscan = nil
		return hasSeq
	}

	if scanner.err == io.EOF {
		return hasSeq
	}

	afterFirst := false
	hasSeq = true

	scanner.resetBeforeNextScan()

	for {
		var char rune

		char, _, scanner.err = scanner.ReadRune()

		if scanner.err != nil {
			if scanner.err == io.EOF {
				hasSeq = scanner.scanned.Len() > 0
				scanner.appendTokenWithTypeToSeq(scanner.tokenTypeProbably)
			}

			return hasSeq
		}

		op, _ := MakeOp(char)
		isSoloOrPrefix := op.isSoloSeqOp(!dotOperatorAsSplit)
		isMixedOp := op.isMixedSeqOp()
		isInlineOp := op.isInlineOp()

		switch {
		// TODO make quotes and backslash operators
		case char == '"' || char == '\'':
			if !scanner.consumeLiteralOrFieldValue(
				char,
				TokenTypeLiteral,
			) {
				hasSeq = false
				return hasSeq
			}

			return hasSeq

		case !afterFirst && isSoloOrPrefix:
			scanner.scanned.WriteRune(char)
			scanner.appendTokenWithTypeToSeq(TokenTypeOperator)

			if op.isSpace() {
				if !scanner.ConsumeSpacesOrErrorOnFalse() {
					hasSeq = false
					return hasSeq
				}
			}

			return hasSeq

		case (!isSoloOrPrefix && !isMixedOp) || (isInlineOp && afterFirst):
			scanner.tokenTypeProbably = TokenTypeIdentifier
			scanner.scanned.WriteRune(char)
			afterFirst = true
			continue

		case isMixedOp:
			scanner.appendTokenWithTypeToSeq(scanner.tokenTypeProbably)
			scanner.scanned.WriteRune(char)
			scanner.appendTokenWithTypeToSeq(TokenTypeOperator)
			afterFirst = true
			continue

		default: // isOperator && afterFirst
			scanner.appendTokenWithTypeToSeq(scanner.tokenTypeProbably)

			if char == '=' {
				if !scanner.consumeField(char) {
					hasSeq = false
					return hasSeq
				}

				return hasSeq
			}

			if scanner.err = scanner.UnreadRune(); scanner.err != nil {
				scanner.err = errors.Wrapf(scanner.err, "%c", char)
				hasSeq = false
			}

			return hasSeq
		}
	}
}

// Consumes any spaces currently available in the underlying RuneReader. If this
// returns false, it means that a read error has occurred, not that no spaces
// were consumed.
func (scanner *Scanner) ConsumeSpacesOrErrorOnFalse() (ok bool) {
	ok = true

	for {
		var r rune

		r, _, scanner.err = scanner.ReadRune()

		if scanner.err != nil {
			ok = false
			return ok
		}

		if unicode.IsSpace(r) {
			continue
		}

		if scanner.err = scanner.UnreadRune(); scanner.err != nil {
			ok = false
			scanner.err = errors.Wrapf(scanner.err, "%c", r)
		}

		return ok
	}
}

// TODO add support for ellipis
func (scanner *Scanner) consumeLiteralOrFieldValue(
	start rune,
	tokenType TokenType,
) (ok bool) {
	ok = true

	lastWasBackslash := false

	for {
		var char rune

		char, _, scanner.err = scanner.ReadRune()

		if scanner.err != nil {
			ok = false
			return ok
		}

		currentIsBackslash := char == '\\'
		escaped := lastWasBackslash && !currentIsBackslash
		end := char == start
		content := !lastWasBackslash && !currentIsBackslash && !end

		if escaped || content {
			scanner.scanned.WriteRune(char)
		}

		if char != start || lastWasBackslash {
			lastWasBackslash = currentIsBackslash
			continue
		}

		scanner.appendTokenWithTypeToSeq(tokenType)

		return ok
	}
}

func (scanner *Scanner) consumeField(start rune) bool {
	scanner.scanned.WriteRune(start)
	ok := scanner.consumeIdentifierLike(TokenTypeLiteral)
	return ok
}

// TODO add support for ellipsis
func (scanner *Scanner) consumeIdentifierLike(
	tokenType TokenType,
) (ok bool) {
	ok = true

	idx := scanner.scanned.Len()

	for {
		var char rune

		char, _, scanner.err = scanner.ReadRune()

		if scanner.err != nil {
			if scanner.err == io.EOF {
				ok = scanner.scanned.Len() > 0
			}

			return ok
		}

		op, _ := MakeOp(char)
		isOperator := op.isSoloSeqOp(true)

		switch {
		case char == '"' || char == '\'':
			if !scanner.consumeLiteralOrFieldValue(char, tokenType) {
				ok = false
				return ok
			}

			return ok

		case !isOperator:
			scanner.scanned.WriteRune(char)
			continue

		default: // wasSplitRune && afterFirst
			scanner.seq.Add(
				tokenType,
				scanner.scanned.Bytes()[idx:scanner.scanned.Len()],
			)

			if scanner.err = scanner.UnreadRune(); scanner.err != nil {
				scanner.err = errors.Wrapf(scanner.err, "%c", char)
				ok = false
			}

			return ok
		}
	}
}

// Valid only until the next call to any scan method. To keep the sequence, make
// a clone of it by calling Seq.Clone()
func (scanner *Scanner) GetSeq() Seq {
	return scanner.seq
}

func (scanner *Scanner) N() int64 {
	return scanner.n
}

func (scanner *Scanner) Error() error {
	if scanner.err == io.EOF {
		return nil
	}

	return scanner.err
}
