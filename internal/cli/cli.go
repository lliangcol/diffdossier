package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/lliangcol/diffdossier/internal/buildinfo"
	"github.com/lliangcol/diffdossier/internal/config"
	"github.com/lliangcol/diffdossier/internal/platform"
	publicschema "github.com/lliangcol/diffdossier/pkg/schema"
)

// Run executes the command without terminating the process, which keeps CLI behavior testable.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return ExitUsage
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return ExitOK
	case "version":
		return runVersion(args[1:], stdout, stderr)
	case "config":
		return runConfig(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "prepare":
		return runPrepare(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return ExitUsage
	}
}

func runConfig(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "validate" {
		fmt.Fprintln(stderr, "usage: diffdossier config validate [--repo PATH] [--config PATH] [--json]")
		return ExitUsage
	}
	flags := flag.NewFlagSet("config validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "target Git repository")
	configPath := flags.String("config", "", "configuration file (default: <repo>/diffdossier.toml)")
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args[1:]); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 {
		return ExitUsage
	}

	resolvedRepo, err := filepath.Abs(*repo)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_USAGE_INVALID_PATH", err.Error()), ExitUsage)
	}
	path := *configPath
	if path == "" {
		path = filepath.Join(resolvedRepo, "diffdossier.toml")
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(resolvedRepo, path)
	}
	cfg, err := config.Load(path)
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_CONFIG_INVALID", err.Error()), ExitUsage)
	}

	result := map[string]any{
		"config_path":    filepath.Clean(path),
		"schema_version": cfg.SchemaVersion,
		"status":         "valid",
	}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(result))
	}
	_, err = fmt.Fprintf(stdout, "configuration valid: %s (schema %d)\n", filepath.Clean(path), cfg.SchemaVersion)
	if err != nil {
		fmt.Fprintf(stderr, "write config validation output: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil || flags.NArg() != 0 {
		return ExitUsage
	}
	paths, err := platform.DefaultPaths()
	if err != nil {
		return writeFailure(stdout, stderr, *jsonOutput, publicschema.NewError("DD_PLATFORM_PATHS", err.Error()), ExitInternal)
	}
	result := map[string]any{
		"config_path":       paths.ConfigFile,
		"state_dir":         paths.StateDir,
		"cache_dir":         paths.CacheDir,
		"network_default":   "none",
		"default_provider":  "manual",
		"telemetry":         false,
		"strong_os_sandbox": false,
	}
	if *jsonOutput {
		return writeJSON(stdout, stderr, publicschema.Success(result))
	}
	_, err = fmt.Fprintf(stdout, "DiffDossier doctor\nconfig: %s\nstate: %s\ncache: %s\nnetwork: none\nprovider: manual\ntelemetry: disabled\n", paths.ConfigFile, paths.StateDir, paths.CacheDir)
	if err != nil {
		fmt.Fprintf(stderr, "write doctor output: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

func writeJSON(stdout, stderr io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintf(stderr, "encode JSON output: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

func writeFailure(stdout, stderr io.Writer, jsonOutput bool, problem publicschema.Problem, code int) int {
	if jsonOutput {
		if writeCode := writeJSON(stdout, stderr, publicschema.Failure(problem)); writeCode != ExitOK {
			return writeCode
		}
		return code
	}
	fmt.Fprintln(stderr, problem.Message)
	return code
}

func runVersion(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("version", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "emit stable JSON")
	if err := flags.Parse(args); errors.Is(err, flag.ErrHelp) {
		return ExitOK
	} else if err != nil {
		return ExitUsage
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "version does not accept positional arguments")
		return ExitUsage
	}

	info := buildinfo.Current()
	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(info); err != nil {
			fmt.Fprintf(stderr, "encode version output: %v\n", err)
			return ExitInternal
		}
		return ExitOK
	}

	if _, err := fmt.Fprintf(
		stdout,
		"diffdossier %s commit=%s built=%s go=%s %s/%s cgo=%s\n",
		info.Version,
		info.Commit,
		info.BuildDate,
		info.GoVersion,
		info.OS,
		info.Architecture,
		info.CGOEnabled,
	); err != nil {
		fmt.Fprintf(stderr, "write version output: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "Usage:")
	fmt.Fprintln(writer, "  diffdossier version [--json]")
	fmt.Fprintln(writer, "  diffdossier doctor [--json]")
	fmt.Fprintln(writer, "  diffdossier prepare [--repo PATH] [--config PATH] [--state-dir PATH] [--json]")
	fmt.Fprintln(writer, "  diffdossier config validate [--repo PATH] [--config PATH] [--json]")
	fmt.Fprintln(writer, "  diffdossier help")
}
