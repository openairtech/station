// Copyright © 2020 Victor Antonovich <victor@antonovich.me>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"

	"github.com/openairtech/api"
)

type Feeder interface {
	Feed(data *StationData)
}

type OpenAirFeeder struct {
	apiServerUrl             string
	measurementsKeepDuration time.Duration

	measurements []api.Measurement
}

func NewOpenAirFeeder(apiServerUrl string, measurementsKeepDuration time.Duration) *OpenAirFeeder {
	return &OpenAirFeeder{
		apiServerUrl:             apiServerUrl,
		measurementsKeepDuration: measurementsKeepDuration,
	}
}

func (oaf *OpenAirFeeder) Feed(data *StationData) {
	// Delete expired buffered measurements
	now := time.Now()
	for {
		if len(oaf.measurements) == 0 {
			break
		}

		// Stop on first unexpired buffered measurement
		t := oaf.measurements[0].Timestamp
		if t != nil && now.Sub(time.Time(*t)) < oaf.measurementsKeepDuration {
			break
		}

		// Remove expired buffered measurement
		oaf.measurements = oaf.measurements[1:]
	}

	// Add last data measurement to buffered measurements
	oaf.measurements = append(oaf.measurements, *data.LastMeasurement)

	f := api.FeederData{
		TokenId:      data.TokenId,
		Version:      data.Version,
		Measurements: oaf.measurements,
	}

	log.Debugf("[OpenAir] posting %d measurement(s) to %s", len(oaf.measurements), oaf.apiServerUrl)

	var r api.Result
	if err := HttpPostData(oaf.apiServerUrl, nil, f, &r); err != nil {
		log.Errorf("[OpenAir] data posting failed: %v", err)
		return
	}
	if r.Status != api.StatusOk {
		log.Errorf("[OpenAir] data posting error: %d: %s", r.Status, r.Message)
		return
	}

	log.Debugf("[OpenAir] successfully posted %d measurement(s) to %s", len(oaf.measurements), oaf.apiServerUrl)

	// Delete successfully posted buffered measurements
	oaf.measurements = nil
}

type LuftdatenFeeder struct {
	apiServerUrl           string
	sensorDataPostInterval time.Duration

	lastSensorDataPostTime time.Time
}

type LuftdatenSensorData struct {
	SoftwareVersion  string                     `json:"software_version"`
	SensorDataValues []LuftdatenSensorDataValue `json:"sensordatavalues"`
}
type LuftdatenSensorDataValue struct {
	ValueType string  `json:"value_type"`
	Value     float32 `json:"value"`
}

func NewLuftdatenFeeder() *LuftdatenFeeder {
	return &LuftdatenFeeder{
		apiServerUrl:           "https://api.luftdaten.info/v1/push-sensor-data/",
		sensorDataPostInterval: 3 * time.Minute,
	}
}

func (lf *LuftdatenFeeder) Feed(data *StationData) {
	sensorId := fmt.Sprintf("raspi-%s", data.TokenId[:12])

	if time.Since(lf.lastSensorDataPostTime) < lf.sensorDataPostInterval {
		log.Debugf("[Luftdaten] %s: skip sensor data posting", sensorId)
		return
	}

	pmSensorData := &LuftdatenSensorData{
		SoftwareVersion: data.Version,
		SensorDataValues: []LuftdatenSensorDataValue{
			{ValueType: "P1", Value: Float32RefRound(data.LastMeasurement.Pm10, 1)},
			{ValueType: "P2", Value: Float32RefRound(data.LastMeasurement.Pm25, 1)},
		},
	}
	lf.postSensorData(sensorId, 1, pmSensorData)

	envSensorData := &LuftdatenSensorData{
		SoftwareVersion: data.Version,
		SensorDataValues: []LuftdatenSensorDataValue{
			{ValueType: "temperature", Value: Float32RefRound(data.LastMeasurement.Temperature, 1)},
			{ValueType: "humidity", Value: Float32RefRound(data.LastMeasurement.Humidity, 1)},
			{ValueType: "pressure", Value: 100 * Float32RefRound(data.LastMeasurement.Pressure, 2)},
		},
	}
	lf.postSensorData(sensorId, 11, envSensorData)

	lf.lastSensorDataPostTime = time.Now()
}

func (lf *LuftdatenFeeder) postSensorData(sensorId string, sensorPin int, sensorData *LuftdatenSensorData) {
	log.Debugf("[Luftdaten] %s: posting sensor [%d] data to %s", sensorId, sensorPin, lf.apiServerUrl)

	headers := map[string]interface{}{
		"X-Sensor": sensorId,
		"X-Pin":    sensorPin,
	}

	var r map[string]*json.RawMessage
	if err := HttpPostData(lf.apiServerUrl, headers, sensorData, &r); err != nil {
		log.Errorf("[Luftdaten] %s: sensor [%d] data posting failed: %v", sensorId, sensorPin, err)
		return
	}

	log.Debugf("[Luftdaten] %s: successfully posted sensor [%d] data", sensorId, sensorPin)
}
