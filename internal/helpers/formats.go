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
	"io"
	"net/http"
	"time"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CloseBody closes the body of an http.Response safely.
// If the response or body is nil, it does nothing.
// The error return value of Close is intentionally ignored.
func CloseBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

// IsComparablePtrEqualComparable compares a pointer to a comparable type with a comparable type.
// If the pointer is nil, it returns true.
// Otherwise, it dereferences the pointer and compares the value with the provided comparable type.
func IsComparablePtrEqualComparable[T comparable](ptr *T, val T) bool {
	// if ptr is nil, consider it equal (no difference between nil and any value)
	if ptr == nil {
		return true
	}
	// use cmp library to compare dereferenced ptr with val
	return cmp.Equal(*ptr, val)
}

// IsComparablePtrEqualComparablePtr compares two pointers to comparable types.
// If both pointers are nil, it returns true.
// If one pointer is nil and the other is not, it returns false.
// Otherwise, it dereferences both pointers and compares their values.
func IsComparablePtrEqualComparablePtr[T comparable](ptr1 *T, ptr2 *T) bool {
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

// TimeToMetaTime returns nil if parameter is nil, otherwise metav1.Time value
func TimeToMetaTime(t *time.Time) *metav1.Time {
	if t == nil {
		return nil
	}
	return &metav1.Time{Time: *t}
}

// StringToMetaTime converts a string pointer representing a time in RFC3339 format to a metav1.Time pointer.
// Returns nil if the input string pointer is nil or if parsing fails.
func StringToMetaTime(s *string) *metav1.Time {
	if s == nil {
		return nil
	}
	parsedTime, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &metav1.Time{Time: parsedTime}
}

// MapToSemicolonSeparatedString converts a map to a semicolon-separated key=value string
func MapToSemicolonSeparatedString(m *map[string]string) string {
	if m == nil || len(*m) == 0 {
		return ""
	}
	result := ""
	for k, v := range *m {
		if result != "" {
			result += ";"
		}
		result += k + "=" + v
	}
	return result
}

// AnySliceToStringSlice converts a []any to []string, skipping non-string elements
func AnySliceToStringSlice(slice []any) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
