package anker

import (
	"encoding/json"
	"fmt"
	"io"

	"go.uber.org/zap"
)

// GetSites retrieves all sites and their devices for the authenticated user
func (c *Client) GetSites() ([]Site, error) {
	c.debugLog("Fetching site list")
	
	resp, err := c.doRequest("POST", EndpointSiteList, nil, true)
	if err != nil {
		return nil, fmt.Errorf("get sites request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	c.debugLog("GetSites response received", zap.Int("response_size", len(bodyBytes)))

	var sitesResp SiteListResponse
	if err := json.Unmarshal(bodyBytes, &sitesResp); err != nil {
		return nil, fmt.Errorf("failed to decode sites response: %w", err)
	}

	if sitesResp.Code != 0 {
		return nil, fmt.Errorf("get sites failed: %s (code: %d)", sitesResp.Msg, sitesResp.Code)
	}

	// Fetch device details for each site using get_scen_info
	sites := sitesResp.Data.SiteList
	c.logger.Info("Sites retrieved", zap.Int("count", len(sites)))

	for i := range sites {
		devices, err := c.getSceneInfo(sites[i].SiteID)
		if err != nil {
			c.logger.Warn("Failed to get scene info for site",
				zap.String("site_id", sites[i].SiteID),
				zap.Error(err),
			)
			// Don't fail the whole operation, just skip devices for this site
			continue
		}
		sites[i].DeviceList = devices

		c.debugLog("Site devices loaded",
			zap.Int("site_index", i),
			zap.String("site_id", sites[i].SiteID),
			zap.String("site_name", sites[i].SiteName),
			zap.Int("device_count", len(devices)),
		)

		for j, device := range devices {
			c.debugLog("Device details",
				zap.Int("device_index", j),
				zap.String("device_sn", device.DeviceSN),
				zap.String("device_name", device.DeviceName),
				zap.String("device_type", device.DeviceType),
			)
		}
	}

	return sites, nil
}

// getSceneInfo fetches device details for a site
func (c *Client) getSceneInfo(siteID string) ([]Device, error) {
	reqBody := SceneInfoRequest{
		SiteID: siteID,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scene info request: %w", err)
	}

	c.debugLog("Fetching scene info", zap.String("site_id", siteID))

	resp, err := c.doRequest("POST", EndpointSceneInfo, body, true)
	if err != nil {
		return nil, fmt.Errorf("get scene info request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	c.debugLog("GetSceneInfo response received",
		zap.String("site_id", siteID),
		zap.Int("response_size", len(bodyBytes)),
	)

	var sceneResp SceneInfoResponse
	if err := json.Unmarshal(bodyBytes, &sceneResp); err != nil {
		return nil, fmt.Errorf("failed to decode scene info response: %w", err)
	}

	if sceneResp.Code != 0 {
		return nil, fmt.Errorf("get scene info failed: %s (code: %d)", sceneResp.Msg, sceneResp.Code)
	}

	// Combine solar_list and solarbank_list into a single device list
	devices := make([]Device, 0)
	devices = append(devices, sceneResp.Data.SolarList...)
	devices = append(devices, sceneResp.Data.SolarbankInfo.SolarbankList...)

	// Set device type if not present
	for i := range devices {
		if devices[i].DeviceType == "" {
			// Infer type from device_pn or list source
			if devices[i].DevicePN != "" {
				if len(devices[i].DevicePN) >= 3 && devices[i].DevicePN[0:3] == "A17" {
					devices[i].DeviceType = "solarbank"
				} else {
					devices[i].DeviceType = "solar"
				}
			}
		}
	}

	c.debugLog("Scene info processed",
		zap.String("site_id", siteID),
		zap.Int("solar_devices", len(sceneResp.Data.SolarList)),
		zap.Int("solarbank_devices", len(sceneResp.Data.SolarbankInfo.SolarbankList)),
		zap.Int("total_devices", len(devices)),
	)

	return devices, nil
}
