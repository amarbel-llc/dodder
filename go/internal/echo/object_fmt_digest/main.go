// framework for generating digests from objects according to specific formats
package object_fmt_digest

import (
	"fmt"
	"os"
	"strings"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/0/key_strings"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/lib/bravo/catgut"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

type (
	FormatterContext interface {
		objects.EncoderContext
		GetObjectId() *ids.ObjectId
	}
)

type keyType = *catgut.String

func GetFormatForPurpose(
	purpose string,
) (format format) {
	var found bool

	if format, found = formatsMap[purpose]; !found {
		panic(errUnknownFormatKey(purpose))
	}

	return format
}

func FormatForPurposeOrError(
	purpose string,
) (format format, err error) {
	var found bool
	if format, found = formatsMap[purpose]; !found {
		err = errUnknownFormatKey(purpose)
		return format, err
	}

	return format, err
}

var (
	formatsList = []format{}
	formatsMap  = map[string]format{}
)

func registerFormat(purpose string, keys ...keyType) {
	format, alreadyExists := formatsMap[purpose]

	if alreadyExists {
		panic(
			fmt.Sprintf(
				"format for purpose %q already registered",
				purpose,
			),
		)
	}

	format.purpose = purpose
	format.keys = keys

	formatsList = append(formatsList, format)
	formatsMap[purpose] = format
}

// TODO register these formats with a hash of the code used to generate them to
// make them immutable. Maybe implement as plugins.
func init() {
	// TODO replace with modern keys
	registerFormat(
		markl.PurposeV5MetadataDigestWithoutTai,
		key_strings.Blob,
		key_strings.Description,
		key_strings.Tag,
		key_strings.Type,
	)

	registerFormat(
		markl.PurposeObjectDigestV1,
		key_strings.Blob,
		key_strings.Description,
		key_strings.ObjectId,
		key_strings.Tag,
		key_strings.Tai,
		key_strings.Type,
		key_strings.ZZRepoPub,
		key_strings.ZZSigMother,
	)

	registerFormat(
		markl.PurposeObjectDigestV2,
		key_strings.Blob,
		key_strings.Description,
		key_strings.ObjectId,
		key_strings.Tag,
		key_strings.Tai,
		key_strings.TypeLock,
		key_strings.ZZRepoPub,
		key_strings.ZZSigMother,
	)

	// V3 = V2 + typed blob-reference coverage: each reference's id, type
	// lock (including its lock value/sig), and alias become part of the
	// signed object digest.
	registerFormat(
		markl.PurposeObjectDigestV3,
		key_strings.Blob,
		key_strings.BlobReference,
		key_strings.Description,
		key_strings.ObjectId,
		key_strings.Tag,
		key_strings.Tai,
		key_strings.TypeLock,
		key_strings.ZZRepoPub,
		key_strings.ZZSigMother,
	)
}

func WriteDigest(
	formatId string,
	context FormatterContext,
	output mad_domain_interfaces.MarklIdMutable,
) (err error) {
	format := GetFormatForPurpose(formatId)

	metadata := context.GetMetadata()

	if metadata.GetTai().IsEmpty() {
		err = ErrEmptyTai
		return err
	}

	var digest mad_domain_interfaces.MarklId

	// TEMPORARY debug instrumentation for investigating the pull
	// ed25519-verification bug. DODDER_DEBUG_DIGEST_ALL=1 logs every
	// object-id WriteDigest processes (to find the real object-id string
	// format at this call site); DODDER_DEBUG_DIGEST_OBJECT_ID=<substr>
	// additionally dumps the exact bytes hashed when the object-id
	// CONTAINS that substring (substring match, not exact, since the
	// object-id format at this call site is not yet confirmed). No-op
	// unless either env var is set. Not meant to be committed long-term --
	// remove once the bug is root-caused.
	debugAll := os.Getenv("DODDER_DEBUG_DIGEST_ALL") != ""
	debugObjectIdSubstr := os.Getenv("DODDER_DEBUG_DIGEST_OBJECT_ID")
	objectIdString := context.GetObjectId().String()

	if debugAll {
		fmt.Fprintf(
			os.Stderr,
			"DODDER_DEBUG_DIGEST_ALL: purpose=%q object_id=%q\n",
			formatId,
			objectIdString,
		)
	}

	if debugObjectIdSubstr != "" && strings.Contains(objectIdString, debugObjectIdSubstr) {
		var sb strings.Builder

		if digest, err = format.writeMetadata(&sb, context); err != nil {
			err = errors.Wrap(err)
			return err
		}

		fmt.Fprintf(
			os.Stderr,
			"DODDER_DEBUG_DIGEST: purpose=%q object_id=%q digest=%s\nbytes=%q\n",
			formatId,
			objectIdString,
			digest,
			sb.String(),
		)
	} else if digest, err = format.writeMetadata(nil, context); err != nil {
		err = errors.Wrap(err)
		return err
	}

	output.ResetWithMarklId(digest)

	if err = output.SetPurposeId(format.purpose); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = markl.AssertIdIsNotNull(
		output,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
