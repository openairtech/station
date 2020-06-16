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
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
)

// Close given closer without error checking
func CloseQuietly(closer io.Closer) {
	_ = closer.Close()
}

// Parse given address and split it to host and port (if any)
func ParseAddr(addr string) (host, port string) {
	e := strings.SplitN(addr, ":", 2)

	if len(e) == 1 {
		return e[0], ""
	}

	return e[0], e[1]
}

// Get wireless interface (with name starting with 'wl') MAC address
// or empty string if no wireless interface found
func WirelessInterfaceMacAddr() string {
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		if strings.HasPrefix(iface.Name, "wl") {
			return iface.HardwareAddr.String()
		}
	}
	return ""
}

// Compute SHA1 checksum for given string
func Sha1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// Convert reference to float32 to its string representation
func Float32RefToString(r *float32) string {
	if r == nil {
		return ""
	}

	return fmt.Sprintf("%.1f", *r)
}

// Round float32 to given number of decimal places
func Float32Round(x float32, places int) float32 {
	pow := math.Pow(10, float64(places))
	return float32(math.Round(pow*float64(x)) / pow)
}

// Round referenced float32 to given number of decimal places
func Float32RefRound(r *float32, places int) float32 {
	if r == nil {
		return 0
	}
	return Float32Round(*r, places)
}

// Convert string slice s to comma-separated values string
func SliceToString(s []string) string {
	return strings.Join(s, ", ")
}

// Check given string a is contained in string slice s
func StringInSlice(a string, s []string) bool {
	for _, b := range s {
		if b == a {
			return true
		}
	}
	return false
}
