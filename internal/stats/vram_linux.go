//go:build linux

package stats

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// detectVRAM tries multiple GPU back-ends in order of preference and returns
// (usedBytes, totalBytes, sourceName). All failures are silently ignored; if no
// GPU is detected the caller receives (0, 0, "").
func detectVRAM() (used, total uint64, source string) {
	// 1. NVIDIA — nvidia-smi (discrete GPU on workstation)
	if u, t, err := nvidiaVRAM(); err == nil {
		return u, t, "nvidia"
	}

	// 2. AMD — sysfs mem_info (discrete GPU on workstation / some laptops)
	if u, t, err := amdVRAM(); err == nil {
		return u, t, "amd"
	}

	// 3. Raspberry Pi VideoCore — vcgencmd
	if u, t, err := piVRAM(); err == nil {
		return u, t, "videocore"
	}

	return 0, 0, ""
}

// nvidiaVRAM queries nvidia-smi for the first GPU's memory stats.
func nvidiaVRAM() (used, total uint64, err error) {
	out, err := exec.Command(
		"nvidia-smi",
		"--query-gpu=memory.used,memory.total",
		"--format=csv,noheader,nounits",
	).Output()
	if err != nil {
		return 0, 0, err
	}

	// Output: "1234, 8192\n"  (values in MiB)
	parts := strings.SplitN(strings.TrimSpace(string(out)), ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected nvidia-smi output: %q", string(out))
	}
	usedMiB, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	totalMiB, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return usedMiB << 20, totalMiB << 20, nil
}

// amdVRAM reads VRAM stats from the DRM sysfs interface used by amdgpu.
func amdVRAM() (used, total uint64, err error) {
	// Look for the first card that exposes mem_info_vram_used
	pattern := "/sys/class/drm/card*/device/mem_info_vram_used"
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return 0, 0, fmt.Errorf("no amdgpu sysfs entry found")
	}

	usedPath := matches[0]
	totalPath := strings.Replace(usedPath, "vram_used", "vram_total", 1)

	usedData, err := os.ReadFile(usedPath)
	if err != nil {
		return 0, 0, err
	}
	totalData, err := os.ReadFile(totalPath)
	if err != nil {
		return 0, 0, err
	}

	used, err = strconv.ParseUint(strings.TrimSpace(string(usedData)), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	total, err = strconv.ParseUint(strings.TrimSpace(string(totalData)), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return used, total, nil
}

// piVRAM reads the VideoCore GPU memory allocation via vcgencmd.
// The Pi doesn't expose used VRAM, so used is always reported as 0.
func piVRAM() (used, total uint64, err error) {
	out, err := exec.Command("vcgencmd", "get_mem", "gpu").Output()
	if err != nil {
		return 0, 0, err
	}

	// Output: "gpu=64M\n"
	s := strings.TrimSpace(string(out))
	s = strings.TrimPrefix(s, "gpu=")
	s = strings.TrimSuffix(s, "M")

	totalMiB, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse vcgencmd output %q: %w", string(out), err)
	}
	return 0, totalMiB << 20, nil
}
