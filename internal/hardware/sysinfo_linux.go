package hardware

import (
	"syscall"
)

func getTotalSystemMemory() uint64 {
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return 8 * 1024 * 1024 * 1024
	}
	return info.Totalram * uint64(info.Unit)
}
