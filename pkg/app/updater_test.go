package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isNewer(t *testing.T) {
	tests := []struct {
		current, latest string
		want            bool
	}{
		{"v0.1.0", "v0.2.0", true},
		{"v0.2.0", "v0.1.0", false},
		{"v0.1.0", "v0.1.0", false},
		{"v1.0.0", "v1.0.1", true},
		{"v1.0.0", "v2.0.0", true},
		{"0.1.0", "v0.2.0", true},    // no v prefix on current
		{"v0.1.0", "0.2.0", true},    // no v prefix on latest
		{"dev", "v0.1.0", false},     // invalid current
		{"v0.1.0", "invalid", false}, // invalid latest
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.latest, func(t *testing.T) {
			assert.Equal(t, tt.want, isNewer(tt.current, tt.latest))
		})
	}
}
