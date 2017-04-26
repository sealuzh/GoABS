package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"testing"
	"time"
)

const (
	maxFiles = 30
	waitTime = 0
)

func TestClearTmpFolder(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
		return
	}
	p := filepath.Join(u.HomeDir, "tmp/clear_test/")
	err = os.MkdirAll(p, os.ModePerm)
	if err != nil {
		t.Fatal(err)
		return
	}

	defer func() {
		err = os.RemoveAll(p)
		if err != nil {
			t.Fatal(err)
		}
	}()

	clear := clearTmpFolder(p)
	n := populateTmpFolder(p)
	time.Sleep(waitTime * time.Second)
	elems, err := ioutil.ReadDir(p)
	if err != nil {
		t.Fatal(err)
		return
	}

	fmt.Println(elems)
	if len(elems) != n {
		t.Fatal(fmt.Sprintf("Invalid number of elements after populate: %d expected, was %d", n, len(elems)))
		return
	}

	clear()

	elems, err = ioutil.ReadDir(p)
	if err != nil {
		t.Fatal(err)
		return
	}
	fmt.Println(elems)

	if len(elems) != 0 {
		t.Fatal(fmt.Sprintf("Invalid number of elements after clear: 0 expected, was %d", len(elems)))
		return
	}

	n = populateTmpFolder(p)
	time.Sleep(waitTime * time.Second)
	elems, err = ioutil.ReadDir(p)
	if err != nil {
		t.Fatal(err)
		return
	}
	fmt.Println(elems)

	if len(elems) != n {
		t.Fatal(fmt.Sprintf("Invalid number of elements after populate: %d expected, was %d", n, len(elems)))
		return
	}

	clear()

	elems, err = ioutil.ReadDir(p)
	if err != nil {
		t.Fatal(err)
		return
	}
	fmt.Println(elems)

	if len(elems) != 0 {
		t.Fatal(fmt.Sprintf("Invalid number of elements after clear: 0 expected, was %d", len(elems)))
		return
	}
}

func populateTmpFolder(path string) int {
	s := rand.NewSource(time.Now().UnixNano())
	n := int(s.Int63()) % maxFiles
	for i := 0; i < n; i++ {
		p := filepath.Join(path, fmt.Sprintf("folder%d", i))
		err := os.Mkdir(p, os.ModePerm)
		if err != nil {
			panic(fmt.Sprintf("Could not create folder: %s", p))
		}
	}
	return n
}
