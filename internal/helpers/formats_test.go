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
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"k8s.io/utils/ptr"
)

func TestIsComparablePtrEqualComparable(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := IsComparablePtrEqualComparable(tc.ptr, tc.val)
			if got != tc.want {
				t.Errorf("IsComparablePtrEqualComparable() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsComparablePtrEqualComparableInt(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := IsComparablePtrEqualComparable(tc.ptr, tc.val)
			if got != tc.want {
				t.Errorf("IsComparablePtrEqualComparable() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsComparablePtrEqualComparablePtr(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := IsComparablePtrEqualComparablePtr(tc.ptr1, tc.ptr2)
			if got != tc.want {
				t.Errorf("IsComparablePtrEqualComparablePtr() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsComparablePtrEqualComparablePtrInt(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := IsComparablePtrEqualComparablePtr(tc.ptr1, tc.ptr2)
			if got != tc.want {
				t.Errorf("IsComparablePtrEqualComparablePtr() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAssignIfNil(t *testing.T) {
	t.Parallel()

	t.Run("NilOuterPointerDoesNothing", func(t *testing.T) {
		t.Parallel()

		// Should not panic
		AssignIfNil[string](nil, "value")
	})

	t.Run("NilInnerPointerAssignsValue", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		original := "original"
		inner := &original
		AssignIfNil(&inner, "new")

		if *inner != "original" {
			t.Errorf("AssignIfNil() changed value to %v, want %v", *inner, "original")
		}
	})

	t.Run("IntNilInnerPointerAssignsValue", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		original := 100
		inner := &original
		AssignIfNil(&inner, 42)

		if *inner != 100 {
			t.Errorf("AssignIfNil() changed value to %v, want %v", *inner, 100)
		}
	})

	t.Run("BoolNilInnerPointerAssignsValue", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		original := false
		inner := &original
		AssignIfNil(&inner, true)

		if *inner != false {
			t.Errorf("AssignIfNil() changed value to %v, want %v", *inner, false)
		}
	})
}

func TestCloseBody(t *testing.T) {
	t.Run("NilResponseDoesNotPanic", func(t *testing.T) {
		CloseBody(nil)
	})

	t.Run("NilBodyDoesNotPanic", func(t *testing.T) {
		resp := &http.Response{Body: nil}
		CloseBody(resp)
	})

	t.Run("ClosesBodySuccessfully", func(t *testing.T) {
		body := io.NopCloser(bytes.NewBufferString("test"))
		resp := &http.Response{Body: body}
		CloseBody(resp)
		// Verify body is closed by trying to read
		_, err := body.Read(make([]byte, 1))
		if err == nil {
			t.Error("Expected error reading from closed body")
		}
	})
}

func TestTimeToMetaTime(t *testing.T) {
	t.Run("NilTimeReturnsNil", func(t *testing.T) {
		result := TimeToMetaTime(nil)
		if result != nil {
			t.Errorf("TimeToMetaTime(nil) = %v, want nil", result)
		}
	})

	t.Run("ValidTimeReturnsMetaTime", func(t *testing.T) {
		now := time.Now()
		result := TimeToMetaTime(&now)
		if result == nil {
			t.Fatal("TimeToMetaTime() returned nil, want non-nil")
		}
		if !result.Time.Equal(now) {
			t.Errorf("TimeToMetaTime() time = %v, want %v", result.Time, now)
		}
	})
}

func TestStringToMetaTime(t *testing.T) {
	t.Run("NilStringReturnsNil", func(t *testing.T) {
		result := StringToMetaTime(nil)
		if result != nil {
			t.Errorf("StringToMetaTime(nil) = %v, want nil", result)
		}
	})

	t.Run("InvalidStringReturnsNil", func(t *testing.T) {
		invalid := "not-a-valid-time"
		result := StringToMetaTime(&invalid)
		if result != nil {
			t.Errorf("StringToMetaTime(invalid) = %v, want nil", result)
		}
	})

	t.Run("ValidRFC3339StringReturnsMetaTime", func(t *testing.T) {
		rfc3339 := "2026-01-20T22:00:00Z"
		result := StringToMetaTime(&rfc3339)
		if result == nil {
			t.Fatal("StringToMetaTime() returned nil, want non-nil")
		}
		expected, _ := time.Parse(time.RFC3339, rfc3339)
		if !result.Time.Equal(expected) {
			t.Errorf("StringToMetaTime() time = %v, want %v", result.Time, expected)
		}
	})
}

func TestMapToSemicolonSeparatedString(t *testing.T) {
	t.Run("NilMapReturnsEmptyString", func(t *testing.T) {
		result := MapToSemicolonSeparatedString(nil)
		if result != "" {
			t.Errorf("MapToSemicolonSeparatedString(nil) = %q, want \"\"", result)
		}
	})

	t.Run("EmptyMapReturnsEmptyString", func(t *testing.T) {
		m := map[string]string{}
		result := MapToSemicolonSeparatedString(&m)
		if result != "" {
			t.Errorf("MapToSemicolonSeparatedString(empty) = %q, want \"\"", result)
		}
	})

	t.Run("SingleEntryMap", func(t *testing.T) {
		m := map[string]string{"key1": "value1"}
		result := MapToSemicolonSeparatedString(&m)
		if result != "key1=value1" {
			t.Errorf("MapToSemicolonSeparatedString() = %q, want \"key1=value1\"", result)
		}
	})

	t.Run("MultipleEntriesMap", func(t *testing.T) {
		m := map[string]string{"key1": "value1", "key2": "value2"}
		result := MapToSemicolonSeparatedString(&m)
		// Map iteration order is not guaranteed, so check both combinations
		if result != "key1=value1;key2=value2" && result != "key2=value2;key1=value1" {
			t.Errorf("MapToSemicolonSeparatedString() = %q, want \"key1=value1;key2=value2\" or \"key2=value2;key1=value1\"", result)
		}
	})
}

func TestAnySliceToStringSlice(t *testing.T) {
	t.Run("NilSliceReturnsEmpty", func(t *testing.T) {
		result := AnySliceToStringSlice(nil)
		if len(result) != 0 {
			t.Errorf("AnySliceToStringSlice(nil) length = %d, want 0", len(result))
		}
	})

	t.Run("EmptySliceReturnsEmpty", func(t *testing.T) {
		slice := []any{}
		result := AnySliceToStringSlice(slice)
		if len(result) != 0 {
			t.Errorf("AnySliceToStringSlice(empty) length = %d, want 0", len(result))
		}
	})

	t.Run("AllStringsReturnsAllElements", func(t *testing.T) {
		slice := []any{"string1", "string2", "string3"}
		result := AnySliceToStringSlice(slice)
		if len(result) != 3 {
			t.Fatalf("AnySliceToStringSlice() length = %d, want 3", len(result))
		}
		if result[0] != "string1" || result[1] != "string2" || result[2] != "string3" {
			t.Errorf("AnySliceToStringSlice() = %v, want [string1 string2 string3]", result)
		}
	})

	t.Run("MixedTypesFiltersNonStrings", func(t *testing.T) {
		slice := []any{"string1", 42, "string2", true, "string3"}
		result := AnySliceToStringSlice(slice)
		if len(result) != 3 {
			t.Fatalf("AnySliceToStringSlice() length = %d, want 3", len(result))
		}
		if result[0] != "string1" || result[1] != "string2" || result[2] != "string3" {
			t.Errorf("AnySliceToStringSlice() = %v, want [string1 string2 string3]", result)
		}
	})

	t.Run("NoStringsReturnsEmpty", func(t *testing.T) {
		slice := []any{42, true, 3.14}
		result := AnySliceToStringSlice(slice)
		if len(result) != 0 {
			t.Errorf("AnySliceToStringSlice(no strings) length = %d, want 0", len(result))
		}
	})
}
