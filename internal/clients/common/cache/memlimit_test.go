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
	"testing"
	"testing/fstest"
)

// cgroupFile returns a MapFS holding a single cgroup file.
func cgroupFile(path, content string) fstest.MapFS {
	return fstest.MapFS{path: &fstest.MapFile{Data: []byte(content)}}
}

// TestContainerMemoryLimit tests reading the memory limit from cgroup v2
// and v1 filesystems.
func TestContainerMemoryLimit(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cgroupFS    fstest.MapFS
		wantLimit   int64
		wantLimited bool
		wantErr     bool
	}{
		"V2Limit": {
			cgroupFS:  cgroupFile(cgroupV2MemoryMax, "536870912\n"),
			wantLimit: 536870912, wantLimited: true,
		},
		"V2Unlimited": {
			cgroupFS: cgroupFile(cgroupV2MemoryMax, "max\n"),
		},
		"V1Limit": {
			cgroupFS:  cgroupFile(cgroupV1MemoryLimit, "268435456\n"),
			wantLimit: 268435456, wantLimited: true,
		},
		"V1Unlimited": {
			cgroupFS: cgroupFile(cgroupV1MemoryLimit, "9223372036854771712\n"),
		},
		"V2TakesPrecedence": {
			cgroupFS: fstest.MapFS{
				cgroupV2MemoryMax:   &fstest.MapFile{Data: []byte("1000")},
				cgroupV1MemoryLimit: &fstest.MapFile{Data: []byte("2000")},
			},
			wantLimit: 1000, wantLimited: true,
		},
		"NoCgroup": {
			cgroupFS: fstest.MapFS{},
		},
		"Garbage": {
			cgroupFS: cgroupFile(cgroupV2MemoryMax, "lots"),
			wantErr:  true,
		},
		"Unreadable": {
			// A directory where the file is expected cannot be read.
			cgroupFS: fstest.MapFS{cgroupV2MemoryMax + "/child": &fstest.MapFile{}},
			wantErr:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			limit, limited, err := containerMemoryLimit(tc.cgroupFS)
			if (err != nil) != tc.wantErr {
				t.Fatalf("containerMemoryLimit() error = %v, wantErr %v", err, tc.wantErr)
			}

			if limit != tc.wantLimit || limited != tc.wantLimited {
				t.Errorf("containerMemoryLimit() = %d, %v, want %d, %v", limit, limited, tc.wantLimit, tc.wantLimited)
			}
		})
	}
}

// TestResolveMaxBytes tests choosing the size budget of the cache.
func TestResolveMaxBytes(t *testing.T) {
	t.Parallel()

	const limit = int64(512 << 20)

	tests := map[string]struct {
		explicit int64
		fraction float64
		limit    int64
		limited  bool
		want     int64
		wantErr  bool
	}{
		"FractionOfLimit": {
			fraction: 0.1, limit: limit, limited: true, want: limit / 10,
		},
		"WholeLimitFraction": {
			fraction: 1, limit: limit, limited: true, want: limit,
		},
		"TinyLimitStillPositive": {
			fraction: 0.1, limit: 5, limited: true, want: 1,
		},
		"NoLimitUsesDefault": {
			fraction: 0.1, want: DefaultMaxBytes,
		},
		"ExplicitBelowLimit": {
			explicit: 64 << 20, fraction: 0.1, limit: limit, limited: true, want: 64 << 20,
		},
		"ExplicitWithoutLimit": {
			explicit: 1 << 30, fraction: 0.1, want: 1 << 30,
		},
		"ExplicitAtLimit": {
			explicit: limit, fraction: 0.1, limit: limit, limited: true, wantErr: true,
		},
		"ExplicitAboveLimit": {
			explicit: 2 * limit, fraction: 0.1, limit: limit, limited: true, wantErr: true,
		},
		"NegativeExplicit": {
			explicit: -1, fraction: 0.1, wantErr: true,
		},
		"ZeroFraction": {
			fraction: 0, limit: limit, limited: true, wantErr: true,
		},
		"FractionAboveOne": {
			fraction: 1.5, limit: limit, limited: true, wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ResolveMaxBytes(tc.explicit, tc.fraction, tc.limit, tc.limited)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ResolveMaxBytes() error = %v, wantErr %v", err, tc.wantErr)
			}

			if got != tc.want {
				t.Errorf("ResolveMaxBytes() = %d, want %d", got, tc.want)
			}
		})
	}
}
