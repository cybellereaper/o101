package wizserver

import (
	"errors"
	"sort"
	"sync"
)

// Realm orchestrates zones and manages global caches for character creation
// information. The implementation is a concurrency safe translation of the
// behaviour in the reference C# server.
type Realm struct {
	mu                 sync.RWMutex
	maxPlayers         uint32
	zones              []*Zone
	characterCache     map[GID]CharacterCreationInfo
	userCharacterIndex map[GID][]GID
}

// NewRealm creates a realm populated with the requested number of zones.
func NewRealm(maxPlayers uint32, zoneCount uint32, maxPlayersPerZone uint32) *Realm {
	zones := make([]*Zone, 0, zoneCount)
	for i := uint32(0); i < zoneCount; i++ {
		zones = append(zones, NewZone(maxPlayersPerZone))
	}

	return &Realm{
		maxPlayers:         maxPlayers,
		zones:              zones,
		characterCache:     make(map[GID]CharacterCreationInfo),
		userCharacterIndex: make(map[GID][]GID),
	}
}

// AssignZone finds a zone with available capacity and adds the supplied player.
// Zones are traversed in deterministic order to keep testing predictable.
func (r *Realm) AssignZone(player *InGameCharacter) error {
	r.mu.RLock()
	zones := make([]*Zone, len(r.zones))
	copy(zones, r.zones)
	r.mu.RUnlock()

	sort.SliceStable(zones, func(i, j int) bool { return i < j })
	for _, zone := range zones {
		if zone.PlayerCount() >= zone.Capacity() {
			continue
		}
		if err := zone.AddPlayer(player); err == nil {
			return nil
		}
	}
	return errors.New("no zone capacity available")
}

// Broadcast sends the provided message to every zone in the realm.
func (r *Realm) Broadcast(src GID, msg Message) {
	r.mu.RLock()
	zones := make([]*Zone, len(r.zones))
	copy(zones, r.zones)
	r.mu.RUnlock()

	for _, zone := range zones {
		zone.Broadcast(src, msg)
	}
}

// CacheCharacterCreationInfo stores the character information keyed by the
// character ID and indexes it for the owning user.
func (r *Realm) CacheCharacterCreationInfo(info CharacterCreationInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.characterCache[info.GlobalID] = info
	r.userCharacterIndex[info.UserID] = append(r.userCharacterIndex[info.UserID], info.GlobalID)
}

// CharacterFromCache retrieves character data if it has been cached previously.
func (r *Realm) CharacterFromCache(id GID) (CharacterCreationInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, ok := r.characterCache[id]
	return info, ok
}

// CharactersForPlayer returns cached characters for the supplied player. When
// no information is available, a default character is synthesised.
func (r *Realm) CharactersForPlayer(playerID GID) []CharacterCreationInfo {
	r.mu.RLock()
	ids := append([]GID(nil), r.userCharacterIndex[playerID]...)
	r.mu.RUnlock()

	if len(ids) == 0 {
		info := DefaultCharacterInfo()
		info.UserID = playerID
		info.GlobalID = NextGID()
		return []CharacterCreationInfo{info}
	}

	characters := make([]CharacterCreationInfo, 0, len(ids))
	for _, id := range ids {
		if info, ok := r.CharacterFromCache(id); ok {
			info.UserID = playerID
			characters = append(characters, info)
		}
	}
	return characters
}

// Zones exposes a copy of the zone slice to allow inspection in tests without
// risking accidental mutations of the internal slice.
func (r *Realm) Zones() []*Zone {
	r.mu.RLock()
	defer r.mu.RUnlock()

	zones := make([]*Zone, len(r.zones))
	copy(zones, r.zones)
	return zones
}
