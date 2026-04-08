package anker

import (
	"fmt"

	"go.uber.org/zap"
)

// GetSites retrieves all sites and their devices for the authenticated user
func (c *Client) GetSites() ([]Site, error) {
	c.debugLog("Fetching site list")
	
	var sitesResp SiteListResponse
	if err := c.handler.execute("POST", EndpointSiteList, nil, &sitesResp, true); err != nil {
		return nil, fmt.Errorf("get sites failed: %w", err)
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

	c.debugLog("Fetching scene info", zap.String("site_id", siteID))

	var sceneResp SceneInfoResponse
	if err := c.handler.execute("POST", EndpointSceneInfo, reqBody, &sceneResp, true); err != nil {
		return nil, fmt.Errorf("get scene info failed: %w", err)
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
