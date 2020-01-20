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
	log "github.com/sirupsen/logrus"
	"time"

	"github.com/openairtech/api"
)

type Endpoint interface {
	FeedStationData(data *StationData)
}

type OpenAirEndpoint struct {
	apiServerUrl             string
	measurementsKeepDuration time.Duration

	measurements []api.Measurement
}

func NewOpenAirEndpoint(apiServerUrl string, measurementsKeepDuration time.Duration) *OpenAirEndpoint {
	return &OpenAirEndpoint{
		apiServerUrl:             apiServerUrl,
		measurementsKeepDuration: measurementsKeepDuration,
	}
}

func (oae *OpenAirEndpoint) FeedStationData(data *StationData) {
	// Delete expired buffered measurements
	now := time.Now()
	for {
		if len(oae.measurements) == 0 {
			break
		}

		// Stop on first unexpired buffered measurement
		t := oae.measurements[0].Timestamp
		if t != nil && now.Sub(time.Time(*t)) < oae.measurementsKeepDuration {
			break
		}

		// Remove expired buffered measurement
		oae.measurements = oae.measurements[1:]
	}

	// Add last data measurement to buffered measurements
	oae.measurements = append(oae.measurements, *data.LastMeasurement)

	f := api.FeederData{
		TokenId:      data.TokenId,
		Version:      data.Version,
		Measurements: oae.measurements,
	}

	log.Debugf("[OpenAir] posting %d measurement(s) to %s", len(oae.measurements), oae.apiServerUrl)

	var r api.Result
	if err := HttpPostData(oae.apiServerUrl, f, &r); err != nil {
		log.Errorf("[OpenAir] data posting failed: %v", err)
		return
	}
	if r.Status != api.StatusOk {
		log.Errorf("[OpenAir] data posting error: %d: %s", r.Status, r.Message)
		return
	}

	log.Debugf("[OpenAir] successfully posted %d measurement(s) to %s", len(oae.measurements), oae.apiServerUrl)

	// Delete successfully posted buffered measurements
	oae.measurements = nil
}
