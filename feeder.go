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
	"strconv"
	"strings"
	"time"

	"github.com/openairtech/api"
)

type Feeder interface {
	Feed(data *StationData)
}

// https://github.com/openairtech/api
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
	if err := HttpPostJson(oaf.apiServerUrl, nil, f, &r); err != nil {
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

type SensorDataValue struct {
	ValueType string  `json:"value_type"`
	Value     float32 `json:"value"`
}

type SensorData struct {
	SoftwareVersion  string            `json:"software_version"`
	SensorDataValues []SensorDataValue `json:"sensordatavalues"`
}

// https://github.com/opendata-stuttgart/meta/wiki/APIs
// https://github.com/opendata-stuttgart/sensors-software/blob/master/airrohr-firmware/airrohr-firmware.ino
type LuftdatenFeeder struct {
	apiServerUrl           string
	sensorDataPostInterval time.Duration

	lastSensorDataPostTime time.Time
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

	pmSensorData := &SensorData{
		SoftwareVersion: data.Version,
		SensorDataValues: []SensorDataValue{
			{ValueType: "P1", Value: Float32RefRound(data.LastMeasurement.Pm10, 1)},
			{ValueType: "P2", Value: Float32RefRound(data.LastMeasurement.Pm25, 1)},
		},
	}
	lf.postSensorData(sensorId, 1, pmSensorData)

	envSensorData := &SensorData{
		SoftwareVersion: data.Version,
		SensorDataValues: []SensorDataValue{
			{ValueType: "temperature", Value: Float32RefRound(data.LastMeasurement.Temperature, 1)},
			{ValueType: "humidity", Value: Float32RefRound(data.LastMeasurement.Humidity, 1)},
			{ValueType: "pressure", Value: 100 * Float32RefRound(data.LastMeasurement.Pressure, 2)},
		},
	}
	lf.postSensorData(sensorId, 11, envSensorData)

	lf.lastSensorDataPostTime = time.Now()
}

func (lf *LuftdatenFeeder) postSensorData(sensorId string, sensorPin int, sensorData *SensorData) {
	log.Debugf("[Luftdaten] %s: posting sensor [%d] data to %s", sensorId, sensorPin, lf.apiServerUrl)

	headers := map[string]interface{}{
		"X-Sensor": sensorId,
		"X-Pin":    sensorPin,
	}

	var r map[string]*json.RawMessage
	if err := HttpPostJson(lf.apiServerUrl, headers, sensorData, &r); err != nil {
		log.Errorf("[Luftdaten] %s: sensor [%d] data posting failed: %v", sensorId, sensorPin, err)
		return
	}

	log.Debugf("[Luftdaten] %s: successfully posted sensor [%d] data", sensorId, sensorPin)
}

// https://github.com/zakarlyukin/aircms/blob/master/docs/index.rst
type AirCmsFeeder struct {
	apiServerUrl           string
	sensorDataPostInterval time.Duration

	lastSensorDataPostTime time.Time
}

func NewAirCmsFeederFeeder() *AirCmsFeeder {
	return &AirCmsFeeder{
		apiServerUrl:           "http://doiot.ru/php/sensors.php",
		sensorDataPostInterval: 3 * time.Minute,
	}
}

func (acf *AirCmsFeeder) Feed(data *StationData) {
	l, err := strconv.ParseInt(data.TokenId[12:20], 16, 64)
	if err != nil {
		log.Errorf("[AirCMS] can't get login from token %s: %v", data.TokenId, err)
		return
	}
	login := strconv.FormatInt(l, 10)

	if time.Since(acf.lastSensorDataPostTime) < acf.sensorDataPostInterval {
		log.Debugf("[AirCMS] %s: skip sensor data posting", login)
		return
	}

	token := fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		data.TokenId[0:2], data.TokenId[2:4], data.TokenId[4:6],
		data.TokenId[6:8], data.TokenId[8:10], data.TokenId[10:12])
	token = strings.ToUpper(token)

	sensorData := &SensorData{
		SoftwareVersion: data.Version,
		SensorDataValues: []SensorDataValue{
			{ValueType: "SDS_P1", Value: Float32RefRound(data.LastMeasurement.Pm10, 1)},
			{ValueType: "SDS_P2", Value: Float32RefRound(data.LastMeasurement.Pm25, 1)},
			{ValueType: "BME280_temperature", Value: Float32RefRound(data.LastMeasurement.Temperature, 1)},
			{ValueType: "BME280_humidity", Value: Float32RefRound(data.LastMeasurement.Humidity, 1)},
			{ValueType: "BME280_pressure", Value: 100 * Float32RefRound(data.LastMeasurement.Pressure, 2)},
		},
	}

	jd, err := json.Marshal(sensorData)
	if err != nil {
		log.Errorf("[AirCMS] %s: can't marshal sensor data: %v", login, err)
		return
	}

	var timestamp time.Time
	if data.LastMeasurement.Timestamp != nil {
		timestamp = time.Time(*data.LastMeasurement.Timestamp)
	} else {
		timestamp = time.Now()
	}

	d := fmt.Sprintf("L=%s&t=%d&airrohr=%s", login, timestamp.Unix(), string(jd))
	log.Debugf("[AirCMS] %s: data to post: %s", login, d)

	postUrl := fmt.Sprintf("%s?h=%s", acf.apiServerUrl, Sha1(Sha1(token)+Sha1(d+token)))
	log.Debugf("[AirCMS] %s: posting sensor data to %s, token: %s", login, acf.apiServerUrl, token)

	var r []byte
	if r, err = HttpPostData(postUrl, nil, []byte(d)); err != nil {
		log.Errorf("[AirCMS] %s: sensor data posting failed: %v", login, err)
		return
	}

	log.Debugf("[AirCMS] %s: successfully posted sensor data, response: %s", login, string(r))

	acf.lastSensorDataPostTime = time.Now()
}
