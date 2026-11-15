package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Gunmer/kamino/internal/release"
)

func main() {
	sha := flag.String("sha", "", "commit de main validado por CI")
	tag := flag.String("tag", "", "tag existente cuya publicación se debe completar")
	flag.Parse()
	if (*sha == "") == (*tag == "") {
		fmt.Fprintln(os.Stderr, "se requiere exactamente una de las opciones --sha o --tag")
		flag.Usage()
		os.Exit(2)
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := release.Run(root, *sha, *tag); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
