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
	"time"
)

// TestOptionsValidate tests the validation of the cache options.
func TestOptionsValidate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		opts    Options
		wantErr bool
	}{
		"DisabledIgnoresValues": {opts: Options{Enabled: false, TTL: -1, MaxEntries: -1}},
		"Defaults":              {opts: Options{Enabled: true, TTL: DefaultTTL, MaxEntries: DefaultMaxEntries}},
		"ZeroTTL":               {opts: Options{Enabled: true, TTL: 0, MaxEntries: 1}, wantErr: true},
		"NegativeTTL":           {opts: Options{Enabled: true, TTL: -time.Second, MaxEntries: 1}, wantErr: true},
		"TTLAtGracePeriod":      {opts: Options{Enabled: true, TTL: MaxTTL, MaxEntries: 1}, wantErr: true},
		"TTLJustBelowMax":       {opts: Options{Enabled: true, TTL: MaxTTL - time.Millisecond, MaxEntries: 1}},
		"ZeroMaxEntries":        {opts: Options{Enabled: true, TTL: time.Second, MaxEntries: 0}, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tc.opts.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// TestConfigure mutates the process-wide default store, so it must not run
// in parallel with other tests relying on Default.
//
//nolint:paralleltest // mutates package-level state
func TestConfigure(t *testing.T) {
	t.Cleanup(func() {
		err := Configure(Options{})
		if err != nil {
			t.Errorf("resetting Configure() error = %v", err)
		}
	})

	if IsEnabled(Default()) {
		t.Fatal("Default() is enabled before Configure()")
	}

	if Configure(Options{Enabled: true, TTL: MaxTTL}) == nil {
		t.Fatal("Configure() with invalid options succeeded")
	}

	if IsEnabled(Default()) {
		t.Fatal("Default() is enabled after a failed Configure()")
	}

	err := Configure(Options{Enabled: true, TTL: time.Second, MaxEntries: 1})
	if err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	if !IsEnabled(Default()) {
		t.Fatal("Default() is not enabled after Configure()")
	}

	// Reconfiguring replaces the store.
	previous := Default()

	err = Configure(Options{Enabled: true, TTL: time.Second, MaxEntries: 1})
	if err != nil {
		t.Fatalf("second Configure() error = %v", err)
	}

	if Default() == previous {
		t.Error("Configure() did not replace the default store")
	}

	err = Configure(Options{})
	if err != nil {
		t.Fatalf("disabling Configure() error = %v", err)
	}

	if IsEnabled(Default()) {
		t.Error("Default() is enabled after disabling Configure()")
	}
}
