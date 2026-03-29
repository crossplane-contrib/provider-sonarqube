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

import "net/http"

// IsResponseNotFound checks if the response from a SonarQube API call indicates that the resource was not found (HTTP 404).
func IsResponseNotFound(resp *http.Response) bool {
	return resp != nil && resp.StatusCode == http.StatusNotFound
}
