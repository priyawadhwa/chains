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

package api

import (
	"io/ioutil"
	"path/filepath"

	"github.com/go-openapi/runtime/middleware"
	"github.com/pkg/errors"
	"github.com/tektoncd/chains/pkg/api/generated/models"
	"github.com/tektoncd/chains/pkg/api/generated/restapi/operations/entry"
)

func GetEntryHandler(params entry.GetEntryParams) middleware.Responder {
	e, err := getEntry(params.PodName)
	if err != nil {
		return entry.NewGetEntryDefault(1).WithPayload(&models.Error{Message: err.Error()})
	}
	return entry.NewGetEntryOK().WithPayload(e)
}

func AddEntryHandler(params entry.AddEntryParams) middleware.Responder {
	if err := addEntry(params.Query); err != nil {
		return entry.NewAddEntryDefault(1).WithPayload(&models.Error{Message: err.Error()})
	}
	return entry.NewAddEntryOK()
}

func getEntry(podName string) (*models.Entry, error) {
	f := filepath.Join(api.storagePath, podName)
	contents, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, errors.Wrapf(err, "getting entry for %s", podName)
	}
	var e models.Entry
	if err := e.UnmarshalBinary(contents); err != nil {
		return nil, errors.Wrap(err, "unmarshal")
	}
	return &e, nil
}

func addEntry(e *models.Entry) error {
	if e == nil {
		return nil
	}
	f := filepath.Join(api.storagePath, *e.PodName)
	contents, err := e.MarshalBinary()
	if err != nil {
		return errors.Wrap(err, "marshal binary")
	}
	if err := ioutil.WriteFile(f, contents, 0644); err != nil {
		return errors.Wrap(err, "writing file")
	}
	return nil
}
