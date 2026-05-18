package invoke

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/tg"
)

// ResolvePeer takes a user-provided peer string and resolves it to an
// InputPeerClass via the client. Supported formats: @username, +1234567890,
// "me"/"self", numeric ID, channel:ID, user:ID.
func ResolvePeer(ctx context.Context, client *telegram.Client, peerStr string) (tg.InputPeerClass, error) {
	if peerStr == "me" || peerStr == "self" {
		return &tg.InputPeerSelf{}, nil
	}

	if idStr, ok := strings.CutPrefix(peerStr, "channel:"); ok {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid channel ID: %q", idStr)
		}
		return client.ResolvePeer(ctx, id)
	}

	if idStr, ok := strings.CutPrefix(peerStr, "user:"); ok {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID: %q", idStr)
		}
		return client.ResolvePeer(ctx, id)
	}

	return client.ResolvePeer(ctx, peerStr)
}

// PeerString returns a human-readable description of a resolved peer.
func PeerString(peer tg.InputPeerClass) string {
	switch p := peer.(type) {
	case *tg.InputPeerSelf:
		return "self"
	case *tg.InputPeerUser:
		return fmt.Sprintf("user:%d", p.UserID)
	case *tg.InputPeerChat:
		return fmt.Sprintf("chat:%d", p.ChatID)
	case *tg.InputPeerChannel:
		return fmt.Sprintf("channel:%d", p.ChannelID)
	default:
		return fmt.Sprintf("%T", peer)
	}
}
