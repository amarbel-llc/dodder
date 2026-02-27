package markl

import (
	"crypto/ed25519"
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"golang.org/x/crypto/curve25519"
)

// actual formats
const (
	// keep sorted
	FormatIdEd25519Pub = "ed25519_pub"
	FormatIdEd25519SSH = "ed25519_ssh"
	FormatIdEd25519Sec = "ed25519_sec"
	FormatIdEd25519Sig = "ed25519_sig"

	FormatIdAgeX25519Pub = "age_x25519_pub"
	FormatIdAgeX25519Sec = "age_x25519_sec"

	FormatIdPivyEcdhP256Pub = "pivy_ecdh_p256_pub"

	FormatIdHashSha256     = "sha256"
	FormatIdHashBlake2b256 = "blake2b256"

	FormatIdNonceSec = "nonce"
)

func init() {
	// Ed22519
	makeFormat(
		FormatPub{
			Id:     FormatIdEd25519Pub,
			Size:   ed25519.PublicKeySize,
			Verify: Ed25519Verify,
		},
	)

	makeFormat(
		FormatSec{
			Id:   FormatIdEd25519Sec,
			Size: ed25519.PrivateKeySize,

			Generate: Ed25519GeneratePrivateKey,

			PubFormatId:  FormatIdEd25519Pub,
			GetPublicKey: Ed25519GetPublicKey,

			SigFormatId: FormatIdEd25519Sig,
			Sign:        Ed25519Sign,
		},
	)

	makeFormat(
		Format{
			Id:   FormatIdEd25519Sig,
			Size: ed25519.SignatureSize,
		},
	)

	makeStubSSHFormat()

	// AgeX25519
	makeFormat(
		Format{
			Id:   FormatIdAgeX25519Pub,
			Size: curve25519.ScalarSize,
		},
	)

	makeFormat(
		FormatSec{
			Id:           FormatIdAgeX25519Sec,
			Size:         curve25519.ScalarSize,
			Generate:     AgeX25519Generate,
			GetIOWrapper: AgeX25519GetIOWrapper,
		},
	)

	// PivyEcdhP256
	makeFormat(
		FormatSec{
			Id:           FormatIdPivyEcdhP256Pub,
			Size:         33,
			GetIOWrapper: PivyEcdhP256GetIOWrapper,
		},
	)

	// Nonce
	makeFormat(
		FormatSec{
			Id:       FormatIdNonceSec,
			Size:     32,
			Generate: NonceGenerate32,
		},
	)
}

var formats map[string]domain_interfaces.MarklFormat = map[string]domain_interfaces.MarklFormat{}

func GetFormatOrError(formatId string) (domain_interfaces.MarklFormat, error) {
	switch formatId {
	case "zit-repo-private_key-v1", "dodder-repo-private_key-v1":
		formatId = FormatIdEd25519Sec
	}

	format, ok := formats[formatId]

	if !ok {
		err := errors.Errorf("unknown format id: %q", formatId)
		return nil, err
	}

	return format, nil
}

// move to Id
func GetFormatSecOrError(
	formatIdGetter domain_interfaces.MarklFormatGetter,
) (formatSec FormatSec, err error) {
	format := formatIdGetter.GetMarklFormat()

	if format == nil {
		err = errors.Errorf("empty format for getter: %s", formatIdGetter)
		return formatSec, err
	}

	formatId := formatIdGetter.GetMarklFormat().GetMarklFormatId()

	if format, err = GetFormatOrError(formatId); err != nil {
		err = errors.Wrap(err)
		return formatSec, err
	}

	var ok bool

	if formatSec, ok = format.(FormatSec); !ok {
		err = errors.Errorf(
			"requested format is not FormatSec, but %T:%s",
			formatSec,
			formatId,
		)
		return formatSec, err
	}

	return formatSec, err
}

type FormatId string

func (formatId FormatId) GetMarklFormat() domain_interfaces.MarklFormat {
	format, err := GetFormatOrError(string(formatId))
	errors.PanicIfError(err)
	return format
}

type Format struct {
	Id   string
	Size int
}

var _ domain_interfaces.MarklFormat = Format{}

func (format Format) GetMarklFormatId() string {
	return format.Id
}

func (format Format) GetSize() int {
	return format.Size
}

func makeFormat(format domain_interfaces.MarklFormat) {
	if format == nil {
		panic("nil format")
	}

	formatId := format.GetMarklFormatId()

	if formatId == "" {
		panic("empty formatId")
	}

	existing, alreadyExists := formats[formatId]

	if alreadyExists {
		panic(
			fmt.Sprintf(
				"format already registered: %q (%T)",
				formatId,
				existing,
			),
		)
	}

	formats[formatId] = format
}
