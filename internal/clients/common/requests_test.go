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
	"net/http"
	"testing"
)

// TestIsResponseNotFound tests detecting not-found HTTP responses.
func TestIsResponseNotFound(t *testing.T) {
	t.Parallel()

	type args struct {
		resp *http.Response
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Response is nil",
			args: args{
				resp: nil,
			},
			want: false,
		},
		{
			name: "Status code is not 404",
			args: args{
				resp: &http.Response{StatusCode: http.StatusInternalServerError},
			},
			want: false,
		},
		{
			name: "Status code is 404",
			args: args{
				resp: &http.Response{StatusCode: http.StatusNotFound},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsResponseNotFound(tt.args.resp); got != tt.want {
				t.Errorf("IsResponseNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}
