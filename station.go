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
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/openairtech/api"
)

func RunStation(ctx context.Context, espHost string, espPort int, apiServerUrl string, updatePeriod time.Duration) {
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
