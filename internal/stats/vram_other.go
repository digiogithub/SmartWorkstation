//go:build !linux

package stats

// detectVRAM is a no-op stub for non-Linux platforms.
// All runtime targets (AMD64 Linux, ARM Linux) use vram_linux.go.
func detectVRAM() (used, total uint64, source string) {
	return 0, 0, ""
}
