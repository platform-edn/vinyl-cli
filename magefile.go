//go:build mage
// +build mage

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"
)

var (
	baseDir     = getMageDir()
	internalDir = fmt.Sprintf(filepath.Join(baseDir, "internal"))
)

func getMageDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	return dir
}

// starts the vinyl server
func Start() error {
	cmd := filepath.Join(baseDir, "cmd", "server")

	err := os.Chdir(cmd)
	if err != nil {
		return fmt.Errorf("Start: %s", err)
	}
	defer os.Chdir(baseDir)

	err = sh.Run("go", "run", "main.go")
	if err != nil {
		return fmt.Errorf("Start: %s", err)
	}

	return nil
}

// updates grpc boilerplate
func Proto() error {
	protodefs := []string{
		"record",
	}

	for _, def := range protodefs {
		protopath := filepath.Join(baseDir, "..", "protobuf", def)

		files, err := ioutil.ReadDir(protopath)
		if err != nil {
			return fmt.Errorf("could not get files in %s: %s", baseDir, err)
		}

		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".proto") {

				err = sh.Run("protoc", "--proto_path="+protopath, "--go-grpc_out=.", file.Name())
				if err != nil {
					return fmt.Errorf("could not create go proto files: %s", err)
				}

				err = sh.Run("protoc", "--proto_path="+protopath, "--go_out=.", file.Name())
				if err != nil {
					return fmt.Errorf("could not create go proto files: %s", err)
				}
			}
		}
	}

	return nil
}

//runs race tests
func Race() error {
	coverage := "coverage.out"

	err := sh.Run("go", "test", "-race", "-covermode=atomic", fmt.Sprintf("-coverprofile=%s", coverage), filepath.Join(internalDir, "..."))
	if err != nil {
		return fmt.Errorf("Race: %s", err)
	}

	file, err := os.Open(filepath.Join(baseDir, coverage))
	if err != nil {
		return fmt.Errorf("Race: %s", err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var txtlines []string

	for scanner.Scan() {
		txtlines = append(txtlines, scanner.Text())
	}

	file.Close()

	output := ""
	for _, line := range txtlines {
		if !strings.Contains(line, ".pb.") {
			if output == "" {
				output = line
			} else {
				output = fmt.Sprintf("%s\n%s", output, line)
			}
		}
	}

	err = ioutil.WriteFile(filepath.Join(baseDir, coverage), []byte(output), 0755)
	if err != nil {
		return fmt.Errorf("Race: %s", err)
	}

	return nil
}

//creates mocks for internal interfaces
func Mock() error {
	err := filepath.Walk(internalDir, mockWalkFunction)
	if err != nil {
		return fmt.Errorf("Mock: %w", err)
	}

	return nil
}

func mockWalkFunction(subDir string, info os.FileInfo, err error) error {
	if err != nil {
		return fmt.Errorf("mockWalkFunction: %w", err)
	}

	if subDir == internalDir {
		return nil
	}

	isDir, err := isDirectory(subDir)
	if err != nil {
		return fmt.Errorf("mockWalkFunction: %w", err)
	}

	if isDir {
		err = createMocks(subDir)
		if err != nil {
			return fmt.Errorf("mockWalkFunction: %w", err)
		}
	}

	return nil
}

func createMocks(subDir string) error {
	os.Chdir(subDir)

	err := sh.Run("mockery", "--all", "--with-expecter", "--case", "underscore")
	if err != nil {
		return fmt.Errorf("MakeMocks: %w", err)
	}

	return nil
}

func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fileInfo.IsDir(), nil
}
