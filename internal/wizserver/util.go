package wizserver

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode/utf16"
)

// BytesEqual performs a constant time comparison between slices and mirrors the
// behaviour of the original C# helper.
func BytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

// ParseChatMessage extracts a UTF-16 message from the proprietary wizard chat
// packet layout. The message is stored after a two byte length prefix followed
// by a null terminated UTF-16LE string.
func ParseChatMessage(packet []byte) (string, error) {
	if len(packet) < 4 {
		return "", errors.New("packet too short")
	}
	// Skip the leading opcode bytes and read the length from the next uint16.
	reader := bytes.NewReader(packet[2:])
	var charCount uint16
	if err := binary.Read(reader, binary.LittleEndian, &charCount); err != nil {
		return "", err
	}
	if int(charCount)*2 > reader.Len() {
		return "", errors.New("declared length exceeds packet size")
	}

	raw := make([]uint16, charCount)
	for i := 0; i < int(charCount); i++ {
		if err := binary.Read(reader, binary.LittleEndian, &raw[i]); err != nil {
			return "", err
		}
	}

	// Trim potential trailing null terminators that the client often appends.
	for len(raw) > 0 && raw[len(raw)-1] == 0 {
		raw = raw[:len(raw)-1]
	}

	return string(utf16.Decode(raw)), nil
}

// HexStringToBytes parses a string containing hexadecimal characters and
// optional spaces into a byte slice.
func HexStringToBytes(hexString string) ([]byte, error) {
	sanitized := strings.ReplaceAll(hexString, " ", "")
	if len(sanitized)%2 != 0 {
		return nil, errors.New("hex string must contain an even amount of characters")
	}
	return hex.DecodeString(sanitized)
}

// Compress applies zlib compression and matches the semantics of the SharpZip
// deflater used by the original server.
func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decompress reverses the Compress operation and returns the uncompressed data.
func Decompress(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SerializePlayer encodes the supplied GameObject into JSON and optionally
// compresses the payload when newObject is false to mimic the hybrid serialiser
// used by the legacy project.
func SerializePlayer(obj GameObject, newObject bool) ([]byte, error) {
	payload, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	if newObject {
		return payload, nil
	}

	compressed, err := Compress(payload)
	if err != nil {
		return nil, err
	}

	framed := make([]byte, 4+len(compressed))
	binary.LittleEndian.PutUint32(framed, uint32(len(payload)))
	copy(framed[4:], compressed)
	return framed, nil
}

// SerializeItem mirrors SerializePlayer but operates on arbitrary metadata.
func SerializeItem(obj map[string]any, newObject bool) ([]byte, error) {
	payload, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	if newObject {
		return payload, nil
	}

	compressed, err := Compress(payload)
	if err != nil {
		return nil, err
	}

	framed := make([]byte, 4+len(compressed))
	binary.LittleEndian.PutUint32(framed, uint32(len(payload)))
	copy(framed[4:], compressed)
	return framed, nil
}

// DefaultCharacterInfo returns a canned character definition which matches the
// behaviour of the original util helper.
func DefaultCharacterInfo() CharacterCreationInfo {
	return CharacterCreationInfo{
		Level:         1,
		SchoolOfFocus: 0x04f836b3,
		Location:      "WizardCity/WC_Ravenwood",
		TemplateID:    1,
		NameIndices:   0,
		AvatarBehavior: AvatarBehavior{
			Gender: GenderNeutral,
			Race:   RaceHuman,
		},
		Equipment: []EquippedItem{},
		GlobalID:  NextGID(),
	}
}

// CharacterCreationToGameObject converts character creation information into an
// in-game object model that is suitable for serialisation.
func CharacterCreationToGameObject(info CharacterCreationInfo) GameObject {
	metadata := map[string]any{
		"level":         info.Level,
		"schoolOfFocus": info.SchoolOfFocus,
		"nameIndices":   info.NameIndices,
		"gender":        info.AvatarBehavior.Gender,
		"race":          info.AvatarBehavior.Race,
	}
	if len(info.Equipment) > 0 {
		metadata["equipment"] = info.Equipment
	}

	return GameObject{
		GlobalID:    info.GlobalID,
		CharacterID: info.GlobalID,
		TemplateID:  GID(info.TemplateID),
		Location:    info.Location,
		Metadata:    metadata,
	}
}

// CreatePlayerInstance builds an in-game character state using the creation
// information and attaches default telemetry values that would usually come from
// the live server.
func CreatePlayerInstance(info CharacterCreationInfo) *InGameCharacter {
	inventory := Inventory{
		Backpacks: map[string]int{"cards": 0},
		Bank:      map[string]int{},
	}

	return &InGameCharacter{
		Object:          CharacterCreationToGameObject(info),
		Inventory:       inventory,
		Initialised:     true,
		X:               1132.0,
		Y:               3,
		Z:               3,
		Direction:       0,
		MarkerX:         0,
		MarkerY:         0,
		MarkerZ:         0,
		MarkerDirection: 0,
		ZoneID:          GID(123004564835992122),
		ZoneName:        "WizardCity/WC_Ravenwood",
		MarkerZoneID:    GID(123004564835992122),
		MarkerZoneName:  "WizardCity/WC_Ravenwood",
		CurrentMana:     15,
		MaxMana:         15,
		GID:             info.GlobalID,
		CharID:          info.GlobalID,
	}
}

// CreateDefaultPlayerInstance uses a new GID and default character data when no
// specific creation information has been provided.
func CreateDefaultPlayerInstance() *InGameCharacter {
	info := DefaultCharacterInfo()
	return CreatePlayerInstance(info)
}
