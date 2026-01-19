package rdp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_SetRemoteApp(t *testing.T) {
	tests := []struct {
		name            string
		app             string
		args            string
		workingDir      string
		initialChannels []string
	}{
		{
			name:            "basic remote app",
			app:             "notepad.exe",
			args:            "",
			workingDir:      "",
			initialChannels: []string{},
		},
		{
			name:            "remote app with args",
			app:             "cmd.exe",
			args:            "/c dir",
			workingDir:      "C:\\Windows\\System32",
			initialChannels: []string{},
		},
		{
			name:            "with existing channels",
			app:             "calc.exe",
			args:            "",
			workingDir:      "",
			initialChannels: []string{"cliprdr", "rdpdr"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				channels: tt.initialChannels,
			}

			client.SetRemoteApp(tt.app, tt.args, tt.workingDir)

			// Verify remote app is set
			assert.NotNil(t, client.remoteApp)
			assert.Equal(t, tt.app, client.remoteApp.App)
			assert.Equal(t, tt.args, client.remoteApp.Args)
			assert.Equal(t, tt.workingDir, client.remoteApp.WorkingDir)

			// Verify "rail" channel is added
			assert.Contains(t, client.channels, "rail")

			// Verify rail state is set to uninitialized
			assert.Equal(t, RailStateUninitialized, client.railState)

			// Verify original channels are preserved
			for _, ch := range tt.initialChannels {
				assert.Contains(t, client.channels, ch)
			}
		})
	}
}

func TestClient_SetRemoteApp_MultipleCalls(t *testing.T) {
	client := &Client{
		channels: []string{},
	}

	// First call
	client.SetRemoteApp("app1.exe", "arg1", "dir1")
	assert.Equal(t, "app1.exe", client.remoteApp.App)
	assert.Len(t, client.channels, 1)

	// Second call should override
	client.SetRemoteApp("app2.exe", "arg2", "dir2")
	assert.Equal(t, "app2.exe", client.remoteApp.App)
	assert.Equal(t, "arg2", client.remoteApp.Args)
	assert.Equal(t, "dir2", client.remoteApp.WorkingDir)
	// Note: "rail" channel gets added again
	assert.Len(t, client.channels, 2)
}
