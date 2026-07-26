package risk

import (
	"testing"

	"github.com/lliangcol/diffdossier/internal/contracts"
	"github.com/lliangcol/diffdossier/internal/inventory"
)

func TestPolicyCanRaiseButNotLower(t *testing.T) {
	path := "payment/handler.go"
	entry := inventory.Entry{Kind: "regular", Path: inventory.PathIdentity{BytesBase64: "cGF5bWVudC9oYW5kbGVyLmdv", UTF8: &path}}
	assessment, err := Assess([]inventory.Entry{entry}, contracts.Graph{}, []Override{{Glob: "payment/*", Level: L1, Reason: "attempt lower"}})
	if err != nil {
		t.Fatal(err)
	}
	if assessment.Paths[0].Level != L4 {
		t.Fatalf("explicit/inferred high risk was lowered: %+v", assessment.Paths[0])
	}
}

func TestUnknownFailsClosed(t *testing.T) {
	path := "artifact.odd"
	entry := inventory.Entry{Kind: "regular", Path: inventory.PathIdentity{BytesBase64: "YXJ0aWZhY3Qub2Rk", UTF8: &path}}
	assessment, err := Assess([]inventory.Entry{entry}, contracts.Graph{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !assessment.Paths[0].NeedsConfirmation || assessment.Paths[0].Level != L2 {
		t.Fatalf("unknown path did not fail closed: %+v", assessment.Paths[0])
	}
}
