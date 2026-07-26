// Package providers defines the external review boundary.
package providers

import (
	"context"
	"errors"

	"github.com/lliangcol/diffdossier/internal/packets"
	"github.com/lliangcol/diffdossier/internal/results"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

var ErrManualPending = errors.New("manual packet is ready; import a separately reviewed result")

type Provider interface {
	Handshake(context.Context) (publicschema.ProviderHandshake, error)
	Review(context.Context, packets.Packet) (results.Result, error)
}

type Manual struct{}

func (Manual) Handshake(context.Context) (publicschema.ProviderHandshake, error) {
	return publicschema.ProviderHandshake{
		ProtocolVersion: "1.0", Provider: "manual",
		Capabilities: []string{"review", "structured-result"}, MaxInputBytes: 250000,
		SupportsResume: true, NetworkAccess: "none",
	}, nil
}

func (Manual) Review(context.Context, packets.Packet) (results.Result, error) {
	return results.Result{}, ErrManualPending
}

type Mock struct {
	Result results.Result
}

func (Mock) Handshake(context.Context) (publicschema.ProviderHandshake, error) {
	return publicschema.ProviderHandshake{
		ProtocolVersion: "1.0", Provider: "mock",
		Capabilities: []string{"review", "structured-result"}, MaxInputBytes: 250000,
		SupportsResume: true, NetworkAccess: "none",
	}, nil
}

func (mock Mock) Review(context.Context, packets.Packet) (results.Result, error) {
	return mock.Result, nil
}
