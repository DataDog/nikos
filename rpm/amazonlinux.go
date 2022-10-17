package rpm

import (
	"bufio"
	"fmt"
	"os"
	"regexp"

	"github.com/DataDog/nikos/rpm/dnf"
	"github.com/DataDog/nikos/types"
)

func NewAmazonLinux2022Backend(target *types.Target, reposDir string, logger types.Logger) (*RedHatBackend, error) {
	releaseVer, err := extractReleaseVersionFromImageID()
	if err != nil {
		return nil, fmt.Errorf("failed to extract release version: %w", err)
	}

	dnfBackend, err := dnf.NewDnfBackend(releaseVer, reposDir, logger, target)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNF backend: %w", err)
	}

	return &RedHatBackend{
		target:     target,
		logger:     logger,
		dnfBackend: dnfBackend,
	}, nil
}

var imageFilePattern = regexp.MustCompile(`image_file="al2022-\w+-(2022.0.\d{8}).*"`)

func extractReleaseVersionFromImageID() (string, error) {
	imageIDPath := types.HostEtc("image-id")
	f, err := os.Open(imageIDPath)
	if err != nil {
		return "", err
	}

	liner := bufio.NewScanner(f)
	for liner.Scan() {
		if submatches := imageFilePattern.FindStringSubmatch(liner.Text()); submatches != nil {
			return submatches[1], nil
		}
	}

	if err := liner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("image_file entry not found in %s", imageIDPath)
}
