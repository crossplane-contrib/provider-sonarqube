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

package common

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// errCrossNamespaceReference is the error returned when a reference
	// targets an object outside the referencing resource's namespace.
	errCrossNamespaceReference = "cross-namespace reference to %s/%s is not allowed: referenced objects must be in namespace %q"
	// errCrossNamespaceSelector is the error returned when a selector
	// targets a namespace other than the referencing resource's namespace.
	errCrossNamespaceSelector = "cross-namespace selector for namespace %q is not allowed: selected objects must be in namespace %q"
)

// namespacePinnedClient is a client whose reads are restricted to a single
// namespace. Writes pass through, so the reference resolver can still patch
// the managed resource it resolves.
type namespacePinnedClient struct {
	client.Client

	namespace string
}

// Get reads the object only if it is in the pinned namespace.
func (c namespacePinnedClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if key.Namespace != c.namespace {
		return errors.Errorf(errCrossNamespaceReference, key.Namespace, key.Name, c.namespace)
	}

	return c.Client.Get(ctx, key, obj, opts...)
}

// List lists objects only if the list is scoped to the pinned namespace.
func (c namespacePinnedClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	listOptions := &client.ListOptions{}
	listOptions.ApplyOptions(opts)

	if listOptions.Namespace != c.namespace {
		return errors.Errorf(errCrossNamespaceSelector, listOptions.Namespace, c.namespace)
	}

	return c.Client.List(ctx, list, opts...)
}

// NewSameNamespaceReferenceResolver returns a reference resolver that only
// resolves references and selectors within the managed resource's own
// namespace. It otherwise behaves like crossplane-runtime's default
// APISimpleReferenceResolver, which honours an explicit namespace on a
// reference or selector and would let a tenant read identifiers from objects
// in other namespaces.
func NewSameNamespaceReferenceResolver(kubeClient client.Client) managed.ReferenceResolver {
	return managed.ReferenceResolverFn(func(ctx context.Context, managedResource resource.Managed) error {
		pinned := namespacePinnedClient{Client: kubeClient, namespace: managedResource.GetNamespace()}

		return managed.NewAPISimpleReferenceResolver(pinned).ResolveReferences(ctx, managedResource)
	})
}
