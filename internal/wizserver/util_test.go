package wizserver

import (
	"encoding/binary"
	"encoding/json"
	"testing"
)

func TestBytesEqual(t *testing.T) {
	a := []byte{1, 2, 3}
	b := []byte{1, 2, 3}
	c := []byte{1, 2, 4}

	if !BytesEqual(a, b) {
		t.Fatalf("expected slices to be equal")
	}
	if BytesEqual(a, c) {
		t.Fatalf("expected slices to differ")
	}
}

func TestParseChatMessage(t *testing.T) {
	header := []byte{0x10, 0x20}
	chars := []uint16{'h', 'i', 0}
	payload := make([]byte, 2+len(chars)*2)
	binary.LittleEndian.PutUint16(payload, uint16(len(chars)))
	for i, r := range chars {
		binary.LittleEndian.PutUint16(payload[2+i*2:], r)
	}

	packet := append(header, payload...)
	msg, err := ParseChatMessage(packet)
	if err != nil {
		t.Fatalf("unexpected error parsing chat message: %v", err)
	}
	if msg != "hi" {
		t.Fatalf("unexpected message contents: %q", msg)
	}
}

func TestHexStringToBytes(t *testing.T) {
	bytes, err := HexStringToBytes("DE AD BE EF")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	if !BytesEqual(bytes, expected) {
		t.Fatalf("expected %v, got %v", expected, bytes)
	}
}

func TestCompressDecompressRoundTrip(t *testing.T) {
	data := []byte("wizard101")
	compressed, err := Compress(data)
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("decompress failed: %v", err)
	}

	if !BytesEqual(data, decompressed) {
		t.Fatalf("round trip mismatch: %v vs %v", data, decompressed)
	}
}

func TestSerializePlayer(t *testing.T) {
	ResetGID()
	info := DefaultCharacterInfo()
	player := CharacterCreationToGameObject(info)

	raw, err := SerializePlayer(player, true)
	if err != nil {
		t.Fatalf("serialize new player failed: %v", err)
	}
	if len(raw) == 0 {
		t.Fatalf("expected JSON payload for new player")
	}

	framed, err := SerializePlayer(player, false)
	if err != nil {
		t.Fatalf("serialize existing player failed: %v", err)
	}
	if len(framed) <= 4 {
		t.Fatalf("expected framed payload to include header")
	}
	size := binary.LittleEndian.Uint32(framed[:4])
	decompressed, err := Decompress(framed[4:])
	if err != nil {
		t.Fatalf("failed to decompress framed payload: %v", err)
	}
	if uint32(len(decompressed)) != size {
		t.Fatalf("size header mismatch: got %d want %d", len(decompressed), size)
	}
}

func TestSerializeItem(t *testing.T) {
	payload := map[string]any{"id": 42, "name": "amulet"}

	raw, err := SerializeItem(payload, true)
	if err != nil {
		t.Fatalf("serialize new item failed: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unable to decode json payload: %v", err)
	}

	framed, err := SerializeItem(payload, false)
	if err != nil {
		t.Fatalf("serialize existing item failed: %v", err)
	}
	if len(framed) <= 4 {
		t.Fatalf("expected framed item payload to include header")
	}
}

func TestCreatePlayerInstance(t *testing.T) {
	ResetGID()
	info := DefaultCharacterInfo()
	player := CreatePlayerInstance(info)

	if !player.Initialised {
		t.Fatalf("expected player to be initialised")
	}
	if player.ZoneName == "" || player.ZoneID == 0 {
		t.Fatalf("expected zone metadata to be populated")
	}
	if player.Inventory.Backpacks["cards"] != 0 {
		t.Fatalf("unexpected inventory initialisation: %#v", player.Inventory)
	}
}
