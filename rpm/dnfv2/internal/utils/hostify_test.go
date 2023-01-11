package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type hostEtcJoinTestEntry struct {
	name       string
	hostEtcEnv string
	join       []string
	expected   string
}

func TestHostEtcJoin(t *testing.T) {
	testEntries := []hostEtcJoinTestEntry{
		{
			name:       "basic",
			hostEtcEnv: "/host/etc",
			join:       []string{"/etc/yum/vars", "testvar"},
			expected:   "/host/etc/yum/vars/testvar",
		},
		{
			name:       "no env",
			hostEtcEnv: "",
			join:       []string{"/etc/yum/vars", "testvar"},
			expected:   "/etc/yum/vars/testvar",
		},
		{
			name:       "single with env",
			hostEtcEnv: "/b",
			join:       []string{"/etc/a"},
			expected:   "/b/a",
		},
		{
			name:       "single no env",
			hostEtcEnv: "",
			join:       []string{"/etc/a"},
			expected:   "/etc/a",
		},
		{
			name:       "env no prefix",
			hostEtcEnv: "/host/etc",
			join:       []string{"/a/b", "/c"},
			expected:   "/a/b/c",
		},
	}

	for _, entry := range testEntries {
		t.Run(entry.name, func(t *testing.T) {
			t.Setenv("HOST_ETC", entry.hostEtcEnv)
			got := HostEtcJoin(entry.join...)
			assert.Equal(t, entry.expected, got)
		})
	}
}
