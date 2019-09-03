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

// Compute SHA1 checksum for given string
func Sha1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// Convert reference to float32 to its string representation
func Float32RefToString(r *float32) string {
	if r == nil {
		return "<none>"
	}

	return fmt.Sprintf("%.1f", *r)
}

// Round float32 x to given number of decimal places
func Float32Round(x float32, places int) float32 {
	pow := math.Pow(10, float64(places))
	return float32(math.Round(pow*float64(x)) / pow)
}
