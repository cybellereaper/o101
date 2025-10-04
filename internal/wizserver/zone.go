package wizserver

import (
	"errors"
	"sync"
)

// Zone models a shard in the realm that owns a subset of the connected
// players. It is responsible for broadcasting messages and managing player
// membership.
type Zone struct {
	mu              sync.RWMutex
	maxPlayers      uint32
	players         map[GID]*InGameCharacter
	mobileIDCounter uint16
}

// NewZone constructs a zone with the requested capacity.
func NewZone(maxPlayers uint32) *Zone {
	return &Zone{
		maxPlayers:      maxPlayers,
		players:         make(map[GID]*InGameCharacter),
		mobileIDCounter: 1,
	}
}

// AddPlayer registers a player inside the zone and assigns a unique mobile ID.
// If the zone is already full an error is returned.
func (z *Zone) AddPlayer(player *InGameCharacter) error {
	z.mu.Lock()
	defer z.mu.Unlock()

	if uint32(len(z.players)) >= z.maxPlayers {
		return errors.New("zone capacity reached")
	}

	if player.MobileID == 0 {
		player.MobileID = z.mobileIDCounter
		z.mobileIDCounter++
	}

	z.players[player.GID] = player
	return nil
}

// RemovePlayer ejects a player from the zone. The removed instance is returned
// to the caller for clean-up. Removing a player that is not managed by the zone
// yields (nil, false).
func (z *Zone) RemovePlayer(id GID) (*InGameCharacter, bool) {
	z.mu.Lock()
	defer z.mu.Unlock()

	player, ok := z.players[id]
	if ok {
		delete(z.players, id)
	}
	return player, ok
}

// Broadcast sends a message to every player in the zone except for the source
// identifier provided.
func (z *Zone) Broadcast(src GID, msgs ...Message) {
	z.mu.RLock()
	players := make([]*InGameCharacter, 0, len(z.players))
	for _, pl := range z.players {
		players = append(players, pl)
	}
	z.mu.RUnlock()

	for _, pl := range players {
		if pl.GID == src || pl.Socket == nil {
			continue
		}
		_ = pl.Socket.Send(msgs...)
	}
}

// PlayerCount returns the number of players currently managed by the zone.
func (z *Zone) PlayerCount() uint32 {
	z.mu.RLock()
	defer z.mu.RUnlock()
	return uint32(len(z.players))
}

// Capacity returns the maximum number of players supported by the zone.
func (z *Zone) Capacity() uint32 {
	return z.maxPlayers
}

// ForEach iterates over the players in the zone. The callback is executed under
// a read lock to allow safe read-only inspection.
func (z *Zone) ForEach(cb func(*InGameCharacter)) {
	z.mu.RLock()
	defer z.mu.RUnlock()

	for _, pl := range z.players {
		cb(pl)
	}
}
