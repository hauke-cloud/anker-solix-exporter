package anker

// API endpoints used by the Anker Solix API client.
// All endpoints are relative to the base URL defined in client.go

const (
	// Authentication endpoints
	EndpointLogin = "/passport/login"

	// Site and device endpoints
	EndpointSiteList     = "/power_service/v1/site/get_site_list"
	EndpointSceneInfo    = "/power_service/v1/site/get_scen_info"

	// Energy data endpoints
	EndpointEnergyAnalysis = "/power_service/v1/site/energy_analysis"
)
