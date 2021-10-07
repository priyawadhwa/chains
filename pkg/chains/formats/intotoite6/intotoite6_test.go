/*
Copyright 2021 The Tekton Authors

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

package intotoite6

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	logtesting "knative.dev/pkg/logging/testing"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/config"

	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

var e1BuildStart = time.Unix(1617011400, 0)
var e1BuildFinished = time.Unix(1617011415, 0)

func TestPayload(t *testing.T) {
	tests := []struct {
		name     string
		testdata string
		builder  string
		expected in_toto.ProvenanceStatement
	}{
		{
			name:     "test1",
			testdata: "testdata/testdata1.json",
			builder:  "test_builder-1",
			expected: expected1,
		},
		{
			name:     "test2",
			testdata: "testdata/testdata2.json",
			builder:  "test_builder-2",
			expected: expected2,
		},
		{
			name:     "test multiple subjects",
			testdata: "testdata/testdata3.json",
			builder:  "test_builder-multiple",
			expected: expectedMultipleSubjects,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var tr v1beta1.TaskRun

			contents, err := ioutil.ReadFile(test.testdata)
			if err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(contents, &tr); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			cfg := config.Config{
				Builder: config.BuilderConfig{
					ID: test.builder,
				},
			}
			i, _ := NewFormatter(cfg, logtesting.TestLogger(t))

			got, err := i.CreatePayload(&tr)

			if err != nil {
				t.Errorf("unexpected error: %s", err.Error())
			}
			if diff := cmp.Diff(test.expected, got); diff != "" {
				t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
			}
		})
	}
}

var expected1 = in_toto.ProvenanceStatement{
	StatementHeader: in_toto.StatementHeader{
		Type:          in_toto.StatementInTotoV01,
		PredicateType: in_toto.PredicateSLSAProvenanceV01,
		Subject: []in_toto.Subject{
			{
				Name: "gcr.io/myimage",
				Digest: in_toto.DigestSet{
					"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
				},
			},
		},
	},
	Predicate: in_toto.ProvenancePredicate{
		Metadata: &in_toto.ProvenanceMetadata{
			BuildStartedOn:  &e1BuildStart,
			BuildFinishedOn: &e1BuildFinished,
		},
		Materials: []in_toto.ProvenanceMaterial{
			{
				URI: "git+https://git.test.com",
				Digest: map[string]string{
					"git_commit": "abcd",
				},
			},
		},
		Builder: in_toto.ProvenanceBuilder{
			ID: "test_builder-1",
		},
		Recipe: in_toto.ProvenanceRecipe{
			Type: tektonID,
			Arguments: []string{
				"IMAGE={string test.io/test/image []}", "CHAINS-GIT_COMMIT={string abcd []}",
				"CHAINS-GIT_URL={string https://git.test.com []}",
				"filename={string /bin/ls []}",
			},
			Environment: []Step{
				{
					Container: "step1",
					Image:     "docker-pullable://gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
				},
				{
					Container: "step2",
					Image:     "docker-pullable://gcr.io/test2/test2@sha256:4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac",
				},
				{
					Container: "step3",
					Image:     "docker-pullable://gcr.io/test3/test3@sha256:f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478",
				},
			},
		},
	},
}

var expected2 = in_toto.ProvenanceStatement{
	StatementHeader: in_toto.StatementHeader{
		Type:          in_toto.StatementInTotoV01,
		PredicateType: in_toto.PredicateSLSAProvenanceV01,
		Subject: []in_toto.Subject{
			{
				Name: "gcr.io/my/image",
				Digest: map[string]string{
					"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
				},
			},
		},
	},
	Predicate: in_toto.ProvenancePredicate{
		Metadata: &in_toto.ProvenanceMetadata{},
		Builder: in_toto.ProvenanceBuilder{
			ID: "test_builder-2",
		},
		Recipe: in_toto.ProvenanceRecipe{
			Type:      tektonID,
			Arguments: []string(nil),
			Environment: []Step{
				{
					Container: "step1",
					Image:     "docker-pullable://gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
				},
			},
		},
	},
}

var expectedMultipleSubjects = in_toto.ProvenanceStatement{
	StatementHeader: in_toto.StatementHeader{
		Type:          in_toto.StatementInTotoV01,
		PredicateType: in_toto.PredicateSLSAProvenanceV01,
		Subject: []in_toto.Subject{
			{
				Name: "gcr.io/myimage1",
				Digest: map[string]string{
					"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
				},
			},
			{
				Name: "gcr.io/myimage2",
				Digest: map[string]string{
					"sha256": "daa1a56e13c85cf164e7d9e595006649e3a04c47fe4a8261320e18a0bf3b0367",
				},
			},
		},
	},
	Predicate: in_toto.ProvenancePredicate{
		Metadata: &in_toto.ProvenanceMetadata{},
		Builder: in_toto.ProvenanceBuilder{
			ID: "test_builder-multiple",
		},
		Recipe: in_toto.ProvenanceRecipe{
			Type:      tektonID,
			Arguments: []string(nil),
			Environment: []Step{
				{
					Container: "step1",
					Image:     "docker-pullable://gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
				},
			},
		},
	},
}

func TestInTotoIte6_CreatePayloadNilTaskRef(t *testing.T) {
	var tr v1beta1.TaskRun
	contents, err := ioutil.ReadFile("testdata/testdata1.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(contents, &tr); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	tr.Spec.TaskRef = nil
	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "testid",
		},
	}
	f, _ := NewFormatter(cfg, logtesting.TestLogger(t))

	p, err := f.CreatePayload(&tr)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}

	ps := p.(in_toto.ProvenanceStatement)
	if diff := cmp.Diff(tr.Name, ps.Predicate.Recipe.EntryPoint); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestNewFormatter(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		cfg := config.Config{
			Builder: config.BuilderConfig{
				ID: "testid",
			},
		}
		f, err := NewFormatter(cfg, logtesting.TestLogger(t))
		if f == nil {
			t.Error("Failed to create formatter")
		}
		if err != nil {
			t.Errorf("Error creating formatter: %s", err)
		}
	})
}

func TestCreatePayloadError(t *testing.T) {
	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "testid",
		},
	}
	f, _ := NewFormatter(cfg, logtesting.TestLogger(t))

	t.Run("Invalid type", func(t *testing.T) {
		p, err := f.CreatePayload("not a task ref")

		if p != nil {
			t.Errorf("Unexpected payload")
		}
		if err == nil {
			t.Errorf("Expected error")
		} else {
			if err.Error() != "intoto does not support type: not a task ref" {
				t.Errorf("wrong error returned: '%s'", err.Error())
			}
		}
	})

}

func TestCorrectPayloadType(t *testing.T) {
	var i InTotoIte6
	if i.Type() != formats.PayloadTypeInTotoIte6 {
		t.Errorf("Invalid type returned: %s", i.Type())
	}
}
