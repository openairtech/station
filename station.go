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
	"context"
	"fmt"
	"math"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/openairtech/api"
)

func RunStation(ctx context.Context, espHost string, espPort int, apiServerUrl string,
	updatePeriod time.Duration, settleTime time.Duration, disablePmCorrectionFlag bool) {
	p := time.Duration(0)

	for {
		select {
		case <-time.After(p):
			p = updatePeriod

			url := fmt.Sprintf("http://%s:%d/json", espHost, espPort)

			log.Debugf("getting sensor data from station %s", url)

			var data EspData
			if err := GetData(url, &data); err != nil {
				log.Errorf("sensor data request failed: %v", err)
				continue
			}

			log.Debugf("received sensor data: %+v", data)

			uptime := time.Duration(data.System.Uptime) * time.Minute
			if uptime < settleTime {
				log.Debugf("ignoring sensor data since station uptime (%+v) is "+
					"shorter than data settle time (%+v)", uptime, settleTime)
				continue
			}

			t := api.UnixTime(time.Now())

			m := api.Measurement{
				Timestamp: &t,
			}

			for _, s := range data.Sensors {
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

			if !disablePmCorrectionFlag {
				correctPm(&m)
			}

			log.Debugf("temperature: %s, humidity: %s, pressure: %s, pm2.5: %s, pm10: %s",
				Float32RefToString(m.Temperature), Float32RefToString(m.Humidity), Float32RefToString(m.Pressure),
				Float32RefToString(m.Pm25), Float32RefToString(m.Pm10))

			f := api.FeederData{
				TokenId:      Sha1(data.WiFi.MACAddress),
				Measurements: []api.Measurement{m},
			}

			log.Debugf("posting data to %s: %+v", apiServerUrl, f)

			var r api.Result

			err := PostData(apiServerUrl, f, &r)
			if err != nil {
				log.Errorf("data posting failed: %v", err)
				continue
			}
			if r.Status != api.StatusOk {
				log.Errorf("data posting error: %d: %s", r.Status, r.Message)
			}

		case <-ctx.Done():
			log.Printf("stopping")
			return
		}
	}
}

// Correct PM values by humidity. Based on code by Piotr Paul et al: https://github.com/piotrkpaul/esp8266-sds011
func correctPm(m *api.Measurement) {
	if m.Humidity == nil {
		return
	}

	if m.Pm25 != nil {
		*m.Pm25 = correctedPm(*m.Pm25, *m.Humidity, 0.48756, 8.60068)
	}

	if m.Pm10 != nil {
		*m.Pm10 = correctedPm(*m.Pm10, *m.Humidity, 0.81559, 5.83411)
	}
}

func correctedPm(pm, humidity float32, a, b float64) float32 {
	return float32(float64(pm) / (1.0 + a*math.Pow(float64(humidity)/100.0, b)))
}
