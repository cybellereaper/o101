package wizserver

import "sync/atomic"

// GID uniquely identifies game objects within the realm.
type GID uint64

var nextGID uint64

// NextGID returns a monotonically increasing identifier that mimics the
// behaviour of the original server which incremented a global counter.
func NextGID() GID {
	id := atomic.AddUint64(&nextGID, 1)
	return GID(id)
}

// ResetGID is exposed for tests that need determinism.
func ResetGID() {
	atomic.StoreUint64(&nextGID, 0)
}

// AvatarGender represents the gender flag used by the character creator.
type AvatarGender string

const (
	GenderNeutral AvatarGender = "neutral"
	GenderMale    AvatarGender = "male"
	GenderFemale  AvatarGender = "female"
)

// AvatarRace represents the character race flag used by the avatar behaviour.
type AvatarRace string

const (
	RaceHuman AvatarRace = "human"
	RaceBeast AvatarRace = "beast"
)

// CharacterCreationInfo contains the data returned by the character creation
// service. Only the subset that is required by the game bootstrap flow is
// modelled here.
type CharacterCreationInfo struct {
	Level          int            `json:"level"`
	SchoolOfFocus  uint32         `json:"schoolOfFocus"`
	Location       string         `json:"location"`
	TemplateID     uint64         `json:"templateId"`
	NameIndices    int            `json:"nameIndices"`
	AvatarBehavior AvatarBehavior `json:"avatarBehavior"`
	Equipment      []EquippedItem `json:"equipment"`
	GlobalID       GID            `json:"globalId"`
	UserID         GID            `json:"userId"`
}

// AvatarBehavior describes cosmetic properties for the avatar.
type AvatarBehavior struct {
	Gender AvatarGender `json:"gender"`
	Race   AvatarRace   `json:"race"`
}

// EquippedItem represents a publicly visible item worn by the character.
type EquippedItem struct {
	Template string `json:"template"`
	Dye      string `json:"dye"`
}

// LoginPlayer bundles together the information required by the login service.
type LoginPlayer struct {
	Characters []CharacterCreationInfo
	UserID     GID
}

// Inventory represents the player's item storage. Only a subset of the
// behaviour is required for replication tests and serialisation.
type Inventory struct {
	Backpacks map[string]int `json:"backpacks"`
	Bank      map[string]int `json:"bank"`
}

// GameObject is a simplified version of the WizClientObject used by the
// original project. It carries enough information to bootstrap a character
// into a zone and to serialise/deserialise its state for persistence.
type GameObject struct {
	GlobalID    GID            `json:"globalId"`
	CharacterID GID            `json:"characterId"`
	TemplateID  GID            `json:"templateId"`
	Location    string         `json:"location"`
	Metadata    map[string]any `json:"metadata"`
}

// InGameCharacter represents the state for an avatar that has transitioned
// into a game zone.
type InGameCharacter struct {
	Object          GameObject
	Inventory       Inventory
	Initialised     bool
	X, Y, Z         float32
	Direction       float32
	MarkerX         float32
	MarkerY         float32
	MarkerZ         float32
	MarkerDirection float32
	ZoneName        string
	ZoneID          GID
	MarkerZoneName  string
	MarkerZoneID    GID
	CharID          GID
	GID             GID
	MobileID        uint16
	CurrentMana     int
	MaxMana         int
	Socket          Socket
}

// Message represents a packet that can be delivered to a socket managed by the
// realm. The concrete type is intentionally left open ended to keep the
// implementation protocol agnostic.
type Message interface{}

// Socket abstracts the minimal functionality required by zones and the realm.
// Tests provide in-memory implementations.
type Socket interface {
	Send(msgs ...Message) error
}
