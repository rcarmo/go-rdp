package rdp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a less than b", 1, 5, 1},
		{"b less than a", 10, 3, 3},
		{"equal values", 7, 7, 7},
		{"negative numbers", -5, -3, -5},
		{"negative and positive", -5, 5, -5},
		{"zero and positive", 0, 10, 0},
		{"zero and negative", 0, -10, -10},
		{"both zero", 0, 0, 0},
		{"large numbers", 1000000, 999999, 999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}
