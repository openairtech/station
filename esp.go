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
	"time"

	"github.com/openairtech/api"
)

type EspData struct {
	System  *EspSystem   `json:"System,omitempty"`
	WiFi    *EspWiFi     `json:"WiFi,omitempty"`
	Sensors []EspSensors `json:"Sensors,omitempty"`
	TTL     int          `json:"TTL,omitempty"`
}

type EspSystem struct {
	Build             int     `json:"Build,omitempty"`
	GitBuild          string  `json:"Git Build,omitempty"`
	SystemLibraries   string  `json:"System libraries,omitempty"`
	Plugins           int     `json:"Plugins,omitempty"`
	PluginDescription string  `json:"Plugin description,omitempty"`
	LocalTime         string  `json:"Local time,omitempty"`
	Unit              int     `json:"Unit,omitempty"`
	Name              string  `json:"Name,omitempty"`
	UnitName          string  `json:"Unit Name,omitempty"`
	Uptime            int     `json:"Uptime,omitempty"`
	LastBootCause     string  `json:"Last boot cause,omitempty"`
	ResetReason       string  `json:"Reset Reason,omitempty"`
	Load              float32 `json:"Load,omitempty"`
	LoadLC            int     `json:"Load LC,omitempty"`
	FreeRAM           int     `json:"Free RAM,omitempty"`
}

type EspWiFi struct {
	Hostname                string `json:"Hostname,omitempty"`
	IPConfig                string `json:"IP config,omitempty"`
	IP                      string `json:"IP,omitempty"`
	SubnetMask              string `json:"Subnet Mask,omitempty"`
	GatewayIP               string `json:"Gateway IP,omitempty"`
	MACAddress              string `json:"MAC address"`       // mega-20190301
	StationMAC              string `json:"STA MAC,omitempty"` // mega-20190903
	DNS1                    string `json:"DNS 1,omitempty"`
	DNS2                    string `json:"DNS 2,omitempty"`
	SSID                    string `json:"SSID,omitempty"`
	BSSID                   string `json:"BSSID,omitempty"`
	Channel                 int    `json:"Channel,omitempty"`
	ConnectedMsec           int    `json:"Connected msec,omitempty"`
	LastDisconnectReason    int    `json:"Last Disconnect Reason,omitempty"`
	LastDisconnectReasonStr string `json:"Last Disconnect Reason str,omitempty"`
	NumberReconnects        int    `json:"Number reconnects,omitempty"`
	RSSI                    int    `json:"RSSI,omitempty"`
}

type EspTaskValues struct {
	ValueNumber int     `json:"ValueNumber,omitempty"`
	Name        string  `json:"Name"`
	NrDecimals  int     `json:"NrDecimals,omitempty"`
	Value       float32 `json:"Value"`
}

type EspDataAcquisition struct {
	Controller int  `json:"Controller"`
	IDX        int  `json:"IDX"`
	Enabled    bool `json:"Enabled,string"`
}

type EspSensors struct {
	TaskValues      []EspTaskValues      `json:"TaskValues,omitempty"`
	DataAcquisition []EspDataAcquisition `json:"DataAcquisition,omitempty"`
	TaskInterval    int                  `json:"TaskInterval,omitempty"`
	Type            string               `json:"Type,omitempty"`
	TaskName        string               `json:"TaskName"`
	TaskEnabled     bool                 `json:"TaskEnabled,string,omitempty"`
	TaskNumber      int                  `json:"TaskNumber,omitempty"`
}

type EspGpioControlResponse struct {
	Log    string `json:"log"`
	Plugin int    `json:"plugin"`
	Pin    int    `json:"pin"`
	Mode   string `json:"mode"`
	State  int    `json:"state"`
}

func NewEspData(m *api.Measurement, uptime time.Duration, name string) *EspData {
	bmeSensor := EspSensors{
		TaskName: "BME280",
		TaskValues: []EspTaskValues{
			{
				Name:  "Temperature",
				Value: *m.Temperature,
			},
			{
				Name:  "Humidity",
				Value: *m.Humidity,
			},
			{
				Name:  "Pressure",
				Value: *m.Pressure,
			},
		},
	}
	sdsSensor := EspSensors{
		TaskName: "SDS011",
		TaskValues: []EspTaskValues{
			{
				Name:  "PM2.5",
				Value: *m.Pm25,
			},
			{
				Name:  "PM10",
				Value: *m.Pm10,
			},
		},
	}
	return &EspData{
		System: &EspSystem{
			UnitName: name,
			Uptime:   int(uptime.Minutes()),
		},
		Sensors: []EspSensors{
			bmeSensor,
			sdsSensor,
		},
	}
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
