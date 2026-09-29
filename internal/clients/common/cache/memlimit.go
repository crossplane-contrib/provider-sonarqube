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
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

const (
	// cgroupRoot is where the container's cgroup filesystem is mounted.
	cgroupRoot = "/sys/fs/cgroup"
	// cgroupV2MemoryMax is the cgroup v2 memory limit file, relative to
	// cgroupRoot.
	cgroupV2MemoryMax = "memory.max"
	// cgroupV1MemoryLimit is the cgroup v1 memory limit file, relative to
	// cgroupRoot.
	cgroupV1MemoryLimit = "memory/memory.limit_in_bytes"
	// cgroupV2Unlimited is the content of memory.max without a limit.
	cgroupV2Unlimited = "max"
	// cgroupV1Unlimited is the lowest value cgroup v1 reports without a
	// limit: the kernel reports the largest page-aligned int64, so any
	// value this high is not a real limit.
	cgroupV1Unlimited int64 = 1 << 62
)

// ContainerMemoryLimit returns the memory limit of the container the process
// runs in, read from its cgroup (v2, then v1). limited is false when the
// container has no memory limit or no cgroup memory controller is mounted.
func ContainerMemoryLimit() (limit int64, limited bool, err error) {
	return containerMemoryLimit(os.DirFS(cgroupRoot))
}

// containerMemoryLimit reads the memory limit from the cgroup filesystem
// cgroupFS.
func containerMemoryLimit(cgroupFS fs.FS) (limit int64, limited bool, err error) {
	for _, file := range []string{cgroupV2MemoryMax, cgroupV1MemoryLimit} {
		content, readErr := fs.ReadFile(cgroupFS, file)
		if errors.Is(readErr, fs.ErrNotExist) {
			continue
		}

		if readErr != nil {
			return 0, false, errors.Wrapf(readErr, "cannot read cgroup memory limit from %s", file)
		}

		return parseCgroupMemoryLimit(file, strings.TrimSpace(string(content)))
	}

	return 0, false, nil
}

// parseCgroupMemoryLimit parses the content of the cgroup memory limit file.
func parseCgroupMemoryLimit(file, content string) (limit int64, limited bool, err error) {
	if content == cgroupV2Unlimited {
		return 0, false, nil
	}

	limit, err = strconv.ParseInt(content, 10, 64)
	if err != nil {
		return 0, false, errors.Wrapf(err, "cannot parse cgroup memory limit %q from %s", content, file)
	}

	if limit <= 0 || limit >= cgroupV1Unlimited {
		return 0, false, nil
	}

	return limit, true, nil
}

// ResolveMaxBytes returns the size budget of the cache.
//
// An explicit budget (greater than 0) wins, but must stay below the container
// memory limit when there is one. Without an explicit budget, the budget is
// memoryFraction of the container memory limit, or DefaultMaxBytes when the
// container has no memory limit. memoryFraction must be in (0, 1].
func ResolveMaxBytes(explicit int64, memoryFraction float64, limit int64, limited bool) (int64, error) {
	if memoryFraction <= 0 || memoryFraction > 1 {
		return 0, errors.Errorf("observe cache memory fraction must be greater than 0 and at most 1, got %v", memoryFraction)
	}

	if explicit < 0 {
		return 0, errors.Errorf("observe cache max bytes must not be negative, got %d", explicit)
	}

	if explicit > 0 {
		if limited && explicit >= limit {
			return 0, errors.Errorf("observe cache max bytes (%d) must be lower than the container memory limit (%d)", explicit, limit)
		}

		return explicit, nil
	}

	if !limited {
		return DefaultMaxBytes, nil
	}

	return max(int64(float64(limit)*memoryFraction), 1), nil
}
