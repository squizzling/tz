package clipboard

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var kernel32 = syscall.MustLoadDLL("kernel32")
var globalAlloc = kernel32.MustFindProc("GlobalAlloc")
var globalFree = kernel32.MustFindProc("GlobalFree")
var globalLock = kernel32.MustFindProc("GlobalLock")
var globalUnlock = kernel32.MustFindProc("GlobalUnlock")

var user32 = syscall.MustLoadDLL("user32")
var closeClipboard = user32.MustFindProc("CloseClipboard")
var emptyClipboard = user32.MustFindProc("EmptyClipboard")
var openClipboard = user32.MustFindProc("OpenClipboard")
var setClipboardData = user32.MustFindProc("SetClipboardData")

func copyStr(s string) (uintptr, error) {
	r := []byte(s)
	hMem, _, err := globalAlloc.Call(2, uintptr(len(r)+1))
	if hMem == 0 {
		fmt.Printf("globalAlloc failed\n")
		return 0, err
	}

	pMem, _, err := globalLock.Call(hMem)
	if pMem == 0 {
		fmt.Printf("globalLock failed\n")
		_, _, _ = globalFree.Call(hMem) // We can't do anything if it fails
		return 0, err
	}

	sMem := unsafe.Slice((*byte)(unsafe.Pointer(pMem)), len(r)+1)
	copy(sMem, r)
	sMem[len(r)] = 0

	if _, _, err := globalUnlock.Call(pMem); err.(windows.Errno) != 0 {
		fmt.Printf("globalUnlock failed\n")
		_, _, _ = globalFree.Call(hMem) // This will probably fail because it's still locked, but we can't do anything
		return 0, err
	}

	return pMem, nil
}

func freeStr(hMem uintptr) error {
	if rc, _, err := globalFree.Call(hMem); rc != 0 {
		fmt.Printf("globalFree failed\n")
		return err
	}

	return nil
}

func setClipboard(hMem uintptr) error {
	if rc, _, err := openClipboard.Call(uintptr(0)); rc == 0 {
		fmt.Printf("openClipboard failed\n")
		return err
	}

	defer func() {
		if rc, _, err := closeClipboard.Call(); rc == 0 {
			panic(err)
		}
	}()

	if rc, _, err := emptyClipboard.Call(); rc == 0 {
		fmt.Printf("emptyClipboard failed\n")
		return err
	}

	if rc, _, err := setClipboardData.Call(1, hMem); rc == 0 {
		fmt.Printf("setClipboardData failed\n")
		return err
	}

	return nil
}

func Set(s string) error {
	hMem, err := copyStr(s)
	if err != nil {
		return err
	}

	errCB := setClipboard(hMem)
	err = freeStr(hMem)
	if errCB != nil {
		return errCB
	}
	return err
}
