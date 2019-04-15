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
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	goquery "github.com/google/go-querystring/query"
)

type Rattle struct {
	// http Client for doing Rattles
	httpClient *http.Client
	// HTTP method (GET, POST, etc.)
	method string
	// raw url string for Rattles
	rawURL string
	// stores key-values pairs to add to Rattle's Headers
	header http.Header
	// url tagged query structs
	parameters []interface{}
	// body provider
	bodyProvider BodyProvider
	// Rattle configs
	config Config
	// http.Response
	resp *http.Response
}

func New(config *Config) *Rattle {
	if config == nil {
		config = NewConfig()
	}
	transport := &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(network, addr, config.HTTPTimeout.ConnectTimeout)
			if err != nil {
				return nil, err
			}
			return newTimeoutConn(conn, config.HTTPTimeout), nil
		},
		ResponseHeaderTimeout: config.HTTPTimeout.HeaderTimeout,
		MaxIdleConnsPerHost:   2000,
	}

	// Proxy
	if config.UseProxy {
		proxyURL, err := url.Parse(config.ProxyHost)
		if err == nil {
			if config.IsAuthProxy {
				proxyURL.User = url.UserPassword(config.ProxyUser, config.ProxyPassword)
			}
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	return &Rattle{
		httpClient: &http.Client{Transport: transport},
		method:     GET,
		header:     make(http.Header),
		parameters: make([]interface{}, 0),
	}
}

func (r *Rattle) New() *Rattle {
	// copy Headers pairs into new Header map
	headerCopy := make(http.Header)
	for k, v := range r.header {
		headerCopy[k] = v
	}
	return &Rattle{
		httpClient:   r.httpClient,
		method:       r.method,
		rawURL:       r.rawURL,
		header:       headerCopy,
		parameters:   append([]interface{}{}, r.parameters...),
		bodyProvider: r.bodyProvider,
	}
}

// Base sets the rawURL. If you intend to extend the url with Path,
// baseUrl should be specified with a trailing slash.
func (r *Rattle) BaseURL(rawURL string) *Rattle {
	r.rawURL = rawURL
	return r
}

// Head sets the Request method to HEAD and sets the given pathURL.
func (r *Rattle) Head(pathURL string) *Rattle {
	r.method = HEAD
	return r.setPath(pathURL)
}

// Get sets the Request method to GET and sets the given pathURL.
func (r *Rattle) Get(pathURL string) *Rattle {
	r.method = GET
	return r.setPath(pathURL)
}

// Post sets the Request method to POST and sets the given pathURL.
func (r *Rattle) Post(pathURL string) *Rattle {
	r.method = POST
	return r.setPath(pathURL)
}

// Put sets the Request method to PUT and sets the given pathURL.
func (r *Rattle) Put(pathURL string) *Rattle {
	r.method = PUT
	return r.setPath(pathURL)
}

// Patch sets the Request method to PATCH and sets the given pathURL.
func (r *Rattle) Patch(pathURL string) *Rattle {
	r.method = PATCH
	return r.setPath(pathURL)
}

// Delete sets the Sling method to DELETE and sets the given pathURL.
func (r *Rattle) Delete(pathURL string) *Rattle {
	r.method = DELETE
	return r.setPath(pathURL)
}

// Options sets the Sling method to OPTIONS and sets the given pathURL.
func (r *Rattle) Options(pathURL string) *Rattle {
	r.method = OPTIONS
	return r.setPath(pathURL)
}

// Path extends the rawURL with the given path by resolving the reference to
// an absolute URL. If parsing errors occur, the rawURL is left unmodified.
func (r *Rattle) setPath(path string) *Rattle {
	hostURL, hostErr := url.Parse(r.rawURL)
	pathURL, pathErr := url.Parse(path)
	if hostErr == nil && pathErr == nil {
		r.rawURL = hostURL.ResolveReference(pathURL).String()
	}
	return r
}

// SetHeader sets the key, value pair in Headers, replacing existing values
// associated with key. Header keys are canonicalized.
func (r *Rattle) SetHeader(key, value string) *Rattle {
	r.header.Set(key, value)
	return r
}

// SetBasicAuth sets the Authorization header to use HTTP Basic Authentication
// with the provided username and password. With HTTP Basic Authentication
// the provided username and password are not encrypted.
func (r *Rattle) SetBasicAuth(username, password string) *Rattle {
	return r.SetHeader("Authorization", "Basic "+genBasicAuth(username, password))
}

// genBasicAuth returns the Host64 encoded username:password for basic auth copied
// from net/http.
func genBasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// BodyProvider sets body provider.
func (r *Rattle) setbodyProvider(body BodyProvider) *Rattle {
	if body == nil {
		return r
	}
	r.bodyProvider = body
	return r
}

// Body sets the Rattle plain body
func (r *Rattle) BodyOriginal(bodyOriginal io.Reader) *Rattle {
	if bodyOriginal == nil {
		return r
	}
	return r.setbodyProvider(bodyOriginalProvider{body: bodyOriginal})
}

// BodyJSON sets the json body
func (r *Rattle) BodyJSON(bodyJSON interface{}, escapeHTML bool) *Rattle {
	if bodyJSON == nil {
		return r
	}
	return r.setbodyProvider(bodyProviderJson{body: bodyJSON, escapeHTML: escapeHTML})
}

// BodyForm sets the form body
func (r *Rattle) BodyForm(bodyForm interface{}) *Rattle {
	if bodyForm == nil {
		return r
	}
	return r.setbodyProvider(bodyProviderForm{body: bodyForm})
}

// BodyFile sets the send file. The value pointed to by the bodyForm
func (r *Rattle) BodyFile(fields interface{}, file bodyProviderFileStruct) *Rattle {
	return r.setbodyProvider(bodyProviderFile{body: fields, file: file})
}

func NewBodyFile(fieldname, filename string, file io.Reader) bodyProviderFileStruct {
	return bodyProviderFileStruct{
		fieldName: fieldname,
		fileName:  filename,
		file:      file,
	}
}

// GetRequest returns a new http.Request created with the request properties.
// Returns any errors parsing the rawURL, encoding query structs, encoding
// the body, or creating the http.Request.
func (r *Rattle) GetRequest() (*http.Request, error) {
	reqURL, err := url.Parse(r.rawURL)
	if err != nil {
		return nil, err
	}

	err = genQuery(reqURL, r.parameters)
	if err != nil {
		return nil, err
	}

	var body io.Reader
	var reqContentType string
	if r.bodyProvider != nil {
		body, reqContentType, err = r.bodyProvider.GetBody()
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(r.method, reqURL.String(), body)
	if err != nil {
		return nil, err
	}
	if !r.config.ReUseTCP {
		req.Close = true
	}

	setHeaders(req, r.header)
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.119 Safari/537.36")
	}

	if reqContentType != "" {
		req.Header.Set(contentType, reqContentType)
	} else {
		req.Header.Del(contentType)
	}

	return req, err
}

// genQuery parses url tagged query structs using go-querystring to
// encode them to url.Values and format them onto the url.RawQuery. Any
// query parsing or encoding errors are returned.
func genQuery(reqURL *url.URL, params []interface{}) error {
	urlValues, err := url.ParseQuery(reqURL.RawQuery)
	if err != nil {
		return err
	}
	// encodes query structs into a url.Values map and merges maps
	for _, param := range params {
		queryValues, err := goquery.Values(param)
		if err != nil {
			return err
		}
		for key, values := range queryValues {
			for _, value := range values {
				urlValues.Add(key, value)
			}
		}
	}
	// url.Values format to a sorted "url encoded" string, e.g. "key=val&foo=bar"
	reqURL.RawQuery = urlValues.Encode()
	return nil
}

// setHeaders adds the key, value pairs from the given http.Header to the
// Rattle. Values for existing keys are appended to the keys values.
func setHeaders(req *http.Request, headers http.Header) {
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
}

// return then response
func (r *Rattle) GetResponse() *http.Response {
	return r.resp
}

// Send is shorthand for calling Rattle and Do.
func (r *Rattle) Send() ([]byte, int, error) {
	req, err := r.GetRequest()
	if err != nil {
		return nil, 0, err
	}
	return r.Do(req)
}

// Do sends an HTTP Rattle and returns the response.
// are write into the value pointed to by result.
// Any error sending the Rattle response is returned.
func (r *Rattle) Do(req *http.Request) ([]byte, int, error) {
	resp, err := r.httpClient.Do(req)
	defer func() {
		if resp != nil {
			resp.Close = true
			resp.Body.Close()
		}
	}()
	if err != nil {
		if r.config.RetryTimes > 0 {
			var retryTimes uint = 0
			retryTicker := time.NewTicker(r.config.HTTPTimeout.ConnectTimeout)
			for range retryTicker.C {
				if retryTimes >= r.config.RetryTimes {
					retryTicker.Stop()
					err = fmt.Errorf("retryTimes:%v %s", retryTimes, err.Error())
					return nil, 0, err
				}
				retryTimes++
				resp, err = r.httpClient.Do(req)
				if err == nil {
					retryTicker.Stop()
					break
				}
			}
		} else {
			return nil, 0, err
		}
	}
	r.resp = resp

	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("%s", resp.Status)
	}
	res, err := ioutil.ReadAll(resp.Body)

	return res, resp.StatusCode, err
}

// AddQuery add queries for GET request
func (r *Rattle) AddQuery(params interface{}) *Rattle {
	if params != nil {
		r.parameters = append(r.parameters, params)
	}
	return r
}
