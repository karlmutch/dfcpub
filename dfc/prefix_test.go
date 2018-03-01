// Package dfc provides distributed file-based cache with Amazon and Google Cloud backends.
//
// Example run:
// 	go test -v -run=prefix -args -numfiles=50 -prefix="filter/a"
//
/*
 * Copyright (c) 2018, NVIDIA CORPORATION. All rights reserved.
 *
 */
package dfc_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/NVIDIA/dfcpub/dfc"
)

const (
	prefixDir = "filter"
)

var (
	prefix           string
	prefixFileNumber int
)

func init() {
	flag.StringVar(&prefix, "prefix", "", "Object name prefix")
}

// if the prefix flag is set via command line the test looks only for the prefix
// and checks if the number of items equals the number of files with
// the names starting with the prefix;
// otherwise, the test creates (PUT) random files and executes 'a*' through 'z*' listings.
func Test_prefix(t *testing.T) {
	flag.Parse()
	fmt.Printf("Looking for files with prefix [%s]\n", prefix)

	if err := dfc.CreateDir(fmt.Sprintf("%s/%s", baseDir, prefixDir)); err != nil {
		t.Fatalf("Failed to create dir %s/%s, err: %v", baseDir, prefixDir, err)
	}

	prefixFileNumber = numfiles

	prefixCreateFiles(t)
	prefixLookup(t)
	prefixCleanup(t)
}

func numberOfFilesWithPrefix(fileNames []string, namePrefix string, commonDir string) int {
	numFiles := 0
	for _, fileName := range fileNames {
		if commonDir != "" {
			fileName = fmt.Sprintf("%s/%s", commonDir, fileName)
		}
		if strings.HasPrefix(fileName, namePrefix) {
			numFiles++
		}
	}
	return numFiles
}

func prefixCreateFiles(t *testing.T) {
	fmt.Printf("Creating files...\n")
	src := rand.NewSource(baseseed + 1000)
	random := rand.New(src)
	buf := make([]byte, blocksize)
	fileNames = make([]string, 0, prefixFileNumber)
	errch := make(chan error, 10)
	var wg = &sync.WaitGroup{}

	for i := 0; i < prefixFileNumber; i++ {
		fileName := fastRandomFilename(random)
		keyName := fmt.Sprintf("%s/%s", prefixDir, fileName)
		filePath := fmt.Sprintf("%s/%s", baseDir, keyName)
		tlogf("Creating file at: %s\n", filePath)
		if _, err := writeRandomData(filePath, buf, int(fileSize), random); err != nil {
			fmt.Fprintf(os.Stdout, "File create fail: %v\n", err)
			t.Error(err)
			return
		}

		wg.Add(1)
		go put(filePath, clibucket, keyName, wg, errch, true)
		fileNames = append(fileNames, fileName)
	}
	wg.Wait()
	select {
	case e := <-errch:
		fmt.Fprintf(os.Stdout, "PUT FAIL: %s\n", e)
		t.Fail()
	default:
	}
}

func prefixLookupOne(t *testing.T) {
	fmt.Printf("Looking up for files than names start with %s\n", prefix)
	var msg = &dfc.GetMsg{GetPrefix: prefix}
	jsbytes, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("Unexpected json-marshal failure, err: %v", err)
		return
	}

	numFiles := 0
	objList := listbucket(t, clibucket, jsbytes)
	for _, entry := range objList.Entries {
		tlogf("Found object: %s\n", entry.Name)
		numFiles++
	}

	realNumFiles := numberOfFilesWithPrefix(fileNames, prefix, prefixDir)
	if realNumFiles == numFiles {
		fmt.Printf("Total files with prefix found: %v\n", numFiles)
	} else {
		t.Errorf("Expected number of files with prefix '%s' is %v but found %v files", prefix, realNumFiles, numFiles)
	}
}

func prefixLookupDefault(t *testing.T) {
	fmt.Printf("Looking up for files in alphabetic order\n")

	letters := "abcdefghijklmnopqrstuvwxyz"
	for i := 0; i < len(letters); i++ {
		key := letters[i : i+1]
		lookFor := fmt.Sprintf("%s/%s", prefixDir, key)
		var msg = &dfc.GetMsg{GetPrefix: lookFor}
		jsbytes, err := json.Marshal(msg)
		if err != nil {
			t.Errorf("Unexpected json-marshal failure, err: %v", err)
			return
		}

		objList := listbucket(t, clibucket, jsbytes)
		numFiles := len(objList.Entries)
		realNumFiles := numberOfFilesWithPrefix(fileNames, key, prefix)

		if numFiles == realNumFiles {
			if numFiles != 0 {
				fmt.Printf("Found %v files starting with '%s'\n", numFiles, key)
			}
		} else {
			t.Errorf("Expected number of files with prefix '%s' is %v but found %v files", key, realNumFiles, numFiles)
		}
	}
}

func prefixLookup(t *testing.T) {
	if prefix == "" {
		prefixLookupDefault(t)
	} else {
		prefixLookupOne(t)
	}
}

func prefixCleanup(t *testing.T) {
	fmt.Printf("Cleaning up...\n")
	errch := make(chan error, 10)
	var wg = &sync.WaitGroup{}

	for _, fileName := range fileNames {
		keyName := fmt.Sprintf("%s/%s", prefixDir, fileName)
		wg.Add(1)
		go del(clibucket, keyName, wg, errch, true)

		if err := os.Remove(fmt.Sprintf("%s/%s", baseDir, keyName)); err != nil {
			fmt.Printf("Failed to delete file: %v\n", err)
			t.Fail()
		}
	}
	wg.Wait()

	select {
	case e := <-errch:
		fmt.Fprintf(os.Stdout, "DEL FAIL: %s\n", e)
		t.Fail()
	default:
	}
}