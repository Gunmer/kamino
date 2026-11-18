// Command kamino is the command-line entry point.
package main

import (
	"os"

	"github.com/Gunmer/kamino/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
