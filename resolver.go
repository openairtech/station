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
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	log "github.com/sirupsen/logrus"
)

func InitResolvers(timeout time.Duration) {
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	http.DefaultTransport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		log.Debugf("dialing: %s", addr)

		host, port := ParseAddr(addr)

		if isLocalDomain(host) {
			h, err := resolveLocalDomain(host, timeout)
			if err == nil {
				addr = fmt.Sprintf("%s:%s", h, port)
				log.Debugf("resolved: [%s] -> [%s]", host, h)
			} else {
				log.Errorf("can't resolve %s: %v", addr, err)
			}
		}

		return dialer.DialContext(ctx, network, addr)
	}
}

func isLocalDomain(host string) bool {
	return strings.HasSuffix(host, ".local")
}

func resolveLocalDomain(host string, timeout time.Duration) (string, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return "", fmt.Errorf("can't create resolver: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()

	entries := make(chan *zeroconf.ServiceEntry)

	err = resolver.Lookup(ctx, strings.TrimSuffix(host, ".local"), "_http._tcp", "local.", entries)
	if err != nil {
		return "", err
	}

	entry := <-entries

	if entry != nil {
		return entry.AddrIPv4[0].String(), nil
	}

	return "", errors.New("resolver timeout: " + host)
}
