package main

import (
	"fmt"
	"github.com/sda1_hacker/iec61850/cmd/scltool/cmds"
	"os"
)

func main() {
	if err := cmds.New().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
