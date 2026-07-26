// Package schemas embeds the public JSON contracts into the binary so their
// exact digests can participate in snapshot invalidation.
package schemas

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
	"sort"
)

//go:embed *.schema.json
var files embed.FS

func Digests() (map[string]string, error) {
	names, err := fs.Glob(files, "*.schema.json")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	result := make(map[string]string, len(names))
	for _, name := range names {
		content, err := files.ReadFile(name)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(content)
		result["schema/"+name] = "sha256:" + hex.EncodeToString(digest[:])
	}
	return result, nil
}

// Read returns an embedded public schema by its repository filename.
func Read(name string) ([]byte, error) {
	return files.ReadFile(name)
}
