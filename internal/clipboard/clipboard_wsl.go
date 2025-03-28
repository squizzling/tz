//go:build !windows

package clipboard

import (
	"os/exec"
)

func Set(s string) {
	// lol
	if err := exec.Command("paste.exe", s).Start(); err != nil {
		panic(err)
	}
}
