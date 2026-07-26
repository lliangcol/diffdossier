package main

import (
	"os"

	"github.com/lliangcol/diffdossier/internal/adapters"
)

func main() {
	os.Exit(adapters.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
