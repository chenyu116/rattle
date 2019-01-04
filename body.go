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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	goquery "github.com/google/go-querystring/query"
)

// BodyProvider provides Body content for http.Request attachment.
type BodyProvider interface {
	// Body returns the io.Reader body.
	GetBody() (io.Reader, string, error)
}

// bodyOriginalProvider provides the wrapped body value as a Body for requests.
type bodyOriginalProvider struct {
	body io.Reader
}

func (p bodyOriginalProvider) GetBody() (io.Reader, string, error) {
	return p.body, "", nil
}

// jsonBodyProvider encodes a JSON tagged struct value as a Body for requests.
// See https://golang.org/pkg/encoding/json/#MarshalIndent for details.
type bodyProviderJson struct {
	body interface{}
	escapeHTML bool
}

func (p bodyProviderJson) GetBody() (io.Reader, string, error) {
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(p.escapeHTML)
	err := encoder.Encode(p.body)
	if err != nil {
		return nil, "", err
	}
	return buf, contentTypeJson, nil
}

// formBodyProvider encodes a url tagged struct value as Body for requests.
// See https://godoc.org/github.com/google/go-querystring/query for details.
type bodyProviderForm struct {
	body interface{}
}

func (p bodyProviderForm) GetBody() (io.Reader, string, error) {
	values, err := goquery.Values(p.body)
	if err != nil {
		return nil, "", err
	}
	return strings.NewReader(values.Encode()), contentTypeForm, nil
}

type bodyProviderFileStruct struct {
	fileName  string
	fieldName string
	content   io.Reader
}

type bodyProviderFile struct {
	body interface{}
	file    bodyProviderFileStruct
}

func (p bodyProviderFile) GetBody() (io.Reader, string, error) {
	if p.file.fileName == "" {
		return nil, "", fmt.Errorf("field not defined %s", "fileName")
	}
	if p.file.fieldName == "" {
		p.file.fieldName = p.file.fileName
	}
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	fw, err := writer.CreateFormFile(p.file.fieldName, p.file.fileName)
	if err != nil {
		return nil, "", fmt.Errorf("CreateFormFile %v", err)
	}
	_, err = io.Copy(fw, p.file.content)
	if err != nil {
		return nil, "", fmt.Errorf("copying fileWriter %v", err)
	}

	if p.body != nil {
		values, err := goquery.Values(p.body)
		if err != nil {
			return nil, "", err
		}
		for k, _ := range values {
			err = writer.WriteField(k, values.Get(k))
			if err != nil {
				return nil, "", fmt.Errorf("WriteField err:%v", err)
			}
		}
	}

	err = writer.Close() // close writer before POST request
	if err != nil {
		return nil, "", fmt.Errorf("writerClose: %v", err)
	}

	return body, writer.FormDataContentType(), nil
}
