package providers

import (
	"context"
	"errors"
	"testing"

	"github.com/lliangcol/diffdossier/internal/packets"
)

func TestManualNeverExecutesModel(t *testing.T) {
	handshake, err := (Manual{}).Handshake(context.Background())
	if err != nil || handshake.NetworkAccess != "none" {
		t.Fatalf("handshake=%+v err=%v", handshake, err)
	}
	if _, err := (Manual{}).Review(context.Background(), packets.Packet{}); !errors.Is(err, ErrManualPending) {
		t.Fatalf("manual review err=%v", err)
	}
}
