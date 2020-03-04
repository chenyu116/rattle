/*
   Copyright [2018] [Chen.Yu]

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package rattle

import (
	"reflect"
	"testing"
)

type TestParams struct {
	Name  string `url:"name"`
	Count int    `url:"count"`
}

var params = TestParams{Name: "recent", Count: 25}

func TestNew(t *testing.T) {
	rattle := New()
	if rattle.header == nil {
		t.Errorf("Header map not initialized with make")
	}
	if rattle.parameters == nil {
		t.Errorf("parameters not initialized with make")
	}
}

func TestRattleChild(t *testing.T) {
	Rattle := New().BaseURL("http://example.com").AddQuery(params)
	child := Rattle.New()
	if child.httpClient != Rattle.httpClient {
		t.Errorf("expected %v, got %v", Rattle.httpClient, child.httpClient)
	}
	if child.method != Rattle.method {
		t.Errorf("expected %s, got %s", Rattle.method, child.method)
	}
	if child.rawURL != Rattle.rawURL {
		t.Errorf("expected %s, got %s", Rattle.rawURL, child.rawURL)
	}
	// Header should be a copy of parent Rattle header. For example, calling
	// baseRattle.Add("k","v") should not mutate previously created child Rattles
	if Rattle.header != nil {
		// struct literal cases don't init Header in usual way, skip header check
		if !reflect.DeepEqual(Rattle.header, child.header) {
			t.Errorf("not DeepEqual: expected %v, got %v", Rattle.header, child.header)
		}
		Rattle.header.Add("K", "V")
		if child.header.Get("K") != "" {
			t.Errorf("child.header was a reference to original map, should be copy")
		}
	}
	// parameters slice should be a new slice with a copy of the contents
	if len(Rattle.parameters) > 0 {
		// mutating one slice should not mutate the other
		child.parameters[0] = nil
		if Rattle.parameters[0] == nil {
			t.Errorf("child.parameters was a re-slice, expected slice with copied contents")
		}
	}
	// body should be copied
	if child.bodyProvider != Rattle.bodyProvider {
		t.Errorf("expected %v, got %v", Rattle.bodyProvider, child.bodyProvider)
	}
}

func TestProxy(t *testing.T) {
	config := NewConfig()
	config.UseProxy = true
	config.ProxyHost = "http://127.0.0.1:1080"
	Rattle := New(config).BaseURL("http://example.com").AddQuery(params)
	_, _, err := Rattle.Send()
	if err != nil {
		t.Errorf("expected %v", err)
	}
}
func TestRequest_query(t *testing.T) {
	cases := []struct {
		rattle      *Rattle
		expectedURL string
	}{
		{New().Get("http://example.com").AddQuery(params), "http://example.com?count=25&name=recent"},
		{New().Get("http://example.com").AddQuery(params).New(), "http://example.com?count=25&name=recent"},
	}
	for _, c := range cases {
		req, _ := c.rattle.GetRequest()
		if req.URL.String() != c.expectedURL {
			t.Errorf("expected url %s, got %s for %+v", c.expectedURL, req.URL.String(), c.rattle)
		}
	}
}

func TestRequest_headers(t *testing.T) {
	cases := []struct {
		rattle         *Rattle
		expectedHeader map[string][]string
	}{
		{New().SetHeader("authorization", "OAuth key=\"value\""), map[string][]string{"Authorization": []string{"OAuth key=\"value\""}}},
		// header keys should be canonicalized
		{New().New().SetHeader("authorization", "OAuth key=\"value\""), map[string][]string{"Authorization": []string{"OAuth key=\"value\""}}},
	}
	for _, c := range cases {
		req, _ := c.rattle.GetRequest()
		// type conversion from Header to alias'd map for deep equality comparison
		headerMap := map[string][]string(req.Header)
		if !reflect.DeepEqual(c.expectedHeader, headerMap) {
			t.Errorf("not DeepEqual: expected %v, got %v", c.expectedHeader, headerMap)
		}
	}
}
