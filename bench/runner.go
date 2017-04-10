package bench

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	goPathVariable   = "GOPATH"
	srcFolder        = "src"
	depFolder        = "vendor"
	goTestFileSuffix = "_test.go"

	cmdName        = "go"
	cmdArgsTest    = "test"
	cmdArgsBench   = "-bench=."
	cmdArgsCount   = "-count=%d"
	cmdArgsNoTests = "-run=^$"

	benchResultUnit = "ns/op"

	defaultPathSize = 20
)

func Run(projectRoot string, wi int, mi int, test string, run int, out csv.Writer) error {
	dirs, err := dirs(projectRoot)
	if err != nil {
		fmt.Printf("Could not retrieve directories\n")
		return err
	}

	cmdCount := fmt.Sprintf(cmdArgsCount, (wi + mi))
	cmdArgs := []string{cmdArgsTest, cmdArgsBench, cmdCount, cmdArgsNoTests}
	env := env(goPath(projectRoot))

	for _, dir := range dirs {
		fmt.Printf("# Execute Benchmarks in Dir: %s\n", dir)

		err = os.Chdir(dir)
		if err != nil {
			return err
		}
		c := exec.Command(cmdName, cmdArgs...)
		c.Env = env

		res, err := c.CombinedOutput()
		if err != nil {
			fmt.Printf("Error while executing command '%s\n", c.Args)
			return fmt.Errorf("%v\n%s", err, res)
		}

		err = parseAndSaveBenchOut(test, run, strings.Replace(dir, projectRoot, "", -1), string(res), out)
		if err != nil {
			return err
		}
	}

	return err
}

func env(goPath string) []string {
	env := os.Environ()
	ret := make([]string, 0, len(env)+1)
	added := false
	goPathDecl := fmt.Sprintf("%s=%s", goPathVariable, goPath)
	for _, e := range env {
		if strings.HasPrefix(e, goPathVariable) {
			ret = append(ret, goPathDecl)
			added = true
		} else {
			ret = append(ret, e)
		}
	}
	if !added {
		ret = append(ret, goPathDecl)
	}
	return ret
}

func parseAndSaveBenchOut(test string, run int, pkg string, res string, out csv.Writer) error {
	resArr := strings.Split(res, "\n")
	for _, benchRes := range resArr {
		benchResArr, err := parseLine(benchRes)
		if err != nil {
			return fmt.Errorf("Could not parse line '%s'", benchRes)
		}
		// benchResArr := strings.Split(benchRes, " ")
		if len(benchResArr) != 4 || strings.TrimSpace(benchResArr[3]) != benchResultUnit {
			// not a benchmark line
			continue
		}
		out.Write([]string{strconv.FormatInt(int64(run), 10),
			test,
			fmt.Sprintf("%s%s", pkg, strings.TrimSpace(benchResArr[0])),
			strings.TrimSpace(benchResArr[2]),
		})
	}
	out.Flush()
	return nil
}

func parseLine(l string) ([]string, error) {
	ret := make([]string, 0, 10)
	b := bytes.NewBuffer([]byte{})
	inWord := false
	for _, c := range l {
		if c != ' ' && c != '	' {
			inWord = true
			_, err := b.WriteRune(c)
			if err != nil {
				return nil, err
			}
		} else if inWord {
			inWord = false
			ret = append(ret, b.String())
			b = bytes.NewBuffer([]byte{})
		}
	}

	// add potential last element
	if inWord {
		ret = append(ret, b.String())
	}

	return ret, nil
}

func goPath(p string) string {
	pathArr := strings.Split(p, string(filepath.Separator))
	var c int
	for i, el := range pathArr {
		if el == srcFolder {
			c = i
			break
		}
	}
	return fmt.Sprintf("/%s", filepath.Join(pathArr[:c]...))
}

func dirs(root string) ([]string, error) {
	paths := make([]string, 0, defaultPathSize)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		if !isValidDir(path) {
			return filepath.SkipDir
		}

		fileInfos, err := ioutil.ReadDir(path)
		if err != nil {
			return err
		}

		// check wether directory contains test files
		validDir := false
		for _, f := range fileInfos {
			if !f.IsDir() {
				if strings.HasSuffix(f.Name(), goTestFileSuffix) {
					validDir = true
					break
				}
			}
		}

		if validDir {
			paths = append(paths, path)
		}

		return err
	})
	return paths, err
}

func isValidDir(path string) bool {
	pathElems := strings.Split(path, string(filepath.Separator))

	for _, el := range pathElems {
		// remove everything from dependencies folder
		if el == depFolder {
			return false
		}
		// remove all hidden folders
		if strings.HasPrefix(el, ".") {
			return false
		}
	}

	return true
}
