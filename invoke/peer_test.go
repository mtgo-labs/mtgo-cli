package invoke

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestResolvePeerSelf(t *testing.T) {
	peer, err := ResolvePeer(nil, nil, "me")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := peer.(*tg.InputPeerSelf); !ok {
		t.Errorf("expected InputPeerSelf, got %T", peer)
	}
}

func TestResolvePeerSelfAlias(t *testing.T) {
	peer, err := ResolvePeer(nil, nil, "self")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := peer.(*tg.InputPeerSelf); !ok {
		t.Errorf("expected InputPeerSelf, got %T", peer)
	}
}

func TestPeerString(t *testing.T) {
	tests := []struct {
		peer tg.InputPeerClass
		want string
	}{
		{&tg.InputPeerSelf{}, "self"},
		{&tg.InputPeerUser{UserID: 123}, "user:123"},
		{&tg.InputPeerChat{ChatID: 456}, "chat:456"},
		{&tg.InputPeerChannel{ChannelID: 789}, "channel:789"},
	}
	for _, tt := range tests {
		got := PeerString(tt.peer)
		if got != tt.want {
			t.Errorf("PeerString(%T) = %s, want %s", tt.peer, got, tt.want)
		}
	}
}
