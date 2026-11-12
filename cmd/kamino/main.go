package main

import (
	"fmt"
	"os"

	"github.com/Gunmer/kamino/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

var _ = fmt.Sprintf
