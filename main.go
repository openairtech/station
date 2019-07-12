package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	Version   = "unknown"
	Timestamp = "unknown"
)

func main() {
	versionFlag := flag.Bool("v", false, "print the version number and quit")

	debugFlag := flag.Bool("d", false, "enable debug logging")

	espHost := flag.String("h", "OpenAir.local", "ESP station address")
	espPort := flag.Int("p", 80, "ESP station port")

	apiServerUrl := flag.String("a", "https://api.openair.city/v1/feeder", "feeder endpoint address")

	updatePeriod := flag.Duration("t", 1*time.Minute, "data update period")

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

	log.Printf("starting: %s built %s", Version, Timestamp)

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

	RunStation(ctx, *espHost, *espPort, *apiServerUrl, *updatePeriod, *disablePmCorrectionFlag)

	log.Printf("exiting...")
}
