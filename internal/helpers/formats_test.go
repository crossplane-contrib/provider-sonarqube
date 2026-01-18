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

package helpers

import (
	"testing"

	"k8s.io/utils/ptr"
)

func TestIsComparablePtrEqualComparable(t *testing.T) {
	tests := map[string]struct {
		ptr  *string
		val  string
		want bool
	}{
		"NilPointerReturnsTrue": {
			ptr:  nil,
			val:  "any",
			want: true,
		},
		"MatchingValueReturnsTrue": {
			ptr:  ptr.To("hello"),
			val:  "hello",
			want: true,
		},
		"DifferentValueReturnsFalse": {
			ptr:  ptr.To("hello"),
			val:  "world",
			want: false,
		},
		"EmptyStringMatch": {
			ptr:  ptr.To(""),
			val:  "",
			want: true,
		},
		"EmptyStringNoMatch": {
			ptr:  ptr.To(""),
			val:  "nonempty",
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := IsComparablePtrEqualComparable(tc.ptr, tc.val)
			if got != tc.want {
				t.Errorf("IsComparablePtrEqualComparable() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsComparablePtrEqualComparableInt(t *testing.T) {
	tests := map[string]struct {
		ptr  *int
		val  int
		want bool
	}{
		"NilPointerReturnsTrue": {
			ptr:  nil,
			val:  42,
			want: true,
		},
		"MatchingValueReturnsTrue": {
			ptr:  ptr.To(42),
			val:  42,
			want: true,
		},
		"DifferentValueReturnsFalse": {
			ptr:  ptr.To(42),
			val:  24,
			want: false,
		},
		"ZeroValueMatch": {
			ptr:  ptr.To(0),
			val:  0,
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := IsComparablePtrEqualComparable(tc.ptr, tc.val)
			if got != tc.want {
				t.Errorf("IsComparablePtrEqualComparable() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsComparablePtrEqualComparablePtr(t *testing.T) {
	tests := map[string]struct {
		ptr1 *string
		ptr2 *string
		want bool
	}{
		"BothNilReturnsTrue": {
			ptr1: nil,
			ptr2: nil,
			want: true,
		},
		"FirstNilReturnsFalse": {
			ptr1: nil,
			ptr2: ptr.To("hello"),
			want: false,
		},
		"SecondNilReturnsFalse": {
			ptr1: ptr.To("hello"),
			ptr2: nil,
			want: false,
		},
		"MatchingValuesReturnsTrue": {
			ptr1: ptr.To("hello"),
			ptr2: ptr.To("hello"),
			want: true,
		},
		"DifferentValuesReturnsFalse": {
			ptr1: ptr.To("hello"),
			ptr2: ptr.To("world"),
			want: false,
		},
		"EmptyStringMatch": {
			ptr1: ptr.To(""),
			ptr2: ptr.To(""),
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := IsComparablePtrEqualComparablePtr(tc.ptr1, tc.ptr2)
			if got != tc.want {
				t.Errorf("IsComparablePtrEqualComparablePtr() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsComparablePtrEqualComparablePtrInt(t *testing.T) {
	tests := map[string]struct {
		ptr1 *int
		ptr2 *int
		want bool
	}{
		"BothNilReturnsTrue": {
			ptr1: nil,
			ptr2: nil,
			want: true,
		},
		"FirstNilReturnsFalse": {
			ptr1: nil,
			ptr2: ptr.To(42),
			want: false,
		},
		"SecondNilReturnsFalse": {
			ptr1: ptr.To(42),
			ptr2: nil,
			want: false,
		},
		"MatchingValuesReturnsTrue": {
			ptr1: ptr.To(42),
			ptr2: ptr.To(42),
			want: true,
		},
		"DifferentValuesReturnsFalse": {
			ptr1: ptr.To(42),
			ptr2: ptr.To(24),
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := IsComparablePtrEqualComparablePtr(tc.ptr1, tc.ptr2)
			if got != tc.want {
				t.Errorf("IsComparablePtrEqualComparablePtr() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAssignIfNil(t *testing.T) {
	t.Run("NilOuterPointerDoesNothing", func(t *testing.T) {
		// Should not panic
		AssignIfNil[string](nil, "value")
	})

	t.Run("NilInnerPointerAssignsValue", func(t *testing.T) {
		var inner *string
		AssignIfNil(&inner, "hello")
		if inner == nil {
			t.Error("AssignIfNil() did not assign value to nil pointer")
		}
		if *inner != "hello" {
			t.Errorf("AssignIfNil() assigned %v, want %v", *inner, "hello")
		}
	})

	t.Run("NonNilInnerPointerKeepsOriginalValue", func(t *testing.T) {
		original := "original"
		inner := &original
		AssignIfNil(&inner, "new")
		if *inner != "original" {
			t.Errorf("AssignIfNil() changed value to %v, want %v", *inner, "original")
		}
	})

	t.Run("IntNilInnerPointerAssignsValue", func(t *testing.T) {
		var inner *int
		AssignIfNil(&inner, 42)
		if inner == nil {
			t.Error("AssignIfNil() did not assign value to nil pointer")
		}
		if *inner != 42 {
			t.Errorf("AssignIfNil() assigned %v, want %v", *inner, 42)
		}
	})

	t.Run("IntNonNilInnerPointerKeepsOriginalValue", func(t *testing.T) {
		original := 100
		inner := &original
		AssignIfNil(&inner, 42)
		if *inner != 100 {
			t.Errorf("AssignIfNil() changed value to %v, want %v", *inner, 100)
		}
	})

	t.Run("BoolNilInnerPointerAssignsValue", func(t *testing.T) {
		var inner *bool
		AssignIfNil(&inner, true)
		if inner == nil {
			t.Error("AssignIfNil() did not assign value to nil pointer")
		}
		if *inner != true {
			t.Errorf("AssignIfNil() assigned %v, want %v", *inner, true)
		}
	})

	t.Run("BoolNonNilInnerPointerKeepsOriginalValue", func(t *testing.T) {
		original := false
		inner := &original
		AssignIfNil(&inner, true)
		if *inner != false {
			t.Errorf("AssignIfNil() changed value to %v, want %v", *inner, false)
		}
	})
}
