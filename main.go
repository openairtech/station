package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	StationModeEsp = "esp"
	StationModeRpi = "rpi"
)

var (
	Version   = "unknown"
	Timestamp = "unknown"
)

func main() {
	versionFlag := flag.Bool("v", false, "print the version number and quit")

	debugFlag := flag.Bool("d", false, "enable debug logging")

	mode := flag.String("m", StationModeEsp, "station mode (esp/rpi)")

	espHost := flag.String("h", "OpenAir.local", "ESP station address")
	espPort := flag.Int("p", 80, "ESP station port")

	rpiI2cBusId := flag.Int("i", 1, "RPi station I2C bus ID")
	rpiSerialPort := flag.String("s", "/dev/ttyAMA0", "RPi station serial port name")

	apiServerUrl := flag.String("a", "https://api.openair.city/v1/feeder", "feeder endpoint address")

	updateInterval := flag.Duration("t", 1*time.Minute, "data update interval")

	keepDuration := flag.Duration("k", 6*time.Hour, "buffered data keep duration")

	settleTime := flag.Duration("S", 5*time.Minute, "data settle time after station restart")

	resolverTimeout := flag.Duration("r", 15*time.Second, "name resolver timeout")

	httpTimeout := flag.Duration("T", 15*time.Second, "http client timeout")

	disablePmCorrectionFlag := flag.Bool("c", false, "disable PM values correction by humidity")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("Build version: %s\n", Version)
		fmt.Printf("Build timestamp: %s\n", Timestamp)
		return
	}

	if *debugFlag {
		log.SetLevel(log.DebugLevel)
	}

	InitResolvers(*resolverTimeout)

	InitHttp(*httpTimeout)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	defer func() {
		signal.Stop(signalCh)
		cancel()
	}()

	go func() {
		select {
		case sig := <-signalCh:
			log.Printf("received %v signal", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	var station Station

	version := fmt.Sprintf("%s-%s_%s-%s_%s", *mode, Version, Timestamp, runtime.GOARCH, runtime.GOOS)

	if *mode == StationModeEsp {
		station = NewEspStation(version, *espHost, *espPort)
	} else if *mode == StationModeRpi {
		var err error
		if station, err = NewRpiStation(version, *rpiI2cBusId, 0x76, *rpiSerialPort, 3); err != nil {
			log.Fatalf("can't initialize RPi station: %v", err)
		}
	} else {
		log.Fatalf("unknown station mode: %s", *mode)
	}

	log.Printf("starting station, version: %s", version)

	feeders := []Feeder{
		NewOpenAirFeeder(*apiServerUrl, *keepDuration),
		NewLuftdatenFeeder(),
	}

	RunStation(ctx, station, feeders, *updateInterval, *settleTime, *disablePmCorrectionFlag)

	log.Printf("exiting...")
}
