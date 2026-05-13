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

// Package helpers provides utility functions for formatting and comparison.
package helpers

import (
	"io"
	"net/http"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CloseBody closes the body of an [http.Response] safely.
// If the response or body is nil, it does nothing.
// The error return value of Close is intentionally ignored.
func CloseBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

// IsComparablePtrEqualComparable compares a pointer to a comparable type with
// a comparable type.
// If the pointer is nil, it returns true.
// Otherwise, it dereferences the pointer and compares the value with the
// provided comparable type.
func IsComparablePtrEqualComparable[T comparable](ptr *T, val T) bool {
	// if ptr is nil, consider it equal (no difference between nil and any value)
	if ptr == nil {
		return true
	}
	// use cmp library to compare dereferenced ptr with val
	return cmp.Equal(*ptr, val)
}

// IsComparableSlicePtrEqualComparableSlice compares a pointer to a slice of
// comparable types with a slice of comparable types.
// If the pointer is nil, it returns true.
// Otherwise, it dereferences the pointer and compares the slice with the
// provided slice of comparable types.
func IsComparableSlicePtrEqualComparableSlice[T comparable](ptr *[]T, val []T) bool {
	// if ptr is nil, consider it equal (no difference between nil and any value)
	if ptr == nil {
		return true
	}
	// use cmp library to compare dereferenced ptr with val
	return cmp.Equal(*ptr, val, cmpopts.EquateEmpty())
}

// IsComparableMapPtrEqualComparableMap compares a pointer to a map of
// comparable types with a map of comparable types.
// If the pointer is nil, it returns true.
// Otherwise, it dereferences the pointer and compares the map with the
// provided map of comparable types.
func IsComparableMapPtrEqualComparableMap[K comparable, V comparable](ptr *map[K]V, val map[K]V) bool {
	// if ptr is nil, consider it equal (no difference between nil and any value)
	if ptr == nil {
		return true
	}
	// use cmp library to compare dereferenced ptr with val
	return cmp.Equal(*ptr, val, cmpopts.EquateEmpty())
}

// IsComparablePtrEqualComparablePtr compares two pointers to comparable types.
// If both pointers are nil, it returns true.
// If one pointer is nil and the other is not, it returns false.
// Otherwise, it dereferences both pointers and compares their values.
func IsComparablePtrEqualComparablePtr[T comparable](ptr1, ptr2 *T) bool {
	// if both pointers are nil, consider them equal
	if ptr1 == nil && ptr2 == nil {
		return true
	}
	// if one pointer is nil and the other is not, consider them not equal
	if ptr1 == nil || ptr2 == nil {
		return false
	}
	// use cmp library to compare dereferenced ptr1 with dereferenced ptr2
	return cmp.Equal(*ptr1, *ptr2)
}

// AreStringSlicesEqual compares two slices of strings for equality, ignoring
// the order of elements.
// It returns true if both slices contain the same strings, regardless of their
// order, and false otherwise.
func AreStringSlicesEqual(sliceA, sliceB []string) bool {
	if len(sliceA) != len(sliceB) {
		return false
	}

	counts := make(map[string]int, len(sliceA)+len(sliceB))
	// Count the occurrences of each string in sliceA
	for _, s := range sliceA {
		counts[s]++
	}
	// Subtract the counts based on the occurrences in sliceB
	for _, s := range sliceB {
		counts[s]--
		if counts[s] < 0 {
			return false
		}
	}

	return true
}

// AreStringSlicesEqualDeDuped compares two slices of strings for equality,
// ignoring the order of elements and duplicates.
// It returns true if both slices contain the same unique strings, regardless
// of their order and duplicates, and false otherwise.
func AreStringSlicesEqualDeDuped(sliceA, sliceB []string) bool {
	if len(sliceA) == 0 && len(sliceB) == 0 {
		return true
	}

	setA := make(map[string]struct{}, len(sliceA))
	setB := make(map[string]struct{}, len(sliceB))
	// Add unique strings from sliceA to setA
	for _, s := range sliceA {
		setA[s] = struct{}{}
	}
	// Add unique strings from sliceB to setB
	for _, s := range sliceB {
		setB[s] = struct{}{}
	}
	// Compare the sets for equality
	if len(setA) != len(setB) {
		return false
	}

	for s := range setA {
		if _, exists := setB[s]; !exists {
			return false
		}
	}

	return true
}

// AssignIfNil assigns the value to the pointer if the pointer is nil.
func AssignIfNil[T any](ptr **T, val T) {
	// return early if ptr is nil to avoid dereferencing a nil pointer
	if ptr == nil {
		return
	}
	// assign val to ptr if ptr is nil
	if *ptr == nil {
		*ptr = &val
	}
}

// IntTimestampToMetaTime returns nil if timestamp is nil, else returns the
// Converted metav1.Time from timestamp. Timestamp must be in milliseconds.
func IntTimestampToMetaTime(timestamp *int64) *metav1.Time {
	if timestamp == nil {
		return nil
	}

	return &metav1.Time{Time: time.UnixMilli(*timestamp)}
}

// TimeToMetaTime returns nil if parameter is nil, otherwise metav1.Time value.
func TimeToMetaTime(t *time.Time) *metav1.Time {
	if t == nil {
		return nil
	}

	return &metav1.Time{Time: *t}
}

// StringToMetaTime converts a string pointer to a *metav1.Time.
// It tries the following formats in order: RFC3339 (with colon TZ offset),
// ISO 8601 without colon TZ offset (e.g. SonarQube's "+0000" format),
// and date-only ("2006-01-02").
// Returns nil if the input is nil or cannot be parsed by any format.
func StringToMetaTime(value *string) *metav1.Time {
	if value == nil {
		return nil
	}

	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05-0700", time.DateOnly} {
		parsedTime, err := time.Parse(layout, *value)
		if err != nil {
			continue
		}

		mt := metav1.NewTime(parsedTime.UTC())

		return &mt
	}

	return nil
}

// AnySliceToStringSlice converts a []any to []string, skipping non-string
// elements.
func AnySliceToStringSlice(slice []any) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}

	return result
}

// AssignIfNonNil assigns a value to a pointer if the reference
// pointer is not nil.
// If the reference pointer is nil, it does nothing.
func AssignIfNonNil[T any](ptr, ref *T) {
	// return early if ptr is nil to avoid dereferencing a nil pointer
	if ptr == nil {
		return
	}
	// assign value of ref to ptr if ref is not nil
	if ref != nil {
		*ptr = *ref
	}
}

// NewStringSetFromSlice creates a new set of strings from a slice of strings.
func NewStringSetFromSlice(slice []string) map[string]struct{} {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	return set
}
