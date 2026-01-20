package rdp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrUnsupportedRequestedProtocol(t *testing.T) {
	err := ErrUnsupportedRequestedProtocol

	assert.NotNil(t, err)
	assert.Equal(t, "unsupported requested protocol", err.Error())
}

func TestErrUnsupportedRequestedProtocol_ErrorInterface(t *testing.T) {
	err := error(ErrUnsupportedRequestedProtocol)

	assert.NotNil(t, err)
	assert.IsType(t, ErrUnsupportedRequestedProtocol, err)
}
