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
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/openairtech/api"

	"github.com/stretchr/testify/require"
)

func testReadEspData(fn string) (*EspData, error) {
	b, err := ioutil.ReadFile(filepath.Join("testdata", fn))
	if err != nil {
		return nil, err
	}
	var d EspData
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func TestEspData_Measurement(t *testing.T) {
	ts := api.UnixTime(time.Now())
	temp := float32(25)
	humidity := float32(33.5)
	pressure := float32(1015.1)
	pm25 := float32(1.8)
	pm10 := float32(14.5)

	m := api.Measurement{
		Timestamp:   &ts,
		Temperature: &temp,
		Humidity:    &humidity,
		Pressure:    &pressure,
		Pm25:        &pm25,
		Pm10:        &pm10,
		Aqi:         nil,
	}

	tests := []struct {
		name string
		file string
		want api.Measurement
	}{
		{name: "esp-mega-20190301", file: "esp-mega-20190301.json", want: m},
		{name: "esp-mega-20190903", file: "esp-mega-20190903.json", want: m},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed, err := testReadEspData(tt.file)
			require.NoError(t, err)
			got := ed.Measurement(ts)
			require.Equal(t, tt.want, *got)
		})
	}
}

func TestEspWiFi_MacAddress(t *testing.T) {
	tests := []struct {
		name string
		file string
		want string
	}{
		{name: "esp-mega-20190301", file: "esp-mega-20190301.json", want: "12:34:56:78:90:AB"},
		{name: "esp-mega-20190903", file: "esp-mega-20190903.json", want: "12:34:56:78:90:AB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed, err := testReadEspData(tt.file)
			require.NoError(t, err)
			got := ed.WiFi.MacAddress()
			require.Equal(t, tt.want, got)
		})
	}
}
