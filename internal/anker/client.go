package anker

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL    = "https://ankerpower-api-eu.anker.com"
	apiVersion = "v2"
)

type Client struct {
	email      string
	password   string
	country    string
	httpClient *http.Client
	authToken  string
	userID     string
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Country  string `json:"country"`
}

type LoginResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		UserID      string `json:"user_id"`
		AuthToken   string `json:"auth_token"`
		Email       string `json:"email"`
		NickName    string `json:"nick_name"`
		Country     string `json:"country"`
	} `json:"data"`
}

type SiteListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		SiteList []Site `json:"site_list"`
	} `json:"data"`
}

type Site struct {
	SiteID      string   `json:"site_id"`
	SiteName    string   `json:"site_name"`
	SiteAdmin   bool     `json:"site_admin"`
	DeviceList  []Device `json:"device_list,omitempty"`
}

type Device struct {
	DeviceSN     string  `json:"device_sn"`
	DeviceName   string  `json:"device_name"`
	DeviceType   string  `json:"device_type"`
	DevicePowerW float64 `json:"device_power_w,omitempty"`
	GenerateTime int64   `json:"generate_time,omitempty"`
}

type EnergyDataRequest struct {
	SiteID    string `json:"site_id"`
	DeviceSN  string `json:"device_sn"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Type      string `json:"type"` // "day", "week", "month", "year"
}

type EnergyDataResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Power []PowerData `json:"power"`
	} `json:"data"`
}

type PowerData struct {
	Time      int64   `json:"time"`
	Solar     float64 `json:"solar_power,omitempty"`
	Output    float64 `json:"output_power,omitempty"`
	Grid      float64 `json:"grid_power,omitempty"`
	Battery   float64 `json:"battery_power,omitempty"`
	BatterySoC float64 `json:"battery_soc,omitempty"`
}

type Measurement struct {
	Timestamp   time.Time
	SiteID      string
	SiteName    string
	DeviceSN    string
	DeviceName  string
	DeviceType  string
	SolarPower  float64
	OutputPower float64
	GridPower   float64
	BatteryPower float64
	BatterySoC  float64
}

func NewClient(email, password, country string) *Client {
	return &Client{
		email:    email,
		password: password,
		country:  country,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Login() error {
	passwordHash := c.hashPassword(c.password)
	
	reqBody := LoginRequest{
		Email:    c.email,
		Password: passwordHash,
		Country:  c.country,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	resp, err := c.doRequest("POST", "/passport/login", body, false)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	if loginResp.Code != 0 {
		return fmt.Errorf("login failed: %s (code: %d)", loginResp.Msg, loginResp.Code)
	}

	c.authToken = loginResp.Data.AuthToken
	c.userID = loginResp.Data.UserID

	return nil
}

func (c *Client) GetSites() ([]Site, error) {
	resp, err := c.doRequest("POST", "/power_service/v1/site/get_site_list", nil, true)
	if err != nil {
		return nil, fmt.Errorf("get sites request failed: %w", err)
	}
	defer resp.Body.Close()

	var sitesResp SiteListResponse
	if err := json.NewDecoder(resp.Body).Decode(&sitesResp); err != nil {
		return nil, fmt.Errorf("failed to decode sites response: %w", err)
	}

	if sitesResp.Code != 0 {
		return nil, fmt.Errorf("get sites failed: %s (code: %d)", sitesResp.Msg, sitesResp.Code)
	}

	return sitesResp.Data.SiteList, nil
}

func (c *Client) GetEnergyData(siteID, deviceSN string, startTime, endTime time.Time) ([]PowerData, error) {
	reqBody := EnergyDataRequest{
		SiteID:    siteID,
		DeviceSN:  deviceSN,
		StartTime: startTime.Unix(),
		EndTime:   endTime.Unix(),
		Type:      "day",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal energy request: %w", err)
	}

	resp, err := c.doRequest("POST", "/power_service/v1/site/get_site_device_data", body, true)
	if err != nil {
		return nil, fmt.Errorf("get energy data request failed: %w", err)
	}
	defer resp.Body.Close()

	var energyResp EnergyDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&energyResp); err != nil {
		return nil, fmt.Errorf("failed to decode energy response: %w", err)
	}

	if energyResp.Code != 0 {
		return nil, fmt.Errorf("get energy data failed: %s (code: %d)", energyResp.Msg, energyResp.Code)
	}

	return energyResp.Data.Power, nil
}

func (c *Client) doRequest(method, path string, body []byte, needAuth bool) (*http.Response, error) {
	url := baseURL + path
	
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Country", c.country)
	req.Header.Set("Timezone", "Europe/Berlin")
	req.Header.Set("Model-Type", "DESKTOP")
	req.Header.Set("App-Name", "anker_power")
	req.Header.Set("Os-Type", "web")

	if needAuth && c.authToken != "" {
		req.Header.Set("X-Auth-Token", c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *Client) hashPassword(password string) string {
	hash := md5.Sum([]byte(password))
	return hex.EncodeToString(hash[:])
}

func (c *Client) IsAuthenticated() bool {
	return c.authToken != ""
}
