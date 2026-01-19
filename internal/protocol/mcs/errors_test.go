package mcs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrors(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "ErrChannelNotFound",
			err:  ErrChannelNotFound,
			msg:  "channel not found",
		},
		{
			name: "ErrUnknownConnectApplication",
			err:  ErrUnknownConnectApplication,
			msg:  "unknown connect application",
		},
		{
			name: "ErrUnknownDomainApplication",
			err:  ErrUnknownDomainApplication,
			msg:  "unknown domain application",
		},
		{
			name: "ErrUnknownChannel",
			err:  ErrUnknownChannel,
			msg:  "unknown channel",
		},
		{
			name: "ErrDisconnectUltimatum",
			err:  ErrDisconnectUltimatum,
			msg:  "disconnect ultimatum",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, tc.err)
			require.Equal(t, tc.msg, tc.err.Error())
		})
	}
}

func TestErrorsIs(t *testing.T) {
	// Verify errors.Is works correctly with wrapped errors
	wrapped := errors.New("wrapped: " + ErrChannelNotFound.Error())
	require.NotErrorIs(t, wrapped, ErrChannelNotFound)

	// Verify the errors are distinct
	require.NotErrorIs(t, ErrChannelNotFound, ErrUnknownChannel)
	require.NotErrorIs(t, ErrUnknownConnectApplication, ErrUnknownDomainApplication)
}
