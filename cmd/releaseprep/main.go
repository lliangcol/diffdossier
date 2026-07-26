// Command releaseprep builds and verifies DiffDossier release artifacts.
// It is maintainer tooling, not part of the stable DiffDossier CLI.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/lliangcol/diffdossier/internal/releaseprep"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "build":
		err = runBuild(os.Args[2:])
	case "verify":
		err = runVerify(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runBuild(args []string) error {
	flags := flag.NewFlagSet("releaseprep build", flag.ContinueOnError)
	repo := flags.String("repo", ".", "clean DiffDossier repository")
	output := flags.String("output", "dist", "new output directory that does not exist")
	version := flags.String("version", "", "exact release tag or candidate version")
	commit := flags.String("commit", "", "exact full HEAD commit")
	candidate := flags.Bool("candidate", false, "permit an untagged local candidate; never publish it as a release")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	manifest, err := releaseprep.Build(releaseprep.Options{Repo: *repo, Output: *output, Version: *version, Commit: *commit, Candidate: *candidate})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(manifest)
}

func runVerify(args []string) error {
	flags := flag.NewFlagSet("releaseprep verify", flag.ContinueOnError)
	dir := flags.String("dir", "dist", "release artifact directory")
	smoke := flags.Bool("smoke", false, "extract and run version/doctor for the current platform")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	result, err := releaseprep.Verify(*dir, *smoke)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: releaseprep build --version VERSION --commit COMMIT --output DIR [--candidate]")
	fmt.Fprintln(os.Stderr, "       releaseprep verify --dir DIR [--smoke]")
}
