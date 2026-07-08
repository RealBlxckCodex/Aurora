package hardware

import (
	"fmt"
	"os/exec"
)

func getTotalSystemMemory() uint64 {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 8 * 1024 * 1024 * 1024
	}
	var mem uint64
	if _, err := fmt.Sscanf(string(out), "%d", &mem); err != nil {
		return 8 * 1024 * 1024 * 1024
	}
	return mem
}
