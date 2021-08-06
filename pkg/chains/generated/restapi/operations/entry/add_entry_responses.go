// Code generated by go-swagger; DO NOT EDIT.

// Copyright 2021 The Tekton Authors
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
//

package entry

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/tektoncd/chains/pkg/chains/generated/models"
)

// AddEntryOKCode is the HTTP code returned for type AddEntryOK
const AddEntryOKCode int = 200

/*AddEntryOK It worked!

swagger:response addEntryOK
*/
type AddEntryOK struct {
}

// NewAddEntryOK creates AddEntryOK with default headers values
func NewAddEntryOK() *AddEntryOK {

	return &AddEntryOK{}
}

// WriteResponse to the client
func (o *AddEntryOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(200)
}

/*AddEntryDefault There was an internal error in the server while processing the request

swagger:response addEntryDefault
*/
type AddEntryDefault struct {
	_statusCode int

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewAddEntryDefault creates AddEntryDefault with default headers values
func NewAddEntryDefault(code int) *AddEntryDefault {
	if code <= 0 {
		code = 500
	}

	return &AddEntryDefault{
		_statusCode: code,
	}
}

// WithStatusCode adds the status to the add entry default response
func (o *AddEntryDefault) WithStatusCode(code int) *AddEntryDefault {
	o._statusCode = code
	return o
}

// SetStatusCode sets the status to the add entry default response
func (o *AddEntryDefault) SetStatusCode(code int) {
	o._statusCode = code
}

// WithPayload adds the payload to the add entry default response
func (o *AddEntryDefault) WithPayload(payload *models.Error) *AddEntryDefault {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the add entry default response
func (o *AddEntryDefault) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *AddEntryDefault) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(o._statusCode)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
