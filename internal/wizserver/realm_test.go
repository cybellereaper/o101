package wizserver

import "testing"

func TestRealmAssignZoneDistributesPlayers(t *testing.T) {
	ResetGID()
	realm := NewRealm(4, 2, 1)

	players := []*InGameCharacter{
		CreateDefaultPlayerInstance(),
		CreateDefaultPlayerInstance(),
	}
	for _, player := range players {
		player.Socket = &MockSocket{}
		if err := realm.AssignZone(player); err != nil {
			t.Fatalf("unexpected assign error: %v", err)
		}
	}

	zones := realm.Zones()
	if got := zones[0].PlayerCount(); got != 1 {
		t.Fatalf("expected zone 0 to host 1 player, got %d", got)
	}
	if got := zones[1].PlayerCount(); got != 1 {
		t.Fatalf("expected zone 1 to host 1 player, got %d", got)
	}

	third := CreateDefaultPlayerInstance()
	third.Socket = &MockSocket{}
	if err := realm.AssignZone(third); err == nil {
		t.Fatalf("expected assignment failure when all zones are at capacity")
	}
}

func TestRealmCharacterCache(t *testing.T) {
	ResetGID()
	realm := NewRealm(2, 1, 2)

	info := DefaultCharacterInfo()
	info.UserID = NextGID()
	realm.CacheCharacterCreationInfo(info)

	cached, ok := realm.CharacterFromCache(info.GlobalID)
	if !ok {
		t.Fatalf("expected character to be cached")
	}
	if cached.GlobalID != info.GlobalID {
		t.Fatalf("cached character mismatch: %#v", cached)
	}

	characters := realm.CharactersForPlayer(info.UserID)
	if len(characters) != 1 {
		t.Fatalf("expected a single cached character, got %d", len(characters))
	}
	if characters[0].GlobalID != info.GlobalID {
		t.Fatalf("unexpected cached character id: %d", characters[0].GlobalID)
	}
}

func TestRealmCharacterFallback(t *testing.T) {
	ResetGID()
	realm := NewRealm(2, 1, 2)
	playerID := NextGID()

	characters := realm.CharactersForPlayer(playerID)
	if len(characters) != 1 {
		t.Fatalf("expected default character, got %d entries", len(characters))
	}
	if characters[0].UserID != playerID {
		t.Fatalf("default character user id mismatch, got %d want %d", characters[0].UserID, playerID)
	}
}
