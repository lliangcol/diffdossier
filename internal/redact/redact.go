// Package redact performs deterministic defensive secret scanning and log
// redaction. It is a safety layer, not proof that arbitrary content is safe.
package redact

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"sort"
)

const MaxScanBytes = 16 * 1024 * 1024

type Finding struct {
	Rule        string `json:"rule"`
	Offset      int    `json:"offset"`
	MatchDigest string `json:"match_digest"`
}
type Manifest struct {
	SchemaVersion string    `json:"schema_version"`
	InputDigest   string    `json:"input_digest"`
	OutputDigest  string    `json:"output_digest"`
	Findings      []Finding `json:"findings"`
	Truncated     bool      `json:"truncated"`
}

var rules = []struct {
	name       string
	expression *regexp.Regexp
}{
	{"private-key", regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`)},
	{"aws-access-key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"github-token", regexp.MustCompile(`(?:ghp|github_pat)_[A-Za-z0-9_]{20,}`)},
	{"jwt", regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`)},
	{"assigned-secret", regexp.MustCompile(`(?i)(?:api[_-]?key|client[_-]?secret|access[_-]?token|password)\s*[:=]\s*[^\s,;]{8,}`)},
}

func Scan(input []byte) ([]Finding, error) {
	if len(input) > MaxScanBytes {
		return nil, errors.New("redaction scan input exceeds 16 MiB")
	}
	findings := []Finding{}
	for _, rule := range rules {
		for _, location := range rule.expression.FindAllIndex(input, -1) {
			match := input[location[0]:location[1]]
			findings = append(findings, Finding{Rule: rule.name, Offset: location[0], MatchDigest: digest(match)})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Offset != findings[j].Offset {
			return findings[i].Offset < findings[j].Offset
		}
		return findings[i].Rule < findings[j].Rule
	})
	return findings, nil
}

func ContainsSecret(input []byte) bool {
	findings, err := Scan(input)
	return err != nil || len(findings) > 0
}

func Redact(input []byte) ([]byte, Manifest, error) {
	return RedactKnown(input, nil)
}

func RedactKnown(input []byte, known []string) ([]byte, Manifest, error) {
	if len(input) > MaxScanBytes {
		return nil, Manifest{}, errors.New("redaction input exceeds 16 MiB")
	}
	output := append([]byte(nil), input...)
	findings := []Finding{}
	knownCopy := append([]string(nil), known...)
	sort.Slice(knownCopy, func(i, j int) bool { return len(knownCopy[i]) > len(knownCopy[j]) })
	seen := map[string]bool{}
	for _, secret := range knownCopy {
		if secret == "" || seen[secret] {
			continue
		}
		seen[secret] = true
		needle := []byte(secret)
		for {
			offset := bytes.Index(output, needle)
			if offset < 0 {
				break
			}
			findings = append(findings, Finding{Rule: "known-value", Offset: offset, MatchDigest: digest(needle)})
			output = bytes.Join([][]byte{output[:offset], []byte("[REDACTED:known-value]"), output[offset+len(needle):]}, nil)
		}
	}
	for _, rule := range rules {
		matches := rule.expression.FindAllIndex(output, -1)
		for index := len(matches) - 1; index >= 0; index-- {
			location := matches[index]
			match := append([]byte(nil), output[location[0]:location[1]]...)
			findings = append(findings, Finding{Rule: rule.name, Offset: location[0], MatchDigest: digest(match)})
			replacement := []byte("[REDACTED:" + rule.name + "]")
			output = bytes.Join([][]byte{output[:location[0]], replacement, output[location[1]:]}, nil)
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Offset != findings[j].Offset {
			return findings[i].Offset < findings[j].Offset
		}
		return findings[i].Rule < findings[j].Rule
	})
	return output, Manifest{SchemaVersion: "1.0", InputDigest: digest(input), OutputDigest: digest(output), Findings: findings, Truncated: false}, nil
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}
