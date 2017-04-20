package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"bitbucket.org/sealuzh/goptc/data"
)

func main() {
	path, err := filepath.Abs(os.Args[1])
	if err != nil {
		panic(err)
	}
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	number, err := strconv.Atoi(os.Args[2])
	if err != nil {
		panic(err)
	}

	r := bufio.NewReader(f)

	funs := make([]data.Function, 0, number)
	counter := 0
	for counter <= number {
		l, err := r.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		if counter == 0 {
			// skip first line
			counter++
			continue
		}

		lSplitted := strings.Split(l, " ")

		lArr := strings.Split(lSplitted[0], "/")
		lenArr := len(lArr)
		var recv string
		name := lArr[lenArr-1]
		if strings.Contains(name, ".") {
			fArr := strings.Split(name, ".")
			recv = fArr[0][1 : len(fArr[0])-1]
			name = fArr[1]
		}

		f := data.Function{
			Pkg:      filepath.Join(lArr[:lenArr-2]...),
			File:     lArr[lenArr-2],
			Receiver: recv,
			Name:     name,
		}
		funs = append(funs, f)

		counter++
	}

	json, err := json.Marshal(funs)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(json))
}
