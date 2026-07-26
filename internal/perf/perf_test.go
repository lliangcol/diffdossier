package perf_test

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/lliangcol/diffdossier/internal/contracts"
	"github.com/lliangcol/diffdossier/internal/inventory"
	"github.com/lliangcol/diffdossier/internal/planner"
	"github.com/lliangcol/diffdossier/internal/redact"
	"github.com/lliangcol/diffdossier/internal/risk"
)

func BenchmarkPlanTenThousandPaths(b *testing.B) {
	entries := make([]inventory.Entry, 10000)
	assessment := risk.Assessment{Paths: make([]risk.PathRisk, 10000)}
	for index := range entries {
		path := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("pkg/%05d.go", index)))
		entries[index] = inventory.Entry{Path: inventory.PathIdentity{BytesBase64: path}, Size: 1024, ContentHash: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(path)))}
		assessment.Paths[index] = risk.PathRisk{PathBytesBase64: path, Level: risk.L1}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = planner.Build("snap", entries, contracts.Graph{}, assessment, planner.Limits{MaxFiles: 8, MaxPacketBytes: 250000})
	}
}

func BenchmarkRedactOneMiB(b *testing.B) {
	input := make([]byte, 1024*1024)
	copy(input, []byte("ordinary test log without credentials"))
	b.SetBytes(int64(len(input)))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, _, err := redact.Redact(input); err != nil {
			b.Fatal(err)
		}
	}
}
