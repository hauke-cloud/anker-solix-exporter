package anker

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"go.uber.org/zap"
)

// GetEnergyData retrieves historical energy data for a device
func (c *Client) GetEnergyData(siteID, deviceSN string, startTime, endTime time.Time) ([]PowerData, error) {
	// Determine device type - for now, assume solarbank if device_sn is provided, otherwise use solar_production
	deviceType := "solarbank"
	if deviceSN == "" {
		deviceType = "solar_production"
	}

	reqBody := EnergyDataRequest{
		SiteID:     siteID,
		DeviceSN:   deviceSN,
		DeviceType: deviceType,
		StartTime:  startTime.Format("2006-01-02"),
		EndTime:    endTime.Format("2006-01-02"),
		Type:       "week", // Use week type to get daily values
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal energy request: %w", err)
	}

	c.debugLog("Fetching energy data",
		zap.String("site_id", siteID),
		zap.String("device_sn", deviceSN),
		zap.String("device_type", deviceType),
		zap.String("start_time", startTime.Format("2006-01-02")),
		zap.String("end_time", endTime.Format("2006-01-02")),
	)

	resp, err := c.doRequest("POST", EndpointEnergyAnalysis, body, true)
	if err != nil {
		return nil, fmt.Errorf("get energy data request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	c.debugLog("GetEnergyData response received",
		zap.String("device_sn", deviceSN),
		zap.Int("response_size", len(bodyBytes)),
	)

	var energyResp EnergyDataResponse
	if err := json.Unmarshal(bodyBytes, &energyResp); err != nil {
		return nil, fmt.Errorf("failed to decode energy response: %w", err)
	}

	if energyResp.Code != 0 {
		return nil, fmt.Errorf("get energy data failed: %s (code: %d)", energyResp.Msg, energyResp.Code)
	}

	c.logger.Info("Energy data retrieved",
		zap.String("device_sn", deviceSN),
		zap.Int("data_points", len(energyResp.Data.Power)),
	)

	return energyResp.Data.Power, nil
}
