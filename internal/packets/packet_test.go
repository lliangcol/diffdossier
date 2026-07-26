package packets

import (
	"testing"

	"github.com/lliangcol/diffdossier/internal/inventory"
	"github.com/lliangcol/diffdossier/internal/planner"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

func TestBuildNeverDropsOversizedReferences(t *testing.T) {
	task := planner.Task{
		ID: "task-a", SnapshotID: "snap-a", Oversized: true, TotalBytes: 1000,
		Paths: []planner.PathRef{{
			Scope: inventory.ScopeCommitted, PathBytesBase64: "YQ==", DisplayPath: "a",
			CurrentHash: "sha256:current", PreviousHash: "sha256:previous", RequiredCoverage: "fully_reviewed",
		}},
	}
	packet, err := Build(task, publicschema.PrivateProject)
	if err != nil {
		t.Fatal(err)
	}
	if packet.Status != "incomplete" || len(packet.Files) != 1 || !packet.Files[0].FullReadRequired || packet.TaskInputHash == "" {
		t.Fatalf("packet silently lost oversized input: %+v", packet)
	}
}

func TestSecretDeniedCannotEnterPacket(t *testing.T) {
	if _, err := Build(planner.Task{}, publicschema.SecretDenied); err == nil {
		t.Fatal("secret_denied packet should fail")
	}
}
