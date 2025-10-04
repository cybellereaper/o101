package wizserver

import "testing"

func TestZoneAddPlayerAssignsMobileID(t *testing.T) {
	ResetGID()
	zone := NewZone(2)

	player1 := CreateDefaultPlayerInstance()
	player1.Socket = &MockSocket{}
	if err := zone.AddPlayer(player1); err != nil {
		t.Fatalf("unexpected error adding player1: %v", err)
	}
	if player1.MobileID == 0 {
		t.Fatalf("expected non-zero mobile id for player1")
	}

	player2 := CreateDefaultPlayerInstance()
	player2.Socket = &MockSocket{}
	if err := zone.AddPlayer(player2); err != nil {
		t.Fatalf("unexpected error adding player2: %v", err)
	}
	if player2.MobileID == player1.MobileID {
		t.Fatalf("expected unique mobile id, got %d for both players", player2.MobileID)
	}
}

func TestZoneAddPlayerCapacityExceeded(t *testing.T) {
	ResetGID()
	zone := NewZone(1)

	player1 := CreateDefaultPlayerInstance()
	player1.Socket = &MockSocket{}
	if err := zone.AddPlayer(player1); err != nil {
		t.Fatalf("unexpected error adding player1: %v", err)
	}

	player2 := CreateDefaultPlayerInstance()
	player2.Socket = &MockSocket{}
	if err := zone.AddPlayer(player2); err == nil {
		t.Fatalf("expected capacity error when adding player2")
	}
}

func TestZoneBroadcast(t *testing.T) {
	ResetGID()
	zone := NewZone(3)

	players := make([]*InGameCharacter, 3)
	sockets := make([]*MockSocket, 3)
	for i := 0; i < 3; i++ {
		player := CreateDefaultPlayerInstance()
		socket := &MockSocket{}
		player.Socket = socket
		if err := zone.AddPlayer(player); err != nil {
			t.Fatalf("unexpected error adding player %d: %v", i, err)
		}
		players[i] = player
		sockets[i] = socket
	}

	message := struct{ Text string }{Text: "hello"}
	zone.Broadcast(players[0].GID, message)

	if len(sockets[0].Messages) != 0 {
		t.Fatalf("source socket received broadcast: %#v", sockets[0].Messages)
	}
	for i := 1; i < len(sockets); i++ {
		if got := len(sockets[i].Messages); got != 1 {
			t.Fatalf("expected 1 message for player %d, got %d", i, got)
		}
		if sockets[i].Messages[0] != message {
			t.Fatalf("unexpected message payload for player %d: %#v", i, sockets[i].Messages[0])
		}
	}
}
