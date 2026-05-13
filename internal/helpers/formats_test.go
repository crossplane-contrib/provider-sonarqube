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
)

const (
	// original is the original string value.
	original = "original"
	// testStr1 is the first test string value.
	testStr1 = "testStr1"
	// string3 is the third test string value.
	string3 = "string3"
	// testNewStr is the new test string value.
	testNewStr = "testNewStr"
)

// TestIsComparablePtrEqualComparable tests pointer equality with values.
func TestIsComparablePtrEqualComparable(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		pointer *string
		val     string
		want    bool
	}{
		"NilPointerReturnsTrue": {
			pointer: nil,
			val:     "any",
			want:    true,
		},
		"MatchingValueReturnsTrue": {
			pointer: new("hello"),
			val:     "hello",
			want:    true,
		},
		"DifferentValueReturnsFalse": {
			pointer: new("hello"),
			val:     "world",
			want:    false,
		},
		"EmptyStringMatch": {
			pointer: new(""),
			val:     "",
			want:    true,
		},
		"EmptyStringNoMatch": {
			pointer: new(""),
			val:     "nonempty",
			want:    false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsComparablePtrEqualComparable(tc.pointer, tc.val)
			if got != tc.want {
				t.Errorf("IsComparablePtrEqualComparable() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsComparablePtrEqualComparableInt tests pointer equality with integers.
func TestIsComparablePtrEqualComparableInt(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		pointer *int
		val     int
		want    bool
	}{
		"NilPointerReturnsTrue": {
			pointer: nil,
			val:     42,
			want:    true,
		},
		"MatchingValueReturnsTrue": {
			pointer: new(42),
			val:     42,
			want:    true,
		},
		"DifferentValueReturnsFalse": {
			pointer: new(42),
			val:     24,
			want:    false,
		},
		"ZeroValueMatch": {
			pointer: new(0),
			val:     0,
			want:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsComparablePtrEqualComparable(tc.pointer, tc.val)
			if got != tc.want {
				t.Errorf("IsComparablePtrEqualComparable() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsComparablePtrEqualComparablePtr tests pointer equality with pointers.
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
			ptr2: new("hello"),
			want: false,
		},
		"SecondNilReturnsFalse": {
			ptr1: new("hello"),
			ptr2: nil,
			want: false,
		},
		"MatchingValuesReturnsTrue": {
			ptr1: new("hello"),
			ptr2: new("hello"),
			want: true,
		},
		"DifferentValuesReturnsFalse": {
			ptr1: new("hello"),
			ptr2: new("world"),
			want: false,
		},
		"EmptyStringMatch": {
			ptr1: new(""),
			ptr2: new(""),
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

// TestIsComparablePtrEqualComparablePtrInt tests integer pointer equality.
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
			ptr2: new(42),
			want: false,
		},
		"SecondNilReturnsFalse": {
			ptr1: new(42),
			ptr2: nil,
			want: false,
		},
		"MatchingValuesReturnsTrue": {
			ptr1: new(42),
			ptr2: new(42),
			want: true,
		},
		"DifferentValuesReturnsFalse": {
			ptr1: new(42),
			ptr2: new(24),
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

// TestAssignIfNil tests assigning values to nil pointers.
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

		original := original
		inner := &original
		AssignIfNil(&inner, testNewStr)

		if *inner != original {
			t.Errorf("AssignIfNil() changed value to %v, want %v", *inner, original)
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

// TestCloseBody tests closing HTTP response bodies.
func TestCloseBody(t *testing.T) {
	t.Parallel()

	t.Run("NilResponseDoesNotPanic", func(t *testing.T) {
		t.Parallel()

		CloseBody(nil)
	})

	t.Run("NilBodyDoesNotPanic", func(t *testing.T) {
		t.Parallel()

		resp := &http.Response{Body: nil}
		CloseBody(resp)
	})

	t.Run("ClosesBodySuccessfully", func(t *testing.T) {
		t.Parallel()

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

// TestTimeToMetaTime tests converting [time.Time] to MetaTime.
func TestTimeToMetaTime(t *testing.T) {
	t.Parallel()

	t.Run("NilTimeReturnsNil", func(t *testing.T) {
		t.Parallel()

		result := TimeToMetaTime(nil)
		if result != nil {
			t.Errorf("TimeToMetaTime(nil) = %v, want nil", result)
		}
	})

	t.Run("ValidTimeReturnsMetaTime", func(t *testing.T) {
		t.Parallel()

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

// mustParseTime parses a time string with the given layout, panicking on error.
func mustParseTime(layout, value string) time.Time {
	parsedTime, err := time.Parse(layout, value)
	if err != nil {
		panic(err)
	}

	return parsedTime.UTC()
}

// TestStringToMetaTime tests multi-format datetime string parsing.
func TestStringToMetaTime(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		input   *string
		wantNil bool
		wantUTC time.Time
	}{
		"NilPointerReturnsNil": {
			input:   nil,
			wantNil: true,
		},
		"EmptyStringReturnsNil": {
			input:   new(""),
			wantNil: true,
		},
		"InvalidStringReturnsNil": {
			input:   new("not-a-valid-time"),
			wantNil: true,
		},
		"PartialDatetimeReturnsNil": {
			input:   new("2026-01-20T22"),
			wantNil: true,
		},
		"DateTimeWithoutTZReturnsNil": {
			input:   new("2026-01-20T22:00:00"),
			wantNil: true,
		},
		"RFC3339UTCSuffix": {
			input:   new("2026-01-20T22:00:00Z"),
			wantUTC: mustParseTime(time.RFC3339, "2026-01-20T22:00:00Z"),
		},
		"RFC3339PositiveColonOffset": {
			input:   new("2026-06-01T08:00:00+05:30"),
			wantUTC: mustParseTime(time.RFC3339, "2026-06-01T08:00:00+05:30"),
		},
		"RFC3339NegativeColonOffset": {
			input:   new("2026-03-15T10:00:00-07:00"),
			wantUTC: mustParseTime(time.RFC3339, "2026-03-15T10:00:00-07:00"),
		},
		"SonarQubeFormatUTCOffset": {
			input:   new("2026-01-30T22:02:16+0000"),
			wantUTC: mustParseTime("2006-01-02T15:04:05-0700", "2026-01-30T22:02:16+0000"),
		},
		"SonarQubeFormatPositiveOffset": {
			input:   new("2026-05-14T12:30:00+0530"),
			wantUTC: mustParseTime("2006-01-02T15:04:05-0700", "2026-05-14T12:30:00+0530"),
		},
		"SonarQubeFormatNegativeOffset": {
			input:   new("2026-05-14T15:30:45-0500"),
			wantUTC: mustParseTime("2006-01-02T15:04:05-0700", "2026-05-14T15:30:45-0500"),
		},
		"DateOnly": {
			input:   new("2026-04-15"),
			wantUTC: mustParseTime(time.DateOnly, "2026-04-15"),
		},
		"DateOnlyFirstDayOfYear": {
			input:   new("2026-01-01"),
			wantUTC: mustParseTime(time.DateOnly, "2026-01-01"),
		},
		"DateOnlyLastDayOfYear": {
			input:   new("2026-12-31"),
			wantUTC: mustParseTime(time.DateOnly, "2026-12-31"),
		},
		"ResultIsAlwaysUTC": {
			input:   new("2026-01-30T22:02:16+0530"),
			wantUTC: mustParseTime("2006-01-02T15:04:05-0700", "2026-01-30T22:02:16+0530"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := StringToMetaTime(tc.input)

			if tc.wantNil {
				if result != nil {
					t.Errorf("StringToMetaTime() = %v, want nil", result)
				}

				return
			}

			if result == nil {
				t.Fatal("StringToMetaTime() = nil, want non-nil")
			}

			if !result.Time.Equal(tc.wantUTC) {
				t.Errorf("StringToMetaTime() = %v, want %v", result.Time, tc.wantUTC)
			}
		})
	}
}

// TestAnySliceToStringSlice tests converting any slice to string slice.
func TestAnySliceToStringSlice(t *testing.T) {
	t.Parallel()

	t.Run("NilSliceReturnsEmpty", func(t *testing.T) {
		t.Parallel()

		result := AnySliceToStringSlice(nil)
		if len(result) != 0 {
			t.Errorf("AnySliceToStringSlice(nil) length = %d, want 0", len(result))
		}
	})

	t.Run("EmptySliceReturnsEmpty", func(t *testing.T) {
		t.Parallel()

		slice := []any{}

		result := AnySliceToStringSlice(slice)
		if len(result) != 0 {
			t.Errorf("AnySliceToStringSlice(empty) length = %d, want 0", len(result))
		}
	})

	t.Run("AllStringsReturnsAllElements", func(t *testing.T) {
		t.Parallel()

		slice := []any{testStr1, testStr1, string3}

		result := AnySliceToStringSlice(slice)
		if len(result) != 3 {
			t.Fatalf("AnySliceToStringSlice() length = %d, want 3", len(result))
		}

		if result[0] != testStr1 || result[1] != testStr1 || result[2] != string3 {
			t.Errorf("AnySliceToStringSlice() = %v, want [testStr1 testStr1 string3]", result)
		}
	})

	t.Run("MixedTypesFiltersNonStrings", func(t *testing.T) {
		t.Parallel()

		slice := []any{testStr1, 42, testStr1, true, string3}

		result := AnySliceToStringSlice(slice)
		if len(result) != 3 {
			t.Fatalf("AnySliceToStringSlice() length = %d, want 3", len(result))
		}

		if result[0] != testStr1 || result[1] != testStr1 || result[2] != string3 {
			t.Errorf("AnySliceToStringSlice() = %v, want [testStr1 testStr1 string3]", result)
		}
	})

	t.Run("NoStringsReturnsEmpty", func(t *testing.T) {
		t.Parallel()

		slice := []any{42, true, 3.14}

		result := AnySliceToStringSlice(slice)
		if len(result) != 0 {
			t.Errorf("AnySliceToStringSlice(no strings) length = %d, want 0", len(result))
		}
	})
}

// TestIsComparableSlicePtrEqualComparableSlice tests slice pointer equality.
func TestIsComparableSlicePtrEqualComparableSlice(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		pointer *[]string
		val     []string
		want    bool
	}{
		"NilPointerReturnsTrue": {
			pointer: nil,
			val:     []string{"a", "b"},
			want:    true,
		},
		"MatchingSlicesReturnsTrue": {
			pointer: new([]string{"a", "b", "c"}),
			val:     []string{"a", "b", "c"},
			want:    true,
		},
		"DifferentSlicesReturnsFalse": {
			pointer: new([]string{"a", "b"}),
			val:     []string{"a", "c"},
			want:    false,
		},
		"EmptySliceMatch": {
			pointer: new([]string{}),
			val:     []string{},
			want:    true,
		},
		"NilSliceAndEmptySliceMatch": {
			pointer: new([]string(nil)),
			val:     []string{},
			want:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsComparableSlicePtrEqualComparableSlice(tc.pointer, tc.val)
			if got != tc.want {
				t.Errorf("IsComparableSlicePtrEqualComparableSlice() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIsComparableMapPtrEqualComparableMap tests map pointer equality.
func TestIsComparableMapPtrEqualComparableMap(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		pointer *map[string]int
		val     map[string]int
		want    bool
	}{
		"NilPointerReturnsTrue": {
			pointer: nil,
			val:     map[string]int{"key": 42},
			want:    true,
		},
		"MatchingMapsReturnsTrue": {
			pointer: new(map[string]int{"a": 1, "b": 2}),
			val:     map[string]int{"a": 1, "b": 2},
			want:    true,
		},
		"DifferentMapsReturnsFalse": {
			pointer: new(map[string]int{"a": 1}),
			val:     map[string]int{"a": 2},
			want:    false,
		},
		"EmptyMapMatch": {
			pointer: new(map[string]int{}),
			val:     map[string]int{},
			want:    true,
		},
		"NilMapAndEmptyMapMatch": {
			pointer: new(map[string]int(nil)),
			val:     map[string]int{},
			want:    true,
		},
		"DifferentKeysReturnsFalse": {
			pointer: new(map[string]int{"a": 1}),
			val:     map[string]int{"b": 1},
			want:    false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := IsComparableMapPtrEqualComparableMap(tc.pointer, tc.val)
			if got != tc.want {
				t.Errorf("IsComparableMapPtrEqualComparableMap() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestAssignIfNonNil tests assigning values from non-nil pointers.
func TestAssignIfNonNil(t *testing.T) {
	t.Parallel()

	t.Run("NilPointerDoesNothing", func(t *testing.T) {
		t.Parallel()

		// Should not panic
		val := "value"
		AssignIfNonNil[string](nil, &val)
	})

	t.Run("NilRefDoesNothing", func(t *testing.T) {
		t.Parallel()

		original := original
		pointer := &original
		AssignIfNonNil(pointer, nil)

		if *pointer != original {
			t.Errorf("AssignIfNonNil() changed value to %v, want %v", *pointer, original)
		}
	})

	t.Run("NonNilRefAssignsValue", func(t *testing.T) {
		t.Parallel()

		original := original
		pointer := &original
		testNewStrVal := testNewStr
		AssignIfNonNil(pointer, &testNewStrVal)

		if *pointer != testNewStr {
			t.Errorf("AssignIfNonNil() assigned %v, want %v", *pointer, testNewStr)
		}
	})

	t.Run("IntNilRefDoesNothing", func(t *testing.T) {
		t.Parallel()

		original := 42
		pointer := &original
		AssignIfNonNil(pointer, nil)

		if *pointer != 42 {
			t.Errorf("AssignIfNonNil() changed value to %v, want %v", *pointer, 42)
		}
	})

	t.Run("IntNonNilRefAssignsValue", func(t *testing.T) {
		t.Parallel()

		original := 42
		pointer := &original
		testNewStrVal := 100
		AssignIfNonNil(pointer, &testNewStrVal)

		if *pointer != 100 {
			t.Errorf("AssignIfNonNil() assigned %v, want %v", *pointer, 100)
		}
	})

	t.Run("BoolNonNilRefAssignsValue", func(t *testing.T) {
		t.Parallel()

		original := false
		pointer := &original
		testNewStrVal := true
		AssignIfNonNil(pointer, &testNewStrVal)

		if *pointer != true {
			t.Errorf("AssignIfNonNil() assigned %v, want %v", *pointer, true)
		}
	})
}

// TestAreStringSlicesEqual tests equality of string slices regardless of order.
func TestAreStringSlicesEqual(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		a    []string
		b    []string
		want bool
	}{
		"BothEmpty": {
			a:    []string{},
			b:    []string{},
			want: true,
		},
		"SameElementsDifferentOrder": {
			a:    []string{"a", "b", "c"},
			b:    []string{"c", "a", "b"},
			want: true,
		},
		"DifferentLengths": {
			a:    []string{"a", "b"},
			b:    []string{"a"},
			want: false,
		},
		"DifferentCounts": {
			a:    []string{"a", "a", "b"},
			b:    []string{"a", "b", "b"},
			want: false,
		},
		"NegativeCountBranch": {
			a:    []string{"a", "b"},
			b:    []string{"a", "a"},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := AreStringSlicesEqual(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("AreStringSlicesEqual() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestAreStringSlicesEqualDeDuped tests equality of deduplicated string slices.
func TestAreStringSlicesEqualDeDuped(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		a    []string
		b    []string
		want bool
	}{
		"BothEmpty": {
			a:    []string{},
			b:    []string{},
			want: true,
		},
		"SameElementsWithDuplicates": {
			a:    []string{"a", "a", "b"},
			b:    []string{"b", "a"},
			want: true,
		},
		"DifferentUniqueLengths": {
			a:    []string{"a", "b"},
			b:    []string{"a", "b", "c"},
			want: false,
		},
		"MissingElementInSecondSet": {
			a:    []string{"a", "b"},
			b:    []string{"a", "c"},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := AreStringSlicesEqualDeDuped(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("AreStringSlicesEqualDeDuped() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestNewStringSetFromSlice tests creating a set from a string slice.
func TestNewStringSetFromSlice(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in   []string
		want map[string]struct{}
	}{
		"EmptyInput": {
			in:   []string{},
			want: map[string]struct{}{},
		},
		"UniqueValues": {
			in:   []string{"a", "b", "c"},
			want: map[string]struct{}{"a": {}, "b": {}, "c": {}},
		},
		"WithDuplicates": {
			in:   []string{"a", "a", "b"},
			want: map[string]struct{}{"a": {}, "b": {}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := NewStringSetFromSlice(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("NewStringSetFromSlice() len = %d, want %d", len(got), len(tc.want))
			}

			for k := range tc.want {
				if _, ok := got[k]; !ok {
					t.Fatalf("NewStringSetFromSlice() missing key %q", k)
				}
			}
		})
	}
}
