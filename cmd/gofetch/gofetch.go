package main

import (
	"fmt"
	"io"
	"os"

	"github.com/sealuzh/goabs/deps"
)

const argsLen = 1

var errorOut io.Writer = os.Stderr
var out io.Writer = os.Stdout

func main() {
	args := os.Args
	al := len(args) - 1
	if al != argsLen {
		fmt.Fprintf(errorOut, "Invalid number of arguments. Expected %d, was %d\n", argsLen, al)
		return
	}

	projectPath := args[1]

	err := deps.Fetch(projectPath, "")
	if err != nil {
		fmt.Fprintf(errorOut, "Could not fetch dependencies:\n%v\n", err)
		return
	}
	fmt.Fprintln(out, "Successfully fetched dependencies")
}
