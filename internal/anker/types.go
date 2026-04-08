package anker

import "time"

// API request/response types

type LoginRequest struct {
	AB               string                 `json:"ab"`
	ClientSecretInfo map[string]interface{} `json:"client_secret_info"`
	Enc              int                    `json:"enc"`
	Email            string                 `json:"email"`
	Password         string                 `json:"password"`
	TimeZone         int64                  `json:"time_zone"`
	Transaction      string                 `json:"transaction"`
}

type LoginResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		UserID    string `json:"user_id"`
		AuthToken string `json:"auth_token"`
		Email     string `json:"email"`
		NickName  string `json:"nick_name"`
		Country   string `json:"country"`
	} `json:"data"`
}

type SiteListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		SiteList []Site `json:"site_list"`
	} `json:"data"`
}

type SceneInfoRequest struct {
	SiteID string `json:"site_id"`
}

type SceneInfoResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		SiteID        string `json:"site_id"`
		SiteName      string `json:"site_name"`
		SolarList     []Device `json:"solar_list"`
		SolarbankInfo struct {
			SolarbankList []Device `json:"solarbank_list"`
		} `json:"solarbank_info"`
	} `json:"data"`
}

type EnergyDataRequest struct {
	SiteID     string `json:"site_id"`
	DeviceSN   string `json:"device_sn"`
	DeviceType string `json:"device_type"` // "solarbank", "solar_production", etc.
	Type       string `json:"type"`        // "day", "week", "month", "year"
	StartTime  string `json:"start_time"`  // Format: "2006-01-02"
	EndTime    string `json:"end_time"`    // Format: "2006-01-02"
}

type EnergyDataResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Power                   []PowerData `json:"power"`
		ChargeTotal             string      `json:"charge_total"`
		DischargeTotal          string      `json:"discharge_total"`
		BatteryDischargingTotal string      `json:"battery_discharging_total"`
		BatteryToHomeTotal      string      `json:"battery_to_home_total"`
		SolarToHomeTotal        string      `json:"solar_to_home_total"`
		SolarToBatteryTotal     string      `json:"solar_to_battery_total"`
		SolarToGridTotal        string      `json:"solar_to_grid_total"`
		GridToBatteryTotal      string      `json:"grid_to_battery_total"`
		SolarTotal              string      `json:"solar_total"`
	} `json:"data"`
}

// Domain models

type Site struct {
	SiteID     string   `json:"site_id"`
	SiteName   string   `json:"site_name"`
	SiteAdmin  bool     `json:"site_admin"`
	DeviceList []Device `json:"device_list,omitempty"`
}

type Device struct {
	DeviceSN          string  `json:"device_sn"`
	DeviceName        string  `json:"device_name"`
	DevicePN          string  `json:"device_pn"`
	DeviceType        string  `json:"device_type,omitempty"`
	BatteryPower      string  `json:"battery_power,omitempty"`
	ChargingPower     string  `json:"charging_power,omitempty"`
	OutputPower       string  `json:"output_power,omitempty"`
	PhotovoltaicPower string  `json:"photovoltaic_power,omitempty"`
	BatChargePower    string  `json:"bat_charge_power,omitempty"`
	BatDischargePower string  `json:"bat_discharge_power,omitempty"`
	Status            string  `json:"status,omitempty"`
	// Legacy fields for compatibility
	DevicePowerW float64 `json:"device_power_w,omitempty"`
	GenerateTime int64   `json:"generate_time,omitempty"`
}

type PowerData struct {
	Time  string  `json:"time"`  // Date string in format "2006-01-02"
	Value string  `json:"value"` // Energy value as string
	Rods  *string `json:"rods"`  // Optional field
}

type Measurement struct {
	Timestamp    time.Time
	SiteID       string
	SiteName     string
	DeviceSN     string
	DeviceName   string
	DeviceType   string
	SolarPower   float64
	OutputPower  float64
	GridPower    float64
	BatteryPower float64
	BatterySoC   float64
}
