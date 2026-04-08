package anker

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

// GetCurrentMeasurements fetches current/real-time measurements from scene info
func (c *Client) GetCurrentMeasurements(siteID string) ([]Measurement, error) {
	reqBody := SceneInfoRequest{
		SiteID: siteID,
	}

	c.debugLog("Fetching current measurements", zap.String("site_id", siteID))

	var sceneResp SceneInfoResponse
	if err := c.handler.execute("POST", EndpointSceneInfo, reqBody, &sceneResp, true); err != nil {
		return nil, fmt.Errorf("get scene info failed: %w", err)
	}

	now := time.Now()
	measurements := make([]Measurement, 0)

	// Process solarbank devices
	for _, device := range sceneResp.Data.SolarbankInfo.SolarbankList {
		measurement := Measurement{
			Timestamp:  now,
			SiteID:     siteID,
			SiteName:   sceneResp.Data.SiteName,
			DeviceSN:   device.DeviceSN,
			DeviceName: device.DeviceName,
			DeviceType: "solarbank",
		}

		// Parse string values to float64
		if device.PhotovoltaicPower != "" {
			if val, err := parseFloat(device.PhotovoltaicPower); err == nil {
				measurement.SolarPower = val
			}
		}
		if device.OutputPower != "" {
			if val, err := parseFloat(device.OutputPower); err == nil {
				measurement.OutputPower = val
			}
		}
		if device.BatteryPower != "" {
			if val, err := parseFloat(device.BatteryPower); err == nil {
				measurement.BatterySoC = val
			}
		}
		if device.BatChargePower != "" {
			if val, err := parseFloat(device.BatChargePower); err == nil {
				measurement.BatteryPower = val
			}
		}

		measurements = append(measurements, measurement)
	}

	// Process solar devices
	for _, device := range sceneResp.Data.SolarList {
		measurement := Measurement{
			Timestamp:  now,
			SiteID:     siteID,
			SiteName:   sceneResp.Data.SiteName,
			DeviceSN:   device.DeviceSN,
			DeviceName: device.DeviceName,
			DeviceType: "solar",
		}

		if device.PhotovoltaicPower != "" {
			if val, err := parseFloat(device.PhotovoltaicPower); err == nil {
				measurement.SolarPower = val
			}
		}

		measurements = append(measurements, measurement)
	}

	c.logger.Info("Current measurements retrieved",
		zap.String("site_id", siteID),
		zap.Int("count", len(measurements)),
	)

	return measurements, nil
}
