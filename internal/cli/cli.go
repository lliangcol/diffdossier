package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/lliangcol/diffdossier/internal/buildinfo"
)

const (
	ExitOK       = 0
	ExitUsage    = 2
	ExitInternal = 8
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
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return ExitUsage
	}
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
	fmt.Fprintln(writer, "  diffdossier help")
}
