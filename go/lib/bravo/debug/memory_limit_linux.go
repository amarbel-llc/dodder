package debug

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// Read cgroup version from /proc/cgroups to determine if we're using cgroup v1 or v2
func checkCgroupVersion() string {
	data, err := os.ReadFile("/proc/cgroups")
	if err != nil {
		fmt.Println("Error reading /proc/cgroups:", err)
		return "unknown"
	}

	if strings.Contains(string(data), "memory") {
		return "cgroup v1"
	}
	return "cgroup v2"
}

// Read memory limit from the appropriate cgroup file
func getMemoryLimit() (uint64, error) {
	paths := []string{
		"/sys/fs/cgroup/memory.max",                   // cgroups v2
		"/sys/fs/cgroup/memory/memory.limit_in_bytes", // cgroups v1
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			str := strings.TrimSpace(string(data))
			if str == "max" {
				return 0, fmt.Errorf("no memory limit set")
			}
			val, err := strconv.ParseUint(str, 10, 64)
			if err == nil {
				return val, nil
			}
		}
	}
	return 0, fmt.Errorf("memory limit not found")
}

// Get the current memory usage of the Go process
func getProcessMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}
