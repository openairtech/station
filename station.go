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
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/d2r2/go-bsbmp"
	"github.com/d2r2/go-i2c"
	bmelogger "github.com/d2r2/go-logger"

	"github.com/NotifAi/serial"

	"github.com/openairtech/sds011/go/sds011"

	"github.com/openairtech/api"
)

const (
	// System epoch time (2019-01-01 GMT) as an Unix time
	systemEpoch = 1546300800
	// Heater disabling humidity hysteresis (in percents)
	heaterDisableHumidityHysteresis = 5
)

type HeaterState bool

const (
	HeaterOn  HeaterState = true
	HeaterOff HeaterState = false
)

type StationData struct {
	Version         string
	TokenId         string
	Uptime          time.Duration
	LastMeasurement *api.Measurement
}

type Station interface {
	Version() string
	Start() error
	Stop()
	HeaterState() HeaterState
	TurnHeater(state HeaterState)
	GetData() (*StationData, error)
}

type EspStation struct {
	version string

	host string
	port int

	tokenId string

	heaterPin   int
	heaterState HeaterState

	lastUptime *time.Duration
}

func NewEspStation(version, host string, port int, heaterPin int, tokenId string) *EspStation {
	return &EspStation{
		version:   version,
		host:      host,
		port:      port,
		tokenId:   tokenId,
		heaterPin: heaterPin,
	}
}

func (es *EspStation) Version() string {
	return es.version
}

func (es *EspStation) Start() error {
	log.Print("started ESP station")
	return nil
}

func (es *EspStation) Stop() {
	log.Print("stopped ESP station")
}

func (es *EspStation) HeaterState() HeaterState {
	return es.heaterState
}

func (es *EspStation) TurnHeater(state HeaterState) {
	pinState := 0
	if state == HeaterOn {
		pinState = 1
	}

	url := fmt.Sprintf("http://%s:%d/control?cmd=GPIO,%d,%d", es.host, es.port, es.heaterPin, pinState)
	var response EspGpioControlResponse
	if err := HttpGetData(url, &response); err != nil {
		log.Errorf("can't set heater pin %d state %d: %v", es.heaterPin, pinState, err)
		return
	}

	es.heaterState = state

	if state == HeaterOn {
		log.Debug("heater turned on")
	} else {
		log.Debug("heater turned off")
	}
}

func (es *EspStation) GetData() (*StationData, error) {
	url := fmt.Sprintf("http://%s:%d/json", es.host, es.port)

	log.Debugf("getting sensor data from ESP station %s", url)

	var data EspData
	if err := HttpGetData(url, &data); err != nil {
		log.Errorf("sensor data request failed: %v", err)
		return nil, err
	}

	log.Debugf("received sensor data: %+v", data)

	m := data.Measurement(api.UnixTime(time.Now()))

	var tokenId string
	if es.tokenId != "" {
		tokenId = es.tokenId
	} else {
		tokenId = stationTokenId(data.WiFi.MacAddress())
	}
	log.Debugf("token ID: %s", tokenId)

	uptime := time.Duration(data.System.Uptime) * time.Minute

	if es.lastUptime != nil && uptime < *es.lastUptime {
		log.Warn("ESP station reboot detected")
		es.heaterState = HeaterOff
	}

	es.lastUptime = &uptime

	return &StationData{
		Version:         es.version,
		TokenId:         tokenId,
		Uptime:          uptime,
		LastMeasurement: m,
	}, nil
}

type RpiStation struct {
	version string

	i2cBusId          int
	bmeSensorAddress  int
	sdsSensorPort     string
	sdsSensorInterval int

	startTime time.Time
	tokenId   string

	i2cBus     *i2c.I2C
	serialPort serial.Port

	bmeSensor *bsbmp.BMP
	sdsSensor *sds011.Sensor

	pmLock sync.RWMutex
	pm25   float32
	pm10   float32

	heaterPin   int
	heaterState HeaterState
}

func NewRpiStation(version string, i2cBusId int, bmeSensorAddress int, sdsSensorPort string, sdsSensorInterval int,
	heaterPin int, tokenId string) (*RpiStation, error) {
	if tokenId == "" {
		macAddress := WirelessInterfaceMacAddr()
		if macAddress == "" {
			return nil, errors.New("can't determine RPi station MAC address")
		}
		log.Debugf("MAC address: %s", macAddress)
		tokenId = stationTokenId(macAddress)
	}
	log.Debugf("token ID: %s", tokenId)

	return &RpiStation{
		version:           version,
		startTime:         time.Now(),
		tokenId:           tokenId,
		i2cBusId:          i2cBusId,
		bmeSensorAddress:  bmeSensorAddress,
		sdsSensorPort:     sdsSensorPort,
		sdsSensorInterval: sdsSensorInterval,
		heaterPin:         heaterPin,
	}, nil
}

func (rs *RpiStation) Version() string {
	return rs.version
}

func (rs *RpiStation) Start() error {
	log.Print("starting RPi station...")

	// Init BME280 sensor I2C bus
	if err := rs.initI2cBus(); err != nil {
		return fmt.Errorf("I2C bus init error: %v", err)
	}

	// Init BME280 sensor
	if err := rs.initBmeSensor(); err != nil {
		return fmt.Errorf("BME280 sensor init error: %v", err)
	}

	// Open SDS011 sensor serial port
	if err := rs.initSerialPort(); err != nil {
		return fmt.Errorf("serial port init error: %v", err)
	}

	// Init SDS011 sensor
	if err := rs.initSdsSensor(); err != nil {
		return fmt.Errorf("SDS011 sensor init error: %v", err)
	}

	// Start SDS011 sensor data reading
	go rs.readSdsSensor()

	return nil
}

func (rs *RpiStation) initSerialPort() error {
	var err error
	rs.serialPort, err = serial.OpenPort(serial.Config{
		Name: rs.sdsSensorPort,
		Baud: 9600,
	})
	if err != nil {
		return err
	}
	return rs.flushSerialPort()
}

func (rs *RpiStation) flushSerialPort() error {
	return rs.serialPort.Flush()
}

func (rs *RpiStation) initI2cBus() (err error) {
	rs.i2cBus, err = i2c.NewI2C(uint8(rs.bmeSensorAddress), rs.i2cBusId)
	return
}

func (rs *RpiStation) initBmeSensor() error {
	_ = bmelogger.ChangePackageLogLevel("i2c", bmelogger.ErrorLevel)
	_ = bmelogger.ChangePackageLogLevel("bsbmp", bmelogger.ErrorLevel)

	// Check BME280 sensor presence
	var err error
	if rs.bmeSensor, err = bsbmp.NewBMP(bsbmp.BME280, rs.i2cBus); err != nil {
		return fmt.Errorf("can't find BME280 sensor: %v", err)
	}

	// Check BME280 sensor have valid state
	if err = rs.bmeSensor.IsValidCoefficients(); err != nil {
		return fmt.Errorf("invalid BME280 sensor state: %v", err)
	}

	return nil
}

func (rs *RpiStation) initSdsSensor() error {
	rs.sdsSensor = sds011.NewSensor(rs.serialPort)
	return rs.sdsSensor.SetCycle(uint8(rs.sdsSensorInterval))
}

func (rs *RpiStation) readSdsSensor() {
	for {
		point, err := rs.sdsSensor.Get()
		if err != nil {
			log.Errorf("can't read SDS011 sensor: %v", err)
			time.Sleep(3 * time.Second)
			_ = rs.flushSerialPort()
			continue
		}
		rs.pmLock.Lock()
		rs.pm25 = float32(point.PM25)
		rs.pm10 = float32(point.PM10)
		rs.pmLock.Unlock()
		log.Debugf("read SDS011 sensor values, PM2.5: %v, PM10: %v", point.PM25, point.PM10)
	}
}

func (rs *RpiStation) Stop() {
	log.Print("stopping RPi station...")
	_ = rs.i2cBus.Close()
	rs.sdsSensor.Close()
}

func (rs *RpiStation) HeaterState() HeaterState {
	return rs.heaterState
}

func (rs *RpiStation) TurnHeater(state HeaterState) {
	cmdPinMode := fmt.Sprintf("gpio -1 mode %d out", rs.heaterPin)
	if err := Execute(cmdPinMode, 5*time.Second); err != nil {
		log.Errorf("can't set heater pin %d output mode: %v", rs.heaterPin, err)
		return
	}

	pinState := 0
	if state == HeaterOn {
		pinState = 1
	}
	cmdPinState := fmt.Sprintf("gpio -1 write %d %d", rs.heaterPin, pinState)
	if err := Execute(cmdPinState, 5*time.Second); err != nil {
		log.Errorf("can't set heater pin %d state %d: %v", rs.heaterPin, pinState, err)
		return
	}

	rs.heaterState = state

	if state == HeaterOn {
		log.Debug("heater turned on")
	} else {
		log.Debug("heater turned off")
	}
}

func (rs *RpiStation) GetData() (*StationData, error) {
	timestamp := api.UnixTime(time.Now())

	// Read temperature in Celsius degree
	temperature, err := rs.bmeSensor.ReadTemperatureC(bsbmp.ACCURACY_STANDARD)
	if err != nil {
		return nil, err
	}

	// Read relative humidity
	_, humidity, err := rs.bmeSensor.ReadHumidityRH(bsbmp.ACCURACY_STANDARD)
	if err != nil {
		return nil, err
	}

	// Read pressure in Pa
	pressure, err := rs.bmeSensor.ReadPressurePa(bsbmp.ACCURACY_STANDARD)
	if err != nil {
		return nil, err
	}
	// Convert pressure to hPa
	pressure /= 100

	rs.pmLock.RLock()
	pm25 := rs.pm25
	pm10 := rs.pm10
	rs.pmLock.RUnlock()

	m := &api.Measurement{
		Timestamp:   &timestamp,
		Temperature: &temperature,
		Humidity:    &humidity,
		Pressure:    &pressure,
		Pm25:        &pm25,
		Pm10:        &pm10,
		Aqi:         nil,
	}

	return &StationData{
		Version:         rs.version,
		TokenId:         rs.tokenId,
		Uptime:          time.Since(rs.startTime),
		LastMeasurement: m,
	}, nil
}

func RunStation(ctx context.Context, station Station, feeders []Feeder, publishers []Publisher,
	updateInterval time.Duration, settleTime time.Duration, disablePmCorrection, enableHeater bool,
	heaterTurnOnHumidity int) {
	p := time.Duration(0)

	for _, publisher := range publishers {
		publisher.Start()
	}

	if err := station.Start(); err != nil {
		log.Errorf("can't start station: %v", err)
		return
	}

	// Turn heater off at startup and at exit, if it's enabled
	if enableHeater {
		station.TurnHeater(HeaterOff)
		defer station.TurnHeater(HeaterOff)
	}

	defer station.Stop()

	for _, publisher := range publishers {
		defer publisher.Stop()
	}

	for {
		select {
		case <-time.After(p):
			p = updateInterval

			data, err := station.GetData()
			if err != nil {
				log.Errorf("station data request failed: %v", err)
				continue
			}

			m := data.LastMeasurement

			if enableHeater && m.Humidity != nil {
				humidity := int(*m.Humidity)
				if station.HeaterState() == HeaterOff {
					if humidity >= heaterTurnOnHumidity {
						log.Infof("turning heater ON (humidity: %d%%)", humidity)
						station.TurnHeater(HeaterOn)
					}
				} else if humidity <= heaterTurnOnHumidity-heaterDisableHumidityHysteresis {
					log.Infof("turning heater OFF (humidity: %d%%)", humidity)
					station.TurnHeater(HeaterOff)
				}
			} else if !disablePmCorrection {
				correctPm(m)
			}

			log.Debugf("temperature: %s, humidity: %s, pressure: %s, pm2.5: %s, pm10: %s",
				Float32RefToString(m.Temperature), Float32RefToString(m.Humidity), Float32RefToString(m.Pressure),
				Float32RefToString(m.Pm25), Float32RefToString(m.Pm10))

			if time.Now().Before(time.Unix(systemEpoch, 0)) {
				log.Info("ignoring station data since station system time probably is not in sync")
				continue
			}

			if data.Uptime < settleTime {
				log.Infof("ignoring station data since station uptime (%+v) is "+
					"shorter than data settle time (%+v)", data.Uptime, settleTime)
				continue
			}

			for _, feeder := range feeders {
				feeder.Feed(data)
			}

			for _, publisher := range publishers {
				publisher.Publish(data)
			}

		case <-ctx.Done():
			return
		}
	}
}

func stationTokenId(stationMacAddress string) string {
	return Sha1(strings.ToUpper(stationMacAddress))
}

func correctPm(m *api.Measurement) {
	if m.Humidity == nil {
		return
	}

	if m.Pm25 != nil {
		*m.Pm25 = Float32Round(correctedPm(*m.Pm25, *m.Humidity, 0.48756, 8.60068), 1)
	}

	if m.Pm10 != nil {
		*m.Pm10 = Float32Round(correctedPm(*m.Pm10, *m.Humidity, 0.81559, 5.83411), 1)
	}
}

func correctedPm(pm, humidity float32, a, b float64) float32 {
	return float32(float64(pm) / (1.0 + a*math.Pow(float64(humidity)/100.0, b)))
}
