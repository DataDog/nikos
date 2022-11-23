//go:build !linux

package backend

func ComputeBuiltinVariables(releaseVersion string) (map[string]string, error) {
	if releaseVersion == "" {
		releaseVersion = "2022.0.20220928"
	}

	return map[string]string{
		"arch":       "aarch64",
		"basearch":   "aarch64",
		"releasever": releaseVersion,
	}, nil
}
