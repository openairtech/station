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

	apiServerUrl := flag.String("a", "http://localhost:8081/v1/feeder", "data receiver address")

	updatePeriod := flag.Duration("t", 10*time.Second, "data update period")

	resolverTimeout := flag.Duration("r", 15*time.Second, "name resolver timeout")

	httpTimeout := flag.Duration("T", 15*time.Second, "http client timeout")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("Build version: %s\n", Version)
		fmt.Printf("Build timestamp: %s\n", Timestamp)
		return
	}

	if *debugFlag {
		log.SetLevel(log.DebugLevel)
	}

	log.Printf("starting...")

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

	RunStation(ctx, *espHost, *espPort, *apiServerUrl, *updatePeriod)

	log.Printf("exiting...")
}
