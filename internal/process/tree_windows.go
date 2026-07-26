//go:build windows

package process

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

const (
	jobObjectExtendedLimitInformationClass = 9
	jobObjectLimitKillOnJobClose           = 0x00002000
	processSetQuota                        = 0x0100
	processTerminate                       = 0x0001
)

type jobObjectBasicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}
type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}
type jobObjectExtendedLimitInformation struct {
	BasicLimitInformation jobObjectBasicLimitInformation
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var createJobObject = kernel32.NewProc("CreateJobObjectW")
var setInformationJobObject = kernel32.NewProc("SetInformationJobObject")
var assignProcessToJobObject = kernel32.NewProc("AssignProcessToJobObject")
var terminateJobObject = kernel32.NewProc("TerminateJobObject")
var openProcess = kernel32.NewProc("OpenProcess")
var closeHandle = kernel32.NewProc("CloseHandle")
var commandJobs sync.Map

func configureTree(command *exec.Cmd) error {
	handle, _, callErr := createJobObject.Call(0, 0)
	if handle == 0 {
		return errors.New("create Windows Job Object: " + callErr.Error())
	}
	info := jobObjectExtendedLimitInformation{}
	info.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose
	result, _, callErr := setInformationJobObject.Call(handle, jobObjectExtendedLimitInformationClass, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info))
	if result == 0 {
		closeHandle.Call(handle)
		return errors.New("configure Windows Job Object: " + callErr.Error())
	}
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	commandJobs.Store(command, syscall.Handle(handle))
	return nil
}

func afterStart(command *exec.Cmd) error {
	value, ok := commandJobs.Load(command)
	if !ok {
		return errors.New("Windows Job Object missing")
	}
	job := value.(syscall.Handle)
	process, _, callErr := openProcess.Call(processSetQuota|processTerminate, 0, uintptr(command.Process.Pid))
	if process == 0 {
		return errors.New("open child process for Job Object: " + callErr.Error())
	}
	defer closeHandle.Call(process)
	result, _, callErr := assignProcessToJobObject.Call(uintptr(job), process)
	if result == 0 {
		return errors.New("assign child process to Job Object: " + callErr.Error())
	}
	return nil
}

func finalizeTree(command *exec.Cmd) {
	if value, ok := commandJobs.LoadAndDelete(command); ok {
		closeHandle.Call(uintptr(value.(syscall.Handle)))
	}
}

func watchCancellation(ctx context.Context, command *exec.Cmd, done <-chan struct{}) {
	go func() {
		select {
		case <-ctx.Done():
			terminateTree(command)
		case <-done:
		}
	}()
}

func terminateTree(command *exec.Cmd) {
	if value, ok := commandJobs.Load(command); ok {
		terminateJobObject.Call(uintptr(value.(syscall.Handle)), 1)
		return
	}
	if command.Process != nil {
		_ = command.Process.Kill()
	}
}
