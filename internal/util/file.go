package util

import (
	"fmt"
	"github.com/markusressel/fan2go/internal/ui"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func ReadIntFromFile(path string) (value int, err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		ui.Error("File reading error", err)
		return -1, err
	}
	text := string(data)
	text = strings.TrimSpace(text)
	return strconv.Atoi(text)
}

// WriteIntToFile write a single integer to a file.go path
func WriteIntToFile(value int, path string) (err error) {
	f, err := os.OpenFile(path, os.O_SYNC|os.O_WRONLY, 644)
	if err != nil {
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer f.Close()

	valueAsString := fmt.Sprintf("%d", value)
	_, err = f.WriteString(valueAsString)
	return err
}

// FindFilesMatching finds all files in a given directory, matching the given regex
func FindFilesMatching(path string, expr string) []string {
	r, err := regexp.Compile(expr)
	if err != nil {
		log.Fatalf("Cannot compile expr: %s", expr)
	}

	var result []string
	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf(err.Error())
		}

		if !info.IsDir() && r.MatchString(info.Name()) {
			var devicePath string

			// we may need to adjust the path (pwmconfig cite...)
			_, err := os.Stat(path + "/name")
			if os.IsNotExist(err) {
				devicePath = path + "/device"
			} else {
				devicePath = path
			}

			devicePath, err = filepath.EvalSymlinks(devicePath)
			if err != nil {
				panic(err)
			}

			result = append(result, devicePath)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	return result
}
