package resume

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewState(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "test_state.json")

	logger := zap.NewNop()
	state, err := NewState(stateFile, logger)
	if err != nil {
		t.Fatalf("NewState failed: %v", err)
	}

	if state.filePath != stateFile {
		t.Errorf("Expected filePath %s, got %s", stateFile, state.filePath)
	}

	if len(state.Devices) != 0 {
		t.Errorf("Expected empty devices map, got %d devices", len(state.Devices))
	}
}

func TestStateSaveLoad(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "test_state.json")

	logger := zap.NewNop()
	state, _ := NewState(stateFile, logger)

	// Add some data
	deviceKey := "site1:device1"
	testTime := time.Now()
	state.UpdateDeviceState(deviceKey, testTime)
	state.UpdatePollTime()

	// Save
	if err := state.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Fatal("State file was not created")
	}

	// Load into new state
	newState, _ := NewState(stateFile, logger)

	// Verify data
	deviceState, exists := newState.GetDeviceState(deviceKey)
	if !exists {
		t.Fatal("Device state not found after load")
	}

	// Allow small time difference due to JSON serialization
	if deviceState.LastDataTime.Unix() != testTime.Unix() {
		t.Errorf("Expected LastDataTime %v, got %v", testTime, deviceState.LastDataTime)
	}
}

func TestGetResumeTime(t *testing.T) {
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "test_state.json")

	logger := zap.NewNop()
	state, _ := NewState(stateFile, logger)

	deviceKey := "site1:device1"
	defaultLookback := 24 * time.Hour

	// Test with no previous state
	resumeTime := state.GetResumeTime(deviceKey, defaultLookback)
	expectedTime := time.Now().Add(-defaultLookback)

	// Allow 1 second difference
	if resumeTime.Sub(expectedTime).Abs() > time.Second {
		t.Errorf("Expected resume time around %v, got %v", expectedTime, resumeTime)
	}

	// Add device state
	lastDataTime := time.Now().Add(-1 * time.Hour)
	state.UpdateDeviceState(deviceKey, lastDataTime)

	// Test with existing state
	resumeTime = state.GetResumeTime(deviceKey, defaultLookback)
	if resumeTime.Unix() != lastDataTime.Unix() {
		t.Errorf("Expected resume time %v, got %v", lastDataTime, resumeTime)
	}
}
