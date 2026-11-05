/*
Copyright 2026 The Crossplane Authors.

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

package main

import (
	"net/http"
	"testing"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// crdCacheOptions returns the cache options registered for
// CustomResourceDefinitions, failing the test when there are none.
func crdCacheOptions(t *testing.T, options *cache.Options) cache.ByObject {
	t.Helper()

	for object, byObject := range options.ByObject {
		if _, ok := object.(*apiextensionsv1.CustomResourceDefinition); ok {
			return byObject
		}
	}

	t.Fatal("cacheOptions() has no entry for CustomResourceDefinition")

	return cache.ByObject{}
}

// TestManagerAcceptsCacheOptions checks that a manager can be built from the
// scheme and cache options of main: ctrl.NewManager fails when a type of
// cache.Options.ByObject is missing from the scheme.
func TestManagerAcceptsCacheOptions(t *testing.T) {
	t.Parallel()

	scheme, err := newScheme()
	if err != nil {
		t.Fatalf("newScheme() unexpected error: %v", err)
	}

	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{apiextensionsv1.SchemeGroupVersion})
	mapper.Add(apiextensionsv1.SchemeGroupVersion.WithKind("CustomResourceDefinition"), meta.RESTScopeRoot)

	syncInterval := time.Hour

	_, err = ctrl.NewManager(&rest.Config{Host: "http://127.0.0.1:1"}, ctrl.Options{
		Scheme:         scheme,
		Cache:          cacheOptions(&syncInterval),
		Metrics:        metricsserver.Options{BindAddress: "0"},
		MapperProvider: func(*rest.Config, *http.Client) (meta.RESTMapper, error) { return mapper, nil },
	})
	if err != nil {
		t.Fatalf("NewManager() unexpected error: %v", err)
	}
}

// TestCacheOptionsStripCRDSchema checks that the OpenAPI schema of a CRD does
// not reach the informer cache, while what the CRD gate reads is left intact.
func TestCacheOptionsStripCRDSchema(t *testing.T) {
	t.Parallel()

	syncInterval := time.Hour
	options := cacheOptions(&syncInterval)

	if options.SyncPeriod == nil || *options.SyncPeriod != syncInterval {
		t.Fatalf("cacheOptions().SyncPeriod = %v, want %v", options.SyncPeriod, syncInterval)
	}

	transform := crdCacheOptions(t, &options).Transform
	if transform == nil {
		t.Fatal("cacheOptions() has no Transform for CustomResourceDefinition")
	}

	crd := &apiextensionsv1.CustomResourceDefinition{
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "instance.sonarqube.crossplane.io",
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name:   "v1alpha1",
				Served: true,
				Schema: &apiextensionsv1.CustomResourceValidation{},
			}},
		},
	}

	transformed, err := transform(crd)
	if err != nil {
		t.Fatalf("Transform() unexpected error: %v", err)
	}

	got, ok := transformed.(*apiextensionsv1.CustomResourceDefinition)
	if !ok {
		t.Fatalf("Transform() returned %T, want a CustomResourceDefinition", transformed)
	}

	version := got.Spec.Versions[0]
	if version.Schema != nil {
		t.Error("Transform() kept the CRD schema")
	}

	if version.Name != "v1alpha1" || !version.Served || got.Spec.Group != crd.Spec.Group {
		t.Errorf("Transform() altered the fields read by the CRD gate: %+v", got.Spec)
	}
}
