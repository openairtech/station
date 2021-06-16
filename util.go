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
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// CloseQuietly closes given closer without error checking
func CloseQuietly(closer io.Closer) {
	_ = closer.Close()
}

// ParseAddr parses given address and split it to host and port (if any)
func ParseAddr(addr string) (host, port string) {
	e := strings.SplitN(addr, ":", 2)

	if len(e) == 1 {
		return e[0], ""
	}

	return e[0], e[1]
}

// WirelessInterfaceMacAddr gets wireless interface (with name starting with 'wl') MAC address
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

// Sha1 computes SHA1 checksum for given string
func Sha1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// Float32RefToString converts reference to float32 to its string representation
func Float32RefToString(r *float32) string {
	if r == nil {
		return ""
	}

	return fmt.Sprintf("%.1f", *r)
}

// Float32Round rounds float32 to given number of decimal places
func Float32Round(x float32, places int) float32 {
	pow := math.Pow(10, float64(places))
	return float32(math.Round(pow*float64(x)) / pow)
}

// Float32RefRound rounds referenced float32 to the given number of decimal places
func Float32RefRound(r *float32, places int) float32 {
	if r == nil {
		return 0
	}
	return Float32Round(*r, places)
}

// SliceToString convert string slice s to the comma-separated values string
func SliceToString(s []string) string {
	return strings.Join(s, ", ")
}

// StringInSlice checks given string a is contained in the string slice s
func StringInSlice(a string, s []string) bool {
	for _, b := range s {
		if b == a {
			return true
		}
	}
	return false
}

// SubString extracts substring from input string start position with given length
func SubString(input string, start int, length int) string {
	asRunes := []rune(input)

	if start >= len(asRunes) {
		return ""
	}

	if start+length > len(asRunes) {
		length = len(asRunes) - start
	}

	return string(asRunes[start : start+length])
}

// TruncateString checks length of string a and if needed truncates it
// to the given length by adding an ellipsis to the end
func TruncateString(a string, length int) string {
	if len(a) > length {
		return fmt.Sprintf("%s…", SubString(a, 0, length))
	}
	return a
}

// Execute command using system shell with timeout
func Execute(command string, timeout time.Duration) error {
	// Set shell and command execution flag
	shell, flag := "/bin/sh", "-c"

	// Create command execution and get stdout/stderr
	cmd := exec.Command(shell, flag, command)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer CloseQuietly(stdout)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer CloseQuietly(stderr)

	timeoutFlag := false
	var timeoutFlagLock sync.RWMutex

	// Start command execution
	if err := cmd.Start(); err != nil {
		return err
	}

	// Create command result channel
	result := make(chan error, 1)
	defer close(result)
	go func() {
		err := cmd.Wait()
		timeoutFlagLock.RLock()
		defer timeoutFlagLock.RUnlock()
		if !timeoutFlag {
			result <- err
		}
	}()

	// Log stdout/stderr
	outScanner := bufio.NewScanner(stdout)
	go func() {
		for outScanner.Scan() {
			log.Debugf("STDOUT: %s", outScanner.Text())
		}
		if err := outScanner.Err(); err != nil {
			log.Errorf("STDOUT: error: %v", err)
		}
	}()
	errScanner := bufio.NewScanner(stderr)
	go func() {
		for errScanner.Scan() {
			log.Debugf("STDERR: %s", errScanner.Text())
		}
		if err := errScanner.Err(); err != nil {
			log.Errorf("STDERR: error: %v", err)
		}
	}()

	// Wait for result indefinitely if no timeout set
	if timeout == 0 {
		return <-result
	}

	// Wait for result for given duration if timeout set
	select {
	case <-time.After(timeout):
		timeoutFlagLock.Lock()
		defer timeoutFlagLock.Unlock()
		timeoutFlag = true
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				log.Errorf("timeout (%v): %q, not killed: %v", timeout, command, err)
			} else {
				log.Warningf("timeout (%v): %q, killed", timeout, command)
			}
		} else {
			log.Warningf("timeout (%v): %q, nothing to kill", timeout, command)
		}
		return fmt.Errorf("timeout (%v): %q", timeout, command)
	case err := <-result:
		return err
	}
}
