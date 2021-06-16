/*
Copyright 2020 The Tekton Authors
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

package tekton

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/config"
	"go.uber.org/zap"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// Tekton is a formatter that just captures the TaskRun Status with no modifications.
type Tekton struct {
	logger       *zap.SugaredLogger
	spireEnabled bool
}

func NewFormatter(cfg config.Config, l *zap.SugaredLogger) (formats.Payloader, error) {
	return &Tekton{
		logger: l,
	}, nil
}

// CreatePayload implements the Payloader interface.
func (i *Tekton) CreatePayload(obj interface{}) (interface{}, error) {
	switch v := obj.(type) {
	case *v1beta1.TaskRun:
		if err := i.verifySpire(v); err != nil {
			return nil, errors.Wrap(err, "verifying spire")
		}
		return v.Status, nil
	default:
		return nil, fmt.Errorf("unsupported type %s", v)
	}
}

// check if we even have the SVID cert, return if not
// parse the SVID cert
// verify the provided signatures against the cert
func (i *Tekton) verifySpire(tr *v1beta1.TaskRun) error {
	results := tr.Status.TaskRunResults
	if err := i.validateResults(results); err != nil {
		return errors.Wrap(err, "validating results")
	}
	i.logger.Info("Successfully verified SPIRE")
	return nil
}

func (i *Tekton) validateResults(rs []v1beta1.TaskRunResult) error {
	resultMap := map[string]v1beta1.TaskRunResult{}
	for _, r := range rs {
		resultMap[r.Name] = r
	}
	svid, ok := resultMap["SVID"]
	if !ok {
		i.logger.Error("No SVID certificate found, skipping SPIRE verification")
		return nil
	}
	block, _ := pem.Decode([]byte(svid.Value))
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("invalid SVID: %s", err)
	}

	for key := range resultMap {
		if strings.HasSuffix(key, ".sig") {
			continue
		}
		if key == "SVID" {
			continue
		}
		if err := verifyOne(cert.PublicKey, key, resultMap); err != nil {
			return err
		}
	}

	return nil
}

func verifyOne(pub interface{}, key string, results map[string]v1beta1.TaskRunResult) error {
	signature, ok := results[key+".sig"]
	if !ok {
		return fmt.Errorf("no signature found for %s", key)
	}
	b, err := base64.StdEncoding.DecodeString(signature.Value)
	if err != nil {
		return fmt.Errorf("invalid signature: %s", err)
	}
	h := sha256.Sum256([]byte(results[key].Value))
	// Check val against sig
	switch t := pub.(type) {
	case *ecdsa.PublicKey:
		fmt.Println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~ DOING A FOR REAL VALIDATION  ~~~~~~~")
		if !ecdsa.VerifyASN1(t, h[:], b) {
			return errors.New("invalid signature")
		}
		return nil
	case *rsa.PublicKey:
		return rsa.VerifyPKCS1v15(t, crypto.SHA256, h[:], b)
	case ed25519.PublicKey:
		if !ed25519.Verify(t, []byte(results[key].Value), b) {
			return errors.New("invalid signature")
		}
		return nil
	default:
		return fmt.Errorf("unsupported key type: %s", t)
	}
}

func (i *Tekton) Type() formats.PayloadType {
	return formats.PayloadTypeTekton
}
