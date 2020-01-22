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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

var httpClient http.Client

func InitHttp(timeout time.Duration) {
	httpClient = http.Client{
		Timeout: timeout,
	}
}

func HttpGetData(url string, res interface{}) error {
	r, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer CloseQuietly(r.Body)

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("%d: %s", r.StatusCode, b)
	}

	return json.Unmarshal(b, &res)
}

func HttpPostData(url string, headers map[string]interface{}, data, res interface{}) error {
	jd, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jd))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Add(k, fmt.Sprintf("%v", v))
	}

	r, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer CloseQuietly(r.Body)

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	if r.StatusCode < http.StatusOK || r.StatusCode > http.StatusIMUsed {
		return fmt.Errorf("%d: %s", r.StatusCode, b)
	}

	return json.Unmarshal(b, &res)
}
