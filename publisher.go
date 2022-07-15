// Copyright © 2022 Victor Antonovich <victor@antonovich.me>
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
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Publisher interface {
	Start() error
	Stop()
	Publish(data *StationData)
}

type HttpPublisher struct {
	sync.Mutex

	port int

	server       *http.Server
	serverStopWg *sync.WaitGroup

	lastData *StationData
}

func NewHttpPublisher(port int) *HttpPublisher {
	return &HttpPublisher{
		port: port,
	}
}

func (hp *HttpPublisher) Start() error {
	log.Printf("starting sensor data HTTP publisher at http://0.0.0.0:%d/json", hp.port)
	mux := http.NewServeMux()
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		if hp.lastData == nil {
			w.WriteHeader(503)
			return
		}
		ld := hp.lastData
		ep := NewEspData(ld.LastMeasurement, ld.Uptime, ld.Version)
		jd, err := json.Marshal(ep)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		w.Write(jd)
	})
	hp.server = &http.Server{Addr: fmt.Sprintf(":%d", hp.port), Handler: mux}
	hp.serverStopWg = &sync.WaitGroup{}
	hp.serverStopWg.Add(1)
	go func() {
		defer hp.serverStopWg.Done()
		if err := hp.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Errorf("can't start sensor data HTTP publisher:  %v", err)
		}
	}()
	return nil
}

func (hp *HttpPublisher) Stop() {
	log.Print("stopping sensor data HTTP publisher...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := hp.server.Shutdown(ctx); err != nil {
		log.Errorf("error while stopping sensor data HTTP server: %v", err)
	}
	hp.serverStopWg.Wait()
	log.Print("sensor data HTTP publisher stopped")
}

func (hp *HttpPublisher) Publish(data *StationData) {
	lastData := *data
	hp.Lock()
	defer hp.Unlock()
	hp.lastData = &lastData
}
