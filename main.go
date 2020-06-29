package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const metadataURL = "https://www.tibia.com/launcher/tibiametadata.json"

var wg *sync.WaitGroup

func main() {
	metadataBytes, err := downloadJSON(metadataURL, "")
	if err != nil {
		log.Fatalln(err)
	}

	var metadata TMetadata
	err = json.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		log.Fatalln(err)
	}

	for _, build := range metadata.Launcher.Builds {
		downloadBuild(build)
	}

	for _, client := range metadata.Packages {
		for _, build := range client.Builds {
			downloadBuild(build)
		}
	}

	// wg.Wait()

}

func toPath(URL string) string {
	indexPath := strings.Index(URL, "launcher")
	if indexPath == -1 {
		indexPath = 0
	}

	return URL[indexPath:]
}

func downloadJSON(URL, destination string) ([]byte, error) {
	rawBytes, err := getJSON(URL)
	if err != nil {
		return nil, err
	}

	var jsonAbs string
	if destination == "" {
		jsonFile := toPath(URL)
		jsonAbs, _ = filepath.Abs(jsonFile)
	} else {
		jsonAbs, _ = filepath.Abs(filepath.Join(destination, toPath(URL)))
	}

	// create dir if not exists
	if _, err := os.Stat(jsonAbs); err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(filepath.Dir(jsonAbs), 0755)
			if err != nil {
				return nil, err
			}
		}
	}

	rawBytes = bytes.Trim(rawBytes, "\x00")
	ioutil.WriteFile(jsonAbs, rawBytes, 0644)
	return rawBytes, nil
}

func downloadFile(URL, distributionFolder string) {
	URL = strings.TrimSpace(strings.Replace(URL, "https:/static", "https://static", 1))
	downloadJSON(URL, distributionFolder)
}

func downloadBuild(build Builds) {
	distributionFolder := fmt.Sprintf("%s-%s-current", build.OS, build.Architecture)
	if build.URL != "" {
		downloadURL(build.URL, distributionFolder)
	}

	if build.AssetsURL != "" {
		downloadURL(build.AssetsURL, distributionFolder)
	}
}

func downloadURL(URL, distributionFolder string) {
	rawBytes, err := downloadJSON(URL, distributionFolder)
	if err != nil {
		log.Fatalln(err)
	}

	var files TStructure
	err = json.Unmarshal(rawBytes, &files)
	if err != nil {
		log.Fatalln(err)
	}

	for _, file := range files.Files {
		fileURL := filepath.Join(strings.TrimRight(URL, "/package.json"), file.URL)
		downloadFile(fileURL, distributionFolder)
	}
}

func getJSON(url string) ([]byte, error) {
	log.Printf("downloading %s\n", url)

	client := http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	return buf.Bytes(), nil
}

type TMetadata struct {
	Launcher       Launcher   `json:"launcher"`
	Packages       []Packages `json:"packages"`
	HardwareSurvey string     `json:"hardwaresurvey"`
	Hints          string     `json:"hints"`
}

type Launcher struct {
	Name   string   `json:"name"`
	Builds []Builds `json:"builds"`
}

type Builds struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	URL          string `json:"url"`
	AssetsURL    string `json:"assetsurl"`
}
type Packages struct {
	Name   string   `json:"name"`
	Builds []Builds `json:"builds"`
}

type TStructure struct {
	Files []File `json:"files"`
}

type File struct {
	URL          string `json:"url"`
	UnpackedHash string `json:"unpackedhash"`
	UnpackedSize int    `json:"unpackedsize"`
	PackedHash   string `json:"packedhash"`
	PackedSize   int    `json:"packedsize"`
	LocalFile    string `json:"localfile"`
	Executable   bool   `json:"executable,omitempty"`
}
