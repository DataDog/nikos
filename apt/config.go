// Copyright (C) 2017  Arduino AG (http://www.arduino.cc/)

// Extracted from https://github.com/arduino/go-apt-client
// Fixes:
// - fix option parsing when no component is provided

package apt

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
)

// RepositoryList is an array of Repository definitions
type RepositoryList []*Repository

// Repository contains metadata about a repository installed in the system
type Repository struct {
	Enabled      bool
	SourceRepo   bool
	Options      string
	URI          string
	Distribution string
	Components   string
	Comment      string

	configFile string
}

var aptConfigLineRegexp = regexp.MustCompile(`^(# )?(deb|deb-src)(?: \[(.*)\])? ([^ \[]+) ([^ ]+)(?: ([^#\n]+))?(?: +# *(.*))?$`)

func parseAPTConfigLine(line string) *Repository {
	match := aptConfigLineRegexp.FindAllStringSubmatch(line, -1)
	if len(match) == 0 || len(match[0]) < 6 {
		return nil
	}
	fields := match[0]
	//fmt.Printf("%+v\n", fields)
	return &Repository{
		Enabled:      fields[1] != "# ",
		SourceRepo:   fields[2] == "deb-src",
		Options:      fields[3],
		URI:          fields[4],
		Distribution: fields[5],
		Components:   fields[6],
		Comment:      fields[7],
	}
}

func parseAPTConfigFile(configPath string) (RepositoryList, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("Reading %s: %s", configPath, err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))

	res := RepositoryList{}
	for scanner.Scan() {
		line := scanner.Text()
		repo := parseAPTConfigLine(line)
		//fmt.Printf("%+v\n", repo)
		if repo != nil {
			repo.configFile = configPath
			res = append(res, repo)
		}
	}
	return res, nil
}

// parseAPTConfigFolder scans an APT config folder (usually /etc/apt) to
// get information about all configured repositories, it scans also
// "source.list.d" subfolder to find all the "*.list" files.
func parseAPTConfigFolder(folderPath string) (RepositoryList, error) {
	sources := []string{filepath.Join(folderPath, "sources.list")}

	sourcesFolder := filepath.Join(folderPath, "sources.list.d")
	list, err := ioutil.ReadDir(sourcesFolder)
	if err != nil {
		return nil, fmt.Errorf("Reading %s folder: %s", sourcesFolder, err)
	}
	for _, l := range list {
		if strings.HasSuffix(l.Name(), ".list") {
			sources = append(sources, filepath.Join(sourcesFolder, l.Name()))
		}
	}

	res := RepositoryList{}
	for _, source := range sources {
		repos, err := parseAPTConfigFile(source)
		if err != nil {
			return nil, fmt.Errorf("Parsing %s: %s", source, err)
		}
		res = append(res, repos...)
	}
	return res, nil
}
