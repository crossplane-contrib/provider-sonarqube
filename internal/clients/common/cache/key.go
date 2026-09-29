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

package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"

	"github.com/crossplane/provider-sonarqube/internal/clients/common"
)

// keySeparator separates the Key components in Key.String. It is a byte
// that cannot appear in a URL, an auth type, a hex digest or a namespace.
const keySeparator = "\x00"

// Key identifies a cached entry.
type Key struct {
	// Scope is the identity of the SonarQube connection, see
	// ScopeFromConfig.
	Scope string
	// Namespace is the logical dataset, for example "plugins/installed".
	Namespace string
	// Params is the canonical encoding of the query parameters, see
	// EncodeParams. It is empty for instance-wide lists.
	Params string
}

// String returns a unique, unambiguous representation of the key.
func (k Key) String() string {
	return k.Scope + keySeparator + k.Namespace + keySeparator + k.Params
}

// ScopeFromConfig returns an opaque identifier of the SonarQube connection
// described by config: a SHA-256 digest of its base URL, auth type and
// credentials. Two configs that differ in any of those fields get different
// scopes. The raw secrets are never retained.
func ScopeFromConfig(config common.Config) string {
	var username, password string
	if config.BasicAuth != nil {
		username = config.BasicAuth.Username
		password = config.BasicAuth.Password
	}

	sum := sha256.Sum256([]byte(strings.Join([]string{
		config.BaseURL,
		string(config.AuthType),
		config.Token,
		username,
		password,
	}, keySeparator)))

	return hex.EncodeToString(sum[:])
}

// EncodeParams returns the canonical encoding of the given query
// parameters: keys are sorted, so the same set of parameters always yields
// the same string regardless of map iteration order.
func EncodeParams(params map[string]string) string {
	values := make(url.Values, len(params))
	for name, value := range params {
		values.Set(name, value)
	}

	return values.Encode()
}
