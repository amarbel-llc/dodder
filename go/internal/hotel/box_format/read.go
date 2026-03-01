package box_format

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/doddish"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

// TODO make this error for invalid input
func (format *BoxTransacted) ReadStringFormat(
	object *sku.Transacted,
	runeScanner io.RuneScanner,
) (n int64, err error) {
	scanner := doddish.MakeScanner(runeScanner)

	if err = format.readStringFormatBox(scanner, object); err != nil {
		if err == ErrNotABox {
			err = nil
		} else {
			err = errors.WrapExceptSentinel(err, io.EOF, ErrBoxReadSeq{})
			return n, err
		}
	}

	if scanner.Error() != nil {
		err = errors.Wrap(scanner.Error())
		return n, err
	}

	n = scanner.N()

	if format.optionsPrint.BoxDescriptionInBox {
		return n, err
	}

	// TODO extract into dedicated parser and make incompatible with
	// BoxTransactedWithSignature
	if err = object.GetMetadataMutable().GetDescriptionMutable().ReadFromBoxScanner(
		scanner,
	); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n = scanner.N()

	return n, err
}

func (format *BoxTransacted) openBox(
	scanner *doddish.Scanner,
) (err error) {
	if !scanner.ScanSkipSpace() {
		if scanner.Error() != nil {
			err = errors.Wrap(scanner.Error())
		} else {
			err = io.EOF
		}

		return err
	}

	seq := scanner.GetSeq()

	if !seq.MatchAll(doddish.TokenMatcherOp(doddish.OpGroupOpen)) {
		err = ErrNotABox
		scanner.Unscan()

		return err
	}

	if !scanner.ConsumeSpacesOrErrorOnFalse() {
		if scanner.Error() != nil {
			err = errors.Wrap(scanner.Error())
		} else {
			err = io.ErrUnexpectedEOF
		}

		return err
	}

	return err
}

// TODO switch to returning ErrBoxParse
func (format *BoxTransacted) readStringFormatBox(
	scanner *doddish.Scanner,
	object *sku.Transacted,
) (err error) {
	if err = format.openBox(scanner); err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF, ErrNotABox)
		return err
	}

	{
		if !scanner.ScanDotAllowedInIdentifiers() {
			if scanner.Error() != nil {
				err = errors.Wrap(scanner.Error())
			} else {
				err = io.ErrUnexpectedEOF
			}

			return err
		}

		seq := scanner.GetSeq()

		if err = object.GetObjectIdMutable().SetWithSeq(seq); err != nil {
			err = nil
			object.GetObjectIdMutable().Reset()

			// "some_file.ext"
			if seq.MatchAll(doddish.TokenTypeLiteral) {
				if err = object.GetExternalObjectIdMutable().Set(seq.String()); err != nil {
					err = errors.Wrap(err)
					return err
				}

				// /some_file.ext
			} else if ok, left, _, _ := seq.PartitionFavoringLeft(
				doddish.TokenMatcherOp(doddish.OpPathSeparator),
			); ok && left.Len() == 0 {
				if err = object.GetExternalObjectIdMutable().Set(seq.String()); err != nil {
					err = errors.Wrap(err)
					return err
				}

				// maybe one/uno.zettel or some_dir/some_name.ext
			} else if ok, left, right := seq.MatchEnd(
				doddish.TokenMatcherOp(doddish.OpSigilExternal),
				doddish.TokenTypeIdentifier,
			); ok {
				var genre genres.Genre

				// left: one/uno, right: .zettel
				if err = genre.Set(right.At(1).String()); err == nil {
					if err = ids.SetWithGenre(
						object.GetObjectIdMutable(),
						left.String(),
						genre,
					); err != nil {
						object.GetObjectIdMutable().Reset()
						err = errors.Wrap(err)
						return err
					}

					// left: some_dir/some_name, right: .ext
				} else {
					if err = object.GetExternalObjectIdMutable().Set(seq.String()); err != nil {
						err = errors.Wrap(err)
						return err
					}
				}

			} else {
				err = errors.ErrorWithStackf("unsupported seq: %q", seq)
				return err
			}
		}

		if object.GetObjectId().GetGenre() == genres.InventoryList {
			if err = object.GetMetadataMutable().GetTaiMutable().Set(
				object.GetObjectId().String(),
			); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}
	}

LOOP_AFTER_OBJECT_ID:

	for scanner.Scan() {
		seq := scanner.GetSeq()

		// TODO convert this into a decision tree based on token type sequences
		// instead of a switch
		switch {
		case seq.MatchAll(doddish.TokenMatcherOp(']')):
			break LOOP_AFTER_OBJECT_ID

			// ' '
		case seq.MatchAll(doddish.TokenMatcherOp(' ')):
			continue

			// "value"
		case seq.MatchAll(doddish.TokenTypeLiteral):
			if err = object.GetMetadataMutable().GetDescriptionMutable().Set(
				seq.String(),
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			// @digest
		case seq.MatchAll(doddish.TokenMatcherBlobDigest...):
			if err = format.parseMarklIdTag(object, seq); err != nil {
				err = errors.Wrap(err)
				return err
			}

			// !type
		case seq.MatchAll(doddish.TokenMatcherType...):
			if err = object.GetMetadataMutable().GetTypeMutable().SetType(
				seq.String(),
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			// !type@sig
		case seq.MatchAll(doddish.TokenMatcherTypeLock...):
			typeLock := object.GetMetadataMutable().GetTypeLockMutable()

			if err = typeLock.GetKeyMutable().Set(
				seq[0:2].String(),
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			if err = markl.SetMarklIdWithFormatBlech32(
				typeLock.GetValueMutable(),
				"",
				seq[2:].String(),
			); err != nil {
				err = errors.Wrapf(err, "Seq: %q", seq)
				return err
			}

			// key@abcd
		case seq.MatchAll(doddish.TokenMatcherDodderTag...):
			if err = format.parseMarklIdTag(object, seq); err != nil {
				err = errors.Wrap(err)
				return err
			}

			// key=value key="value"
		case seq.MatchStart(doddish.TokenMatcherKeyValue...) || seq.MatchStart(doddish.TokenMatcherKeyValueLiteral...):
			value := seq[2:]

			field := string_format_writer.Field{
				Key:   string(seq.At(0).Contents),
				Value: value.String(),
			}

			field.ColorType = string_format_writer.ColorTypeUserData
			object.GetMetadataMutable().GetIndexMutable().GetFieldsMutable().Append(field)

		case seq.MatchAll(
			doddish.TokenMatcherOp(doddish.OpTagSeparator),
			doddish.TokenTypeIdentifier,
		):

			fallthrough

			// value
		case seq.MatchAll(doddish.TokenTypeIdentifier):
			var tag ids.TagStruct

			if err = tag.Set(seq.String()); err != nil {
				err = errors.Wrap(err)
				return err
			}

			if tag.IsDodderTag() {
				err = errors.Err405MethodNotAllowed.Errorf("tag: %q", tag)
				return err
			} else {
				if err = object.AddTag(tag); err != nil {
					err = errors.Wrap(err)
					return err
				}
			}

			// value.value
		case seq.MatchAll(doddish.TokenMatcherTai...):
			if err = object.GetMetadataMutable().GetTaiMutable().Set(
				seq.String(),
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

		default:
			err = doddish.ErrUnsupportedSeq{
				Seq: seq,
			}

			scanner.Unscan()

			return err
		}
	}

	if scanner.Error() != nil {
		err = errors.Wrap(scanner.Error())
		return err
	}

	return err
}

func (format *BoxTransacted) parseMarklIdTag(
	object *sku.Transacted,
	seq doddish.Seq,
) (err error) {
	var purposeId string
	var value []byte

	if seq.Len() == 3 {
		purposeId = string(seq.At(0).Contents)
		value = seq.At(2).Contents
	} else {
		// blobs have only the `@` symbol and no purpose, so we implicitly decide
		// the purpose when parsing
		purposeId = markl.PurposeBlobDigestV1
		value = seq.At(1).Contents
	}

	id := objects.GetMarklIdMutableForPurpose(
		object.GetMetadataMutable(),
		purposeId,
	)

	if err = markl.SetMarklIdWithFormatBlech32(
		id,
		purposeId,
		string(value),
	); err != nil {
		err = errors.Wrapf(err, "Seq: %q", seq)
		return err
	}

	return err
}
