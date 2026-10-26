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

package fake

import (
	"sync/atomic"
	"time"
)

// inFlightCallDuration is how long a tracked call stays in flight.
const inFlightCallDuration = 5 * time.Millisecond

// InFlightTracker records the total and peak number of concurrent calls made
// to a mock client.
type InFlightTracker struct {
	current atomic.Int32
	peak    atomic.Int32
	total   atomic.Int32
}

// Track marks a call as in flight for a short time so concurrent calls overlap.
func (tracker *InFlightTracker) Track() {
	tracker.total.Add(1)

	current := tracker.current.Add(1)
	for {
		peak := tracker.peak.Load()
		if current <= peak || tracker.peak.CompareAndSwap(peak, current) {
			break
		}
	}

	time.Sleep(inFlightCallDuration)
	tracker.current.Add(-1)
}

// Total returns the number of tracked calls.
func (tracker *InFlightTracker) Total() int32 {
	return tracker.total.Load()
}

// Peak returns the highest number of calls that were in flight at once.
func (tracker *InFlightTracker) Peak() int32 {
	return tracker.peak.Load()
}
