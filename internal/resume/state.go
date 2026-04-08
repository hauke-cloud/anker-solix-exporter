package resume

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

type DeviceState struct {
	LastExportTime time.Time `json:"last_export_time"`
	LastDataTime   time.Time `json:"last_data_time"`
}

type State struct {
	Devices      map[string]DeviceState `json:"devices"`
	LastPollTime time.Time              `json:"last_poll_time"`
	mu           sync.RWMutex
	filePath     string
	logger       *zap.Logger
}

func NewState(filePath string, logger *zap.Logger) (*State, error) {
	s := &State{
		Devices:  make(map[string]DeviceState),
		filePath: filePath,
		logger:   logger,
	}

	// Try to load existing state
	if err := s.Load(); err != nil {
		if !os.IsNotExist(err) {
			logger.Warn("failed to load resume state, starting fresh",
				zap.Error(err),
			)
		} else {
			logger.Info("no previous state found, starting fresh")
		}
	}

	return s, nil
}

func (s *State) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var state struct {
		Devices      map[string]DeviceState `json:"devices"`
		LastPollTime time.Time              `json:"last_poll_time"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	s.Devices = state.Devices
	s.LastPollTime = state.LastPollTime

	s.logger.Info("loaded resume state",
		zap.Int("devices", len(s.Devices)),
		zap.Time("last_poll", s.LastPollTime),
	)

	return nil
}

func (s *State) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file first
	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, s.filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	s.logger.Debug("saved resume state",
		zap.Int("devices", len(s.Devices)),
	)

	return nil
}

func (s *State) GetDeviceState(deviceKey string) (DeviceState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.Devices[deviceKey]
	return state, ok
}

func (s *State) UpdateDeviceState(deviceKey string, lastDataTime time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Devices[deviceKey] = DeviceState{
		LastExportTime: time.Now(),
		LastDataTime:   lastDataTime,
	}
}

func (s *State) UpdatePollTime() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastPollTime = time.Now()
}

func (s *State) GetLastPollTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastPollTime
}

// GetResumeTime returns the time from which to resume data collection for a device
func (s *State) GetResumeTime(deviceKey string, defaultLookback time.Duration) time.Time {
	state, exists := s.GetDeviceState(deviceKey)
	if !exists || state.LastDataTime.IsZero() {
		// No previous data, start from defaultLookback ago
		return time.Now().Add(-defaultLookback)
	}

	// Resume from last known data point
	return state.LastDataTime
}
