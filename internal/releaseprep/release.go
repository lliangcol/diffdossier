package releaseprep

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const schemaVersion = "1.0"

var targets = []Target{
	{OS: "darwin", Arch: "amd64", Format: "tar.gz"},
	{OS: "darwin", Arch: "arm64", Format: "tar.gz"},
	{OS: "linux", Arch: "amd64", Format: "tar.gz"},
	{OS: "linux", Arch: "arm64", Format: "tar.gz"},
	{OS: "windows", Arch: "amd64", Format: "zip"},
	{OS: "windows", Arch: "arm64", Format: "zip"},
}

var (
	candidateVersionPattern = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z._+-]{0,63}$`)
	hexCommitPattern        = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

type Target struct {
	OS     string `json:"os"`
	Arch   string `json:"architecture"`
	Format string `json:"format"`
}

type Options struct {
	Repo      string
	Output    string
	Version   string
	Commit    string
	Candidate bool
}

type Artifact struct {
	Name   string `json:"name"`
	OS     string `json:"os"`
	Arch   string `json:"architecture"`
	Format string `json:"format"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type Manifest struct {
	SchemaVersion string     `json:"schema_version"`
	Project       string     `json:"project"`
	Version       string     `json:"version"`
	Commit        string     `json:"commit"`
	SourceDate    string     `json:"source_date"`
	GoVersion     string     `json:"go_version"`
	Candidate     bool       `json:"candidate"`
	Artifacts     []Artifact `json:"artifacts"`
}

type Provenance struct {
	SchemaVersion string     `json:"schema_version"`
	Kind          string     `json:"kind"`
	Signed        bool       `json:"signed"`
	Source        string     `json:"source"`
	Version       string     `json:"version"`
	Commit        string     `json:"commit"`
	SourceDate    string     `json:"source_date"`
	GoVersion     string     `json:"go_version"`
	BuildFlags    []string   `json:"build_flags"`
	Artifacts     []Artifact `json:"artifacts"`
}

type VerifyResult struct {
	FilesChecked int
	Version      string
	Commit       string
	SmokeTarget  string
}

func Build(opts Options) (Manifest, error) {
	var empty Manifest
	repo, err := filepath.Abs(opts.Repo)
	if err != nil {
		return empty, fmt.Errorf("resolve repository: %w", err)
	}
	output, err := filepath.Abs(opts.Output)
	if err != nil {
		return empty, fmt.Errorf("resolve output: %w", err)
	}
	if strings.TrimSpace(opts.Version) == "" {
		return empty, errors.New("version is required")
	}
	if !candidateVersionPattern.MatchString(opts.Version) {
		return empty, errors.New("version must be a path-safe identifier of at most 64 characters")
	}
	if !opts.Candidate && !validSemVer(opts.Version) {
		return empty, errors.New("release version must be a v-prefixed semantic version")
	}
	head, err := git(repo, "rev-parse", "HEAD")
	if err != nil {
		return empty, err
	}
	if opts.Commit != "" && opts.Commit != head {
		return empty, fmt.Errorf("requested commit %s does not match HEAD %s", opts.Commit, head)
	}
	status, err := git(repo, "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil {
		return empty, err
	}
	if status != "" {
		return empty, errors.New("release preparation requires a clean working tree")
	}
	if !opts.Candidate {
		tagCommit, err := git(repo, "rev-parse", "refs/tags/"+opts.Version+"^{commit}")
		if err != nil {
			return empty, fmt.Errorf("release tag %q is required: %w", opts.Version, err)
		}
		if tagCommit != head {
			return empty, fmt.Errorf("release tag %q resolves to %s, not HEAD %s", opts.Version, tagCommit, head)
		}
	}
	sourceText, err := git(repo, "show", "-s", "--format=%cI", head)
	if err != nil {
		return empty, err
	}
	sourceDate, err := time.Parse(time.RFC3339, sourceText)
	if err != nil {
		return empty, fmt.Errorf("parse commit date %q: %w", sourceText, err)
	}
	sourceDate = sourceDate.UTC()
	goVersion, err := controlledGoVersion(repo)
	if err != nil {
		return empty, err
	}
	staging, err := prepareStaging(output)
	if err != nil {
		return empty, err
	}
	defer os.RemoveAll(staging)
	work, err := os.MkdirTemp("", "diffdossier-release-")
	if err != nil {
		return empty, fmt.Errorf("create release work directory: %w", err)
	}
	defer os.RemoveAll(work)

	manifest := Manifest{
		SchemaVersion: schemaVersion,
		Project:       "DiffDossier",
		Version:       opts.Version,
		Commit:        head,
		SourceDate:    sourceDate.Format(time.RFC3339),
		GoVersion:     goVersion,
		Candidate:     opts.Candidate,
	}
	for _, target := range targets {
		artifact, err := buildTarget(repo, staging, work, opts.Version, head, sourceDate, target)
		if err != nil {
			return empty, err
		}
		manifest.Artifacts = append(manifest.Artifacts, artifact)
	}
	if err := writeJSON(filepath.Join(staging, "release-manifest.json"), manifest); err != nil {
		return empty, err
	}
	if err := writeJSON(filepath.Join(staging, "diffdossier.spdx.json"), newSBOM(manifest)); err != nil {
		return empty, err
	}
	provenance := Provenance{
		SchemaVersion: schemaVersion,
		Kind:          "local_unsigned_provenance",
		Signed:        false,
		Source:        "https://github.com/lliangcol/diffdossier",
		Version:       opts.Version,
		Commit:        head,
		SourceDate:    sourceDate.Format(time.RFC3339),
		GoVersion:     goVersion,
		BuildFlags:    []string{"CGO_ENABLED=0", "GOENV=off", "GOFLAGS=", "GONOSUMDB=*", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local", "GOWORK=off", "-mod=readonly", "-trimpath", "-buildvcs=false", "-ldflags=-s -w and exact buildinfo"},
		Artifacts:     manifest.Artifacts,
	}
	if err := writeJSON(filepath.Join(staging, "provenance.json"), provenance); err != nil {
		return empty, err
	}
	if err := writeChecksums(staging); err != nil {
		return empty, err
	}
	if _, err := Verify(staging, false); err != nil {
		return empty, fmt.Errorf("self-verify staged release: %w", err)
	}
	if err := os.Rename(staging, output); err != nil {
		return empty, fmt.Errorf("publish complete release directory: %w", err)
	}
	return manifest, nil
}

func Verify(dir string, smoke bool) (VerifyResult, error) {
	var result VerifyResult
	root, err := filepath.Abs(dir)
	if err != nil {
		return result, fmt.Errorf("resolve release directory: %w", err)
	}
	checksums, err := readChecksums(filepath.Join(root, "SHA256SUMS"))
	if err != nil {
		return result, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return result, fmt.Errorf("read release directory: %w", err)
	}
	regular := make(map[string]bool)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return result, fmt.Errorf("inspect release entry %s: %w", entry.Name(), err)
		}
		if !info.Mode().IsRegular() {
			return result, fmt.Errorf("release directory contains non-regular entry %q", entry.Name())
		}
		if entry.Name() != "SHA256SUMS" {
			regular[entry.Name()] = true
		}
	}
	if len(regular) != len(checksums) {
		return result, fmt.Errorf("checksum coverage mismatch: %d release files, %d checksum entries", len(regular), len(checksums))
	}
	for name, want := range checksums {
		if filepath.Base(name) != name || name == "." || name == ".." || !regular[name] {
			return result, fmt.Errorf("invalid or missing checksum target %q", name)
		}
		got, _, err := fileDigest(filepath.Join(root, name))
		if err != nil {
			return result, err
		}
		if got != want {
			return result, fmt.Errorf("checksum mismatch for %s", name)
		}
		result.FilesChecked++
	}
	var manifest Manifest
	if err := readJSON(filepath.Join(root, "release-manifest.json"), &manifest); err != nil {
		return result, err
	}
	if manifest.SchemaVersion != schemaVersion || manifest.Project != "DiffDossier" || !candidateVersionPattern.MatchString(manifest.Version) || !hexCommitPattern.MatchString(manifest.Commit) {
		return result, errors.New("release manifest identity is invalid")
	}
	if !manifest.Candidate && !validSemVer(manifest.Version) {
		return result, errors.New("release manifest version is not semantic")
	}
	result.Version = manifest.Version
	result.Commit = manifest.Commit
	expectedTargets := make(map[string]Target, len(targets))
	for _, target := range targets {
		name := fmt.Sprintf("diffdossier_%s_%s_%s.%s", manifest.Version, target.OS, target.Arch, target.Format)
		expectedTargets[name] = target
	}
	seenTargets := make(map[string]bool, len(targets))
	if len(manifest.Artifacts) != len(targets) {
		return result, fmt.Errorf("manifest has %d artifacts, want %d", len(manifest.Artifacts), len(targets))
	}
	for _, artifact := range manifest.Artifacts {
		target, expected := expectedTargets[artifact.Name]
		if !expected || seenTargets[artifact.Name] || artifact.OS != target.OS || artifact.Arch != target.Arch || artifact.Format != target.Format {
			return result, fmt.Errorf("manifest has invalid or duplicate target %s", artifact.Name)
		}
		seenTargets[artifact.Name] = true
		want, ok := checksums[artifact.Name]
		if !ok || want != artifact.SHA256 {
			return result, fmt.Errorf("manifest/checksum mismatch for %s", artifact.Name)
		}
		info, err := os.Stat(filepath.Join(root, artifact.Name))
		if err != nil {
			return result, err
		}
		if info.Size() != artifact.Size {
			return result, fmt.Errorf("manifest size mismatch for %s", artifact.Name)
		}
	}
	var provenance Provenance
	if err := readJSON(filepath.Join(root, "provenance.json"), &provenance); err != nil {
		return result, err
	}
	if provenance.SchemaVersion != schemaVersion || provenance.Kind != "local_unsigned_provenance" || provenance.Signed || provenance.Version != manifest.Version || provenance.Commit != manifest.Commit || provenance.SourceDate != manifest.SourceDate || provenance.GoVersion != manifest.GoVersion || !reflect.DeepEqual(provenance.Artifacts, manifest.Artifacts) {
		return result, errors.New("local provenance does not exactly bind the release manifest")
	}
	var sbom map[string]any
	if err := readJSON(filepath.Join(root, "diffdossier.spdx.json"), &sbom); err != nil {
		return result, err
	}
	if sbom["spdxVersion"] != "SPDX-2.3" || sbom["SPDXID"] != "SPDXRef-DOCUMENT" {
		return result, errors.New("SPDX document identity is invalid")
	}
	if smoke {
		target, err := smokeCurrent(root, manifest)
		if err != nil {
			return result, err
		}
		result.SmokeTarget = target
	}
	return result, nil
}

func validSemVer(value string) bool {
	if !strings.HasPrefix(value, "v") || len(value) > 64 {
		return false
	}
	version := value[1:]
	if strings.Count(version, "+") > 1 {
		return false
	}
	coreAndPre, build, hasBuild := strings.Cut(version, "+")
	if hasBuild && !validIdentifiers(build, false) {
		return false
	}
	if strings.Count(coreAndPre, "-") > 0 {
		coreAndPreParts := strings.SplitN(coreAndPre, "-", 2)
		coreAndPre = coreAndPreParts[0]
		if !validIdentifiers(coreAndPreParts[1], true) {
			return false
		}
	}
	core := strings.Split(coreAndPre, ".")
	if len(core) != 3 {
		return false
	}
	for _, part := range core {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return false
		}
		if _, err := strconv.ParseUint(part, 10, 64); err != nil {
			return false
		}
	}
	return true
}

func validIdentifiers(value string, rejectNumericLeadingZero bool) bool {
	for _, identifier := range strings.Split(value, ".") {
		if identifier == "" {
			return false
		}
		numeric := true
		for _, character := range identifier {
			if (character < '0' || character > '9') && (character < 'A' || character > 'Z') && (character < 'a' || character > 'z') && character != '-' {
				return false
			}
			if character < '0' || character > '9' {
				numeric = false
			}
		}
		if rejectNumericLeadingZero && numeric && len(identifier) > 1 && identifier[0] == '0' {
			return false
		}
	}
	return true
}

func buildTarget(repo, output, work, version, commit string, sourceDate time.Time, target Target) (Artifact, error) {
	var artifact Artifact
	base := fmt.Sprintf("diffdossier_%s_%s_%s", version, target.OS, target.Arch)
	stage := filepath.Join(work, base)
	if err := os.Mkdir(stage, 0o755); err != nil {
		return artifact, fmt.Errorf("create target stage: %w", err)
	}
	binaryName := "diffdossier"
	if target.OS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(stage, binaryName)
	ldflags := fmt.Sprintf("-s -w -X github.com/lliangcol/diffdossier/internal/buildinfo.Version=%s -X github.com/lliangcol/diffdossier/internal/buildinfo.Commit=%s -X github.com/lliangcol/diffdossier/internal/buildinfo.BuildDate=%s", version, commit, sourceDate.Format(time.RFC3339))
	command := exec.Command("go", "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-ldflags", ldflags, "-o", binaryPath, "./cmd/diffdossier")
	command.Dir = repo
	command.Env = controlledGoEnv(os.Environ(), target)
	outputBytes, err := command.CombinedOutput()
	if err != nil {
		return artifact, fmt.Errorf("build %s/%s: %w: %s", target.OS, target.Arch, err, strings.TrimSpace(string(outputBytes)))
	}
	for _, name := range []string{"LICENSE", "NOTICE", "README.md"} {
		if err := copyFile(filepath.Join(repo, name), filepath.Join(stage, name), 0o644); err != nil {
			return artifact, err
		}
	}
	archiveName := base + "." + target.Format
	archivePath := filepath.Join(output, archiveName)
	if target.Format == "zip" {
		err = writeZip(archivePath, stage, base, sourceDate)
	} else {
		err = writeTarGz(archivePath, stage, base, sourceDate)
	}
	if err != nil {
		return artifact, err
	}
	digest, size, err := fileDigest(archivePath)
	if err != nil {
		return artifact, err
	}
	return Artifact{Name: archiveName, OS: target.OS, Arch: target.Arch, Format: target.Format, SHA256: digest, Size: size}, nil
}

func controlledGoVersion(repo string) (string, error) {
	command := exec.Command("go", "env", "GOVERSION")
	command.Dir = repo
	command.Env = controlledGoEnv(os.Environ(), Target{})
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("read controlled Go builder version: %w: %s", err, strings.TrimSpace(string(output)))
	}
	version := strings.TrimSpace(string(output))
	if version == "" {
		return "", errors.New("controlled Go builder returned an empty version")
	}
	return version, nil
}

func controlledGoEnv(base []string, target Target) []string {
	replacements := map[string]string{
		"CGO_ENABLED":  "0",
		"GOENV":        "off",
		"GOEXPERIMENT": "",
		"GOFLAGS":      "",
		"GONOSUMDB":    "*",
		"GOPROXY":      "off",
		"GOSUMDB":      "off",
		"GOTOOLCHAIN":  "local",
		"GOWORK":       "off",
	}
	if target.OS != "" {
		replacements["GOOS"] = target.OS
	}
	if target.Arch != "" {
		replacements["GOARCH"] = target.Arch
	}
	return replaceEnv(base, replacements)
}

func writeTarGz(path, stage, base string, modTime time.Time) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer file.Close()
	gz := gzip.NewWriter(file)
	gz.Header.ModTime = time.Unix(0, 0).UTC()
	gz.Header.OS = 255
	tarWriter := tar.NewWriter(gz)
	for _, name := range []string{"LICENSE", "NOTICE", "README.md", "diffdossier"} {
		data, err := os.ReadFile(filepath.Join(stage, name))
		if err != nil {
			return err
		}
		mode := int64(0o644)
		if name == "diffdossier" {
			mode = 0o755
		}
		header := &tar.Header{Name: base + "/" + name, Mode: mode, Size: int64(len(data)), ModTime: modTime, AccessTime: time.Time{}, ChangeTime: time.Time{}, Uid: 0, Gid: 0, Typeflag: tar.TypeReg, Format: tar.FormatPAX}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write(data); err != nil {
			return err
		}
	}
	if err := tarWriter.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return file.Close()
}

func writeZip(path, stage, base string, modTime time.Time) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer file.Close()
	writer := zip.NewWriter(file)
	for _, name := range []string{"LICENSE", "NOTICE", "README.md", "diffdossier.exe"} {
		data, err := os.ReadFile(filepath.Join(stage, name))
		if err != nil {
			return err
		}
		header := &zip.FileHeader{Name: base + "/" + name, Method: zip.Deflate}
		header.SetModTime(modTime)
		if name == "diffdossier.exe" {
			header.SetMode(0o755)
		} else {
			header.SetMode(0o644)
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err := entry.Write(data); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return file.Close()
}

func smokeCurrent(root string, manifest Manifest) (string, error) {
	var selected Artifact
	for _, artifact := range manifest.Artifacts {
		if artifact.OS == runtime.GOOS && artifact.Arch == runtime.GOARCH {
			selected = artifact
			break
		}
	}
	if selected.Name == "" {
		return "", fmt.Errorf("no artifact for current platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	temp, err := os.MkdirTemp("", "diffdossier-install-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(temp)
	binary, err := extractBinary(filepath.Join(root, selected.Name), temp, selected.Format)
	if err != nil {
		return "", err
	}
	for _, args := range [][]string{{"version", "--json"}, {"doctor", "--json"}} {
		command := exec.Command(binary, args...)
		command.Env = append(os.Environ(), "HOME="+temp, "XDG_CONFIG_HOME="+filepath.Join(temp, "config"), "XDG_STATE_HOME="+filepath.Join(temp, "state"), "XDG_CACHE_HOME="+filepath.Join(temp, "cache"))
		out, err := command.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("installed binary %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
		if !json.Valid(out) {
			return "", fmt.Errorf("installed binary %s emitted invalid JSON", strings.Join(args, " "))
		}
		if args[0] == "version" {
			var info struct {
				Version string `json:"version"`
				Commit  string `json:"commit"`
			}
			if err := json.Unmarshal(out, &info); err != nil {
				return "", err
			}
			if info.Version != manifest.Version || info.Commit != manifest.Commit {
				return "", fmt.Errorf("installed identity mismatch: version=%s commit=%s", info.Version, info.Commit)
			}
		}
	}
	return runtime.GOOS + "/" + runtime.GOARCH, nil
}

func extractBinary(archivePath, output, format string) (string, error) {
	if format == "zip" {
		reader, err := zip.OpenReader(archivePath)
		if err != nil {
			return "", err
		}
		defer reader.Close()
		for _, file := range reader.File {
			if filepath.Base(file.Name) != "diffdossier.exe" {
				continue
			}
			in, err := file.Open()
			if err != nil {
				return "", err
			}
			defer in.Close()
			path := filepath.Join(output, "diffdossier.exe")
			if err := writeReader(path, in, 0o755); err != nil {
				return "", err
			}
			return path, nil
		}
		return "", errors.New("archive does not contain diffdossier.exe")
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if filepath.Base(header.Name) != "diffdossier" {
			continue
		}
		path := filepath.Join(output, "diffdossier")
		if err := writeReader(path, reader, 0o755); err != nil {
			return "", err
		}
		return path, nil
	}
	return "", errors.New("archive does not contain diffdossier")
}

func prepareStaging(output string) (string, error) {
	if _, err := os.Lstat(output); err == nil {
		return "", errors.New("release output path must not already exist")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect release output path: %w", err)
	}
	parent := filepath.Dir(output)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", fmt.Errorf("create release output parent: %w", err)
	}
	staging, err := os.MkdirTemp(parent, "."+filepath.Base(output)+".staging-")
	if err != nil {
		return "", fmt.Errorf("create release staging directory: %w", err)
	}
	return staging, nil
}

func git(repo string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = repo
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func replaceEnv(base []string, replacements map[string]string) []string {
	result := make([]string, 0, len(base)+len(replacements))
	for _, value := range base {
		key, _, _ := strings.Cut(value, "=")
		if _, replace := replacements[key]; !replace {
			result = append(result, value)
		}
	}
	keys := make([]string, 0, len(replacements))
	for key := range replacements {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		result = append(result, key+"="+replacements[key])
	}
	return result
}

func copyFile(source, target string, mode fs.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open %s: %w", source, err)
	}
	defer in.Close()
	return writeReader(target, in, mode)
}

func writeReader(target string, reader io.Reader, mode fs.FileMode) error {
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, reader); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(target, mode)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func fileDigest(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

func writeChecksums(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	var names []string
	for _, entry := range entries {
		if entry.Type().IsRegular() && entry.Name() != "SHA256SUMS" {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	var builder strings.Builder
	for _, name := range names {
		digest, _, err := fileDigest(filepath.Join(root, name))
		if err != nil {
			return err
		}
		fmt.Fprintf(&builder, "%s  %s\n", digest, name)
	}
	return os.WriteFile(filepath.Join(root, "SHA256SUMS"), []byte(builder.String()), 0o644)
}

func readChecksums(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open checksums: %w", err)
	}
	defer file.Close()
	result := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 67 || line[64:66] != "  " {
			return nil, fmt.Errorf("invalid checksum line %q", line)
		}
		digest := line[:64]
		if _, err := hex.DecodeString(digest); err != nil {
			return nil, fmt.Errorf("invalid checksum digest for %q", line[66:])
		}
		name := line[66:]
		if _, duplicate := result[name]; duplicate {
			return nil, fmt.Errorf("duplicate checksum target %q", name)
		}
		result[name] = digest
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, errors.New("empty SHA256SUMS")
	}
	return result, nil
}

func newSBOM(manifest Manifest) map[string]any {
	packageID := "SPDXRef-Package-DiffDossier"
	return map[string]any{
		"spdxVersion":       "SPDX-2.3",
		"dataLicense":       "CC0-1.0",
		"SPDXID":            "SPDXRef-DOCUMENT",
		"name":              "DiffDossier-" + manifest.Version,
		"documentNamespace": "https://github.com/lliangcol/diffdossier/sbom/" + manifest.Version + "/" + manifest.Commit,
		"creationInfo": map[string]any{
			"created":  manifest.SourceDate,
			"creators": []string{"Tool: diffdossier-releaseprep", "Organization: DiffDossier contributors"},
			"comment":  "Source/module SBOM. The current Go module declares no third-party modules; Go standard-library packages are supplied by the recorded Go toolchain.",
		},
		"packages": []map[string]any{{
			"name":             "github.com/lliangcol/diffdossier",
			"SPDXID":           packageID,
			"versionInfo":      manifest.Version,
			"downloadLocation": "https://github.com/lliangcol/diffdossier/tree/" + manifest.Commit,
			"filesAnalyzed":    false,
			"licenseConcluded": "Apache-2.0",
			"licenseDeclared":  "Apache-2.0",
			"copyrightText":    "Copyright 2026 DiffDossier contributors",
			"externalRefs": []map[string]string{{
				"referenceCategory": "PACKAGE-MANAGER",
				"referenceType":     "purl",
				"referenceLocator":  "pkg:golang/github.com/lliangcol/diffdossier@" + manifest.Version,
			}},
		}},
		"relationships": []map[string]string{{
			"spdxElementId":      "SPDXRef-DOCUMENT",
			"relationshipType":   "DESCRIBES",
			"relatedSpdxElement": packageID,
		}},
	}
}
