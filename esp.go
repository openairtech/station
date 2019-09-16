// Copyright © 2019 Victor Antonovich <victor@antonovich.me>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/openairtech/api"
)

type EspData struct {
	System  EspSystem    `json:"System"`
	WiFi    EspWiFi      `json:"WiFi"`
	Sensors []EspSensors `json:"Sensors"`
	TTL     int          `json:"TTL"`
}

type EspSystem struct {
	Build             int     `json:"Build"`
	GitBuild          string  `json:"Git Build"`
	SystemLibraries   string  `json:"System libraries"`
	Plugins           int     `json:"Plugins"`
	PluginDescription string  `json:"Plugin description"`
	LocalTime         string  `json:"Local time"`
	Unit              int     `json:"Unit"`
	Name              string  `json:"Name"`
	Uptime            int     `json:"Uptime"`
	LastBootCause     string  `json:"Last boot cause"`
	ResetReason       string  `json:"Reset Reason"`
	Load              float32 `json:"Load"`
	LoadLC            int     `json:"Load LC"`
	FreeRAM           int     `json:"Free RAM"`
}

type EspWiFi struct {
	Hostname                string `json:"Hostname"`
	IPConfig                string `json:"IP config"`
	IP                      string `json:"IP"`
	SubnetMask              string `json:"Subnet Mask"`
	GatewayIP               string `json:"Gateway IP"`
	MACAddress              string `json:"MAC address"` // mega-20190301
	StationMAC              string `json:"STA MAC"`     // mega-20190903
	DNS1                    string `json:"DNS 1"`
	DNS2                    string `json:"DNS 2"`
	SSID                    string `json:"SSID"`
	BSSID                   string `json:"BSSID"`
	Channel                 int    `json:"Channel"`
	ConnectedMsec           int    `json:"Connected msec"`
	LastDisconnectReason    int    `json:"Last Disconnect Reason"`
	LastDisconnectReasonStr string `json:"Last Disconnect Reason str"`
	NumberReconnects        int    `json:"Number reconnects"`
	RSSI                    int    `json:"RSSI"`
}

type EspTaskValues struct {
	ValueNumber int     `json:"ValueNumber"`
	Name        string  `json:"Name"`
	NrDecimals  int     `json:"NrDecimals"`
	Value       float32 `json:"Value"`
}

type EspDataAcquisition struct {
	Controller int  `json:"Controller"`
	IDX        int  `json:"IDX"`
	Enabled    bool `json:"Enabled,string"`
}

type EspSensors struct {
	TaskValues      []EspTaskValues      `json:"TaskValues"`
	DataAcquisition []EspDataAcquisition `json:"DataAcquisition"`
	TaskInterval    int                  `json:"TaskInterval"`
	Type            string               `json:"Type"`
	TaskName        string               `json:"TaskName"`
	TaskEnabled     bool                 `json:"TaskEnabled,string"`
	TaskNumber      int                  `json:"TaskNumber"`
}

func (ed *EspData) Measurement(t api.UnixTime) *api.Measurement {
	m := api.Measurement{
		Timestamp: &t,
	}

	for _, s := range ed.Sensors {
		if !s.TaskEnabled {
			continue
		}
		switch s.TaskName {
		case "BME280":
			for _, v := range s.TaskValues {
				cv := v
				switch v.Name {
				case "Temperature":
					m.Temperature = &cv.Value
				case "Humidity":
					m.Humidity = &cv.Value
				case "Pressure":
					m.Pressure = &cv.Value
				}
			}
		case "SDS011":
			for _, v := range s.TaskValues {
				cv := v
				switch v.Name {
				case "PM2.5":
					m.Pm25 = &cv.Value
				case "PM10":
					m.Pm10 = &cv.Value
				}
			}
		}
	}

	return &m
}

func (ew *EspWiFi) MacAddress() string {
	if ew.MACAddress != "" {
		return ew.MACAddress
	}
	return ew.StationMAC
}
