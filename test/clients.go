// +build e2e

/*
Copyright 2020 Tekton Authors LLC
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

/*
Get access to client objects
To initialize client objects you can use the newClients function. It returns a clients struct
that contains initialized clients for accessing:
	- Kubernetes resources
	- Pipelines resources (https://github.com/tektoncd/pipeline)
	- Triggers resources (https://github.com/tektoncd/triggers)
*/

package test

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/tektoncd/pipeline/pkg/names"

	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	knativetest "knative.dev/pkg/test"
)

// clients holds instances of interfaces for making requests to the Pipeline controllers.
type clients struct {
	KubeClient     kubernetes.Interface
	PipelineClient pipelineclientset.Interface
	secret         secret
	// these represent the same registry; internal is accessible from within the cluster
	// external is accessible from outside the cluster via port-forwarding
	internalRegistry string
	externalRegistry string
}

// newClients instantiates and returns several clientsets required for making requests to the
// cluster specified by the combination of clusterName and configPath.
func newClients(t *testing.T, configPath, clusterName string) *clients {
	t.Helper()
	var err error
	c := &clients{}

	cfg, err := knativetest.BuildClientConfig(configPath, clusterName)
	if err != nil {
		t.Fatalf("Failed to create configuration obj from %s for cluster %s: %s", configPath, clusterName, err)
	}

	c.KubeClient, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create kubernetes clientset from config file at %s: %s", configPath, err)
	}

	c.PipelineClient, err = pipelineclientset.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create pipeline clientset from config file at %s: %s", configPath, err)
	}
	return c
}

func setup(ctx context.Context, t *testing.T) (*clients, string, func()) {
	t.Helper()
	namespace := names.SimpleNameGenerator.RestrictLengthWithRandomSuffix("earth")

	c := newClients(t, knativetest.Flags.Kubeconfig, knativetest.Flags.Cluster)
	createNamespace(ctx, t, namespace, c.KubeClient)

	c.secret = setupSecret(ctx, t, c.KubeClient)
	internalRegistry, svc := createRegistry(ctx, t, namespace, c.KubeClient)
	externalRegistry, cancelPortForward := portForward(ctx, t, svc)
	c.internalRegistry, c.externalRegistry = internalRegistry, externalRegistry

	// port forward the registry

	var cleanup = func() {
		t.Logf("Cancelling port forwarding")
		cancelPortForward()
		t.Logf("Deleting namespace %s", namespace)
		if err := c.KubeClient.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil {
			t.Fatalf("Failed to delete namespace %s for tests: %s", namespace, err)
		}
	}
	return c, namespace, cleanup
}

func createNamespace(ctx context.Context, t *testing.T, namespace string, kubeClient kubernetes.Interface) {
	t.Helper()
	t.Logf("Create namespace %s to deploy to", namespace)
	if _, err := kubeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create namespace %s for tests: %s", namespace, err)
	}
}

type secret struct {
	x509Priv *ecdsa.PrivateKey
}

func createRegistry(ctx context.Context, t *testing.T, namespace string, kubeClient kubernetes.Interface) (string, *corev1.Service) {
	t.Helper()
	replicas := int32(1)
	label := map[string]string{"app": "registry"}
	meta := metav1.ObjectMeta{
		Name:      "registry",
		Namespace: namespace,
		Labels:    label,
	}
	deployment := &v1.Deployment{
		ObjectMeta: meta,
		Spec: v1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: label},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: meta,
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "registry",
							Image: "registry:2.7.1@sha256:d5459fcb27aecc752520df4b492b08358a1912fcdfa454f7d2101d4b09991daa",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5000,
								},
							},
						},
					},
				},
			},
		},
	}
	service := &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Selector: label,
			Ports:    []corev1.ServicePort{{Port: int32(5000), Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{IntVal: int32(5000)}}},
		},
	}
	// first, check if the svc already exists
	if svc, err := kubeClient.CoreV1().Services(namespace).Get(ctx, service.Name, metav1.GetOptions{}); err == nil {
		return fmt.Sprintf("%s.%s.svc.cluster.local:5000", svc.Name, svc.Namespace), svc
	}
	t.Logf("Creating insecure registry to deploy in ns %s", namespace)
	if _, err := kubeClient.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create deployment registry for tests: %s", err)
	}
	service, err := kubeClient.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create service for tests: %s", err)
	}
	return fmt.Sprintf("%s.%s.svc.cluster.local:5000", service.Name, service.Namespace), service
}

func portForward(ctx context.Context, t *testing.T, svc *corev1.Service) (string, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, "kubectl", "port-forward", fmt.Sprintf("svc/%s", svc.Name), "5000:5000", "-n", svc.Namespace)
	go func() {
		t.Log("starting port forwarding...")
		if err := cmd.Run(); err != nil {
			t.Logf("port forwarding died: %v\n", err)
		}
	}()
	return "localhost:5000", cancel
}

func setupSecret(ctx context.Context, t *testing.T, c kubernetes.Interface) secret {
	// Only overwrite the secret data if it isn't set.
	namespace := "tekton-chains"
	s := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "signing-secrets",
			Namespace: namespace,
		},
		StringData: map[string]string{},
	}
	// pgp
	paths := []string{"pgp.private-key", "pgp.passphrase", "pgp.public-key"}
	for _, p := range paths {
		b, err := ioutil.ReadFile(filepath.Join("./testdata", p))
		if err != nil {
			t.Error(err)
		}
		s.StringData[p] = string(b)
	}

	// x509
	_, priv := ecdsaKeyPair(t)

	s.StringData["x509.pem"] = toPem(t, priv)

	if _, err := c.CoreV1().Secrets(namespace).Update(ctx, &s, metav1.UpdateOptions{}); err != nil {
		t.Error(err)
	}
	time.Sleep(60 * time.Second)
	return secret{
		x509Priv: priv,
	}
}

func ecdsaKeyPair(t *testing.T) (crypto.PublicKey, *ecdsa.PrivateKey) {
	kp, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	return kp.PublicKey, kp
}

func toPem(t *testing.T, priv *ecdsa.PrivateKey) string {
	b, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	p := pem.EncodeToMemory(&pem.Block{
		Bytes: b,
		Type:  "PRIVATE KEY",
	})
	return string(p)
}
