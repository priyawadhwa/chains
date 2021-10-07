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
	"fmt"
	"sort"
	"strings"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/provenance"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
)

const (
	tektonID    = "https://tekton.dev/attestations/chains@v1"
	commitParam = "CHAINS-GIT_COMMIT"
	urlParam    = "CHAINS-GIT_URL"
)

type InTotoIte6 struct {
	builderID string
	logger    *zap.SugaredLogger
}

func NewFormatter(cfg config.Config, logger *zap.SugaredLogger) (formats.Payloader, error) {
	return &InTotoIte6{
		builderID: cfg.Builder.ID,
		logger:    logger,
	}, nil
}

func (i *InTotoIte6) Wrap() bool {
	return true
}

func (i *InTotoIte6) CreatePayload(obj interface{}) (interface{}, error) {
	var tr *v1beta1.TaskRun
	switch v := obj.(type) {
	case *v1beta1.TaskRun:
		tr = v
	default:
		return nil, fmt.Errorf("intoto does not support type: %s", v)
	}
	return i.generateAttestationFromTaskRun(tr)
}

// generateAttestationFromTaskRun translates a Tekton TaskRun into an InToto ite6 provenance
// attestation.
// Spec: https://github.com/in-toto/attestation/blob/main/spec/predicates/provenance.md
// At a high level, the mapping looks roughly like:
// 	Configured builder id -> Builder.Id
// 	Results with name *_DIGEST -> Subject
// 	Step containers -> Materials
// 	Params with name CHAINS-GIT_* -> Materials and recipe.materials

// 	tekton-chains -> Recipe.type
// 	Taskname -> Recipe.entry_point
func (i *InTotoIte6) generateAttestationFromTaskRun(tr *v1beta1.TaskRun) (interface{}, error) {
	att := intoto.ProvenanceStatement{
		StatementHeader: intoto.StatementHeader{
			Type:          intoto.StatementInTotoV01,
			PredicateType: intoto.PredicateSLSAProvenanceV01,
			Subject:       provenance.GetSubjectDigests(tr, i.logger),
		},
		Predicate: intoto.ProvenancePredicate{
			Metadata:  metadata(tr),
			Materials: materials(tr),
			Builder: intoto.ProvenanceBuilder{
				ID: i.builderID,
			},
			Recipe: recipe(tr),
		},
	}

	return att, nil
}

func metadata(tr *v1beta1.TaskRun) *intoto.ProvenanceMetadata {
	m := &intoto.ProvenanceMetadata{}
	if tr.Status.StartTime != nil {
		m.BuildStartedOn = &tr.Status.StartTime.Time
	}
	if tr.Status.CompletionTime != nil {
		m.BuildFinishedOn = &tr.Status.CompletionTime.Time
	}
	return m
}

// materials will add the following to the attestation materials:
// 1. Any specification for git
func materials(tr *v1beta1.TaskRun) []intoto.ProvenanceMaterial {
	var m []intoto.ProvenanceMaterial
	gitCommit, gitURL := gitInfo(tr)

	// Store git rev as Materials and Recipe.Material
	if gitCommit != "" && gitURL != "" {
		m = append(m, intoto.ProvenanceMaterial{
			URI:    gitURL,
			Digest: map[string]string{"git_commit": gitCommit},
		})
	}
	sort.Slice(m, func(i, j int) bool {
		return m[i].URI <= m[j].URI
	})
	return m
}

type Step struct {
	Container  interface{} `json:"container"`
	Image      interface{} `json:"image"`
	Entrypoint string      `json:"entryPoint"`
}

func recipe(tr *v1beta1.TaskRun) intoto.ProvenanceRecipe {

	var steps []Step
	for _, s := range provenance.Steps(tr) {
		env, ok := s.Environment.(map[string]interface{})
		if !ok {
			continue
		}
		steps = append(steps, Step{
			Container:  env["container"],
			Image:      env["image"],
			Entrypoint: s.EntryPoint,
		})
	}
	return intoto.ProvenanceRecipe{
		Type:        tektonID,
		Arguments:   provenance.Params(tr),
		Environment: steps,
	}
}

func (i *InTotoIte6) Type() formats.PayloadType {
	return formats.PayloadTypeInTotoIte6
}

// gitInfo scans over the input parameters and looks for parameters
// with specified names.
func gitInfo(tr *v1beta1.TaskRun) (commit string, url string) {
	// Scan for git params to use for materials
	for _, p := range tr.Spec.Params {
		if p.Name == commitParam {
			commit = p.Value.StringVal
			continue
		}
		if p.Name == urlParam {
			url = p.Value.StringVal
			// make sure url is PURL (git+https)
			if !strings.HasPrefix(url, "git+") {
				url = "git+" + url
			}
		}
	}
	return
}
