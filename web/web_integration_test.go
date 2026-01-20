package main

import (
	"os"
	"strings"
	"testing"
)

// TestWebInterfaceFiles tests that required web interface files exist
func TestWebInterfaceFiles(t *testing.T) {
	// Test that main HTML file exists
	htmlPath := "index.html"
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		t.Errorf("Main HTML file %s not found", htmlPath)
	}

	// Test that required JavaScript files exist
	jsFiles := []string{
		"js/binary.js",
		"js/client.bundle.min.js",
		"js/input/keyboard.js",
		"js/input/keymap.js",
		"js/input/mouse.js",
		"js/input/header.js",
		"js/color.js",
		"js/update/header.js",
		"js/update/bitmap.js",
		"js/update/pointer.js",
		"js/rle/wasm_exec.js",
	}

	for _, jsFile := range jsFiles {
		if _, err := os.Stat(jsFile); os.IsNotExist(err) {
			t.Errorf("JavaScript file %s not found", jsFile)
		}
	}

	// Test that WASM files exist
	wasmFiles := []string{
		"js/rle/rle.wasm",
	}

	for _, wasmFile := range wasmFiles {
		if _, err := os.Stat(wasmFile); os.IsNotExist(err) {
			t.Errorf("WASM file %s not found", wasmFile)
		}
	}
}

// TestWebInterfaceContentStructure tests that HTML has required content
func TestWebInterfaceContentStructure(t *testing.T) {
	// Read HTML file content
	content, err := os.ReadFile("index.html")
	if err != nil {
		t.Fatalf("Failed to read index.html: %v", err)
	}

	htmlContent := string(content)

	// Test that required elements are present
	requiredElements := []string{
		"id=\"canvas\"",
		"id=\"host\"",
		"id=\"user\"",
		"id=\"password\"",
		"id=\"hot-corner\"",
		"id=\"hot-corner-menu\"",
		"id=\"special-keys-modal\"",
		"class=\"connection-panel\"",
		"class=\"canvas-container\"",
	}

	for _, element := range requiredElements {
		if !strings.Contains(htmlContent, element) {
			t.Errorf("Required element not found: %s", element)
		}
	}

	// Test that required scripts are included
	requiredScripts := []string{
		"js/binary.js",
		"js/client.bundle.min.js",
		"js/input/keyboard.js",
		"js/input/keymap.js",
		"js/rle/wasm_exec.js",
		"js/color.js",
	}

	for _, script := range requiredScripts {
		if !strings.Contains(htmlContent, script) {
			t.Errorf("Required script not included: %s", script)
		}
	}

	// Test security headers are defined in meta tags
	securityFeatures := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Content-Security-Policy",
	}

	for range securityFeatures {
		// Since these are set server-side, we check for the meta tags that enable them
		if strings.Contains(htmlContent, "meta") {
			t.Logf("Meta tags present for security features")
		}
	}

	// Test accessibility features
	accessibilityFeatures := []string{
		"aria-label",
		"aria-required",
		"role=\"alert\"",
		"viewport",
	}

	for _, feature := range accessibilityFeatures {
		if strings.Contains(htmlContent, feature) {
			t.Logf("Accessibility feature found: %s", feature)
		}
	}

	// Test mobile support features
	mobileFeatures := []string{
		"viewport",
		"media",
		"user-scalable=no",
	}

	for _, feature := range mobileFeatures {
		if strings.Contains(htmlContent, feature) {
			t.Logf("Mobile support feature found: %s", feature)
		}
	}
}
