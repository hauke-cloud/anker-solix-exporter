package anker

import (
	"fmt"
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

	c.debugLog("Fetching energy data",
		zap.String("site_id", siteID),
		zap.String("device_sn", deviceSN),
		zap.String("device_type", deviceType),
		zap.String("start_time", startTime.Format("2006-01-02")),
		zap.String("end_time", endTime.Format("2006-01-02")),
	)

	var energyResp EnergyDataResponse
	if err := c.handler.execute("POST", EndpointEnergyAnalysis, reqBody, &energyResp, true); err != nil {
		return nil, fmt.Errorf("get energy data failed: %w", err)
	}

	c.logger.Info("Energy data retrieved",
		zap.String("device_sn", deviceSN),
		zap.Int("data_points", len(energyResp.Data.Power)),
	)

	return energyResp.Data.Power, nil
}
