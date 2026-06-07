package remote_proto

import (
	"bytes"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestControlRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	original := control{
		ProtocolVersion:   ProtocolVersion,
		Role:              RoleServer,
		WebSocket:         true,
		InventoryListType: "inventory_list-v2",
		ExpandEdges:       true,
		PublicKey:         "some-public-key",
		Nonce:             "some-nonce",
	}

	encoded, err := encodeControl(TypeCapabilities, original)
	t.AssertNoError(err)

	// The encoded payload must be a typed hyphence document carrying the
	// type in its metadata line.
	if !bytes.Contains(encoded, []byte("! "+TypeCapabilities)) {
		t.Fatalf("encoded control missing type line: %q", encoded)
	}

	typeString, decoded, err := decodeControl(encoded)
	t.AssertNoError(err)
	t.AssertEqual("!"+TypeCapabilities, typeString)
	t.AssertEqual(original.ProtocolVersion, decoded.ProtocolVersion)
	t.AssertEqual(original.Role, decoded.Role)
	t.AssertEqual(original.WebSocket, decoded.WebSocket)
	t.AssertEqual(original.InventoryListType, decoded.InventoryListType)
	t.AssertEqual(original.PublicKey, decoded.PublicKey)
	t.AssertEqual(original.Nonce, decoded.Nonce)
}

func TestControlBareAck(t1 *testing.T) {
	t := ui.MakeT(t1)

	encoded, err := encodeControl(TypeAck, control{Status: StatusComplete})
	t.AssertNoError(err)

	typeString, decoded, err := decodeControl(encoded)
	t.AssertNoError(err)
	t.AssertEqual("!"+TypeAck, typeString)
	t.AssertEqual(StatusComplete, decoded.Status)
}

func TestFrameRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	var buffer bytes.Buffer

	t.AssertNoError(
		writeControlFrame(&buffer, TypeWant, control{
			Direction: DirectionFetch,
			Query:     ":z",
		}),
	)

	payload := []byte("raw blob bytes")
	t.AssertNoError(writeFrame(&buffer, frameKindBlob, payload))

	// First frame: the want control.
	kind, length, err := readFrameHeader(&buffer)
	t.AssertNoError(err)
	t.AssertEqual(frameKindControl, kind)

	got, err := readFramePayload(&buffer, length)
	t.AssertNoError(err)

	typeString, msg, err := decodeControl(got)
	t.AssertNoError(err)
	t.AssertEqual("!"+TypeWant, typeString)
	t.AssertEqual(DirectionFetch, msg.Direction)
	t.AssertEqual(":z", msg.Query)

	// Second frame: the raw blob.
	kind, length, err = readFrameHeader(&buffer)
	t.AssertNoError(err)
	t.AssertEqual(frameKindBlob, kind)

	got, err = readFramePayload(&buffer, length)
	t.AssertNoError(err)
	t.AssertEqual(string(payload), string(got))
}
