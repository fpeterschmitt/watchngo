package pkg

import (
	"bufio"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"
	"sync"
)

// Executor provides a minimal workflow to run commands.
type Executor interface {
	// Running must return true when the command is still running.
	Running() bool
	// Exec takes command program with first param, then arguments.
	Exec(params ...string) error
}

// NewPrintExec only prints to stdout the full file path that triggered an
// event so you can pipe the output and do whatever you want with it.
func NewPrintExec(output io.Writer) Executor {
	return &printExec{output: output}
}

type printExec struct {
	output io.Writer
}

func (e *printExec) Running() bool {
	return false
}

func (e *printExec) Exec(params ...string) error {
	_, err := e.output.Write([]byte(strings.Join(params, " ") + "\n"))
	return err
}

// NewUnixShellExec returns an executor that will run your command through
// /bin/sh -c "<command>". Your command will be quoted before to avoid any
// problems.
func NewUnixShellExec(output io.Writer) Executor {
	return &unixShellExec{
		rawExec: NewRawExec(output),
	}
}

type unixShellExec struct {
	rawExec Executor
}

func (e *unixShellExec) Exec(params ...string) error {
	return e.rawExec.Exec(append([]string{"/bin/sh", "-c"}, params...)...)
}

func (e *unixShellExec) Running() bool {
	return e.rawExec.Running()
}

// NewRawExec will run your command without shell.
// FIXME: commands with arguments are NOT supported for now.
func NewRawExec(output io.Writer) Executor {
	return &rawExec{output: output}
}

type rawExec struct {
	lock      sync.RWMutex
	executing bool
	output    io.Writer
}

func (e *rawExec) setExecuting(on bool) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.executing = on
}

func (e *rawExec) Running() bool {
	e.lock.Lock()
	defer e.lock.Unlock()
	return e.executing
}

func (e *rawExec) Exec(params ...string) error {
	e.setExecuting(true)
	defer e.setExecuting(false)

	rp, wp := io.Pipe()
	var cmd *exec.Cmd
	var execError error

	cmd = exec.Command(params[0], params[1:]...)
	cmd.Stdout = wp
	cmd.Stderr = wp

	execFinished := make(chan bool, 1)

	go func() {
		if err := cmd.Run(); err != nil {
			execError = err
		}
		wp.Close()
		execFinished <- true
	}()

	reader := bufio.NewReader(rp)

	for {
		b, err := reader.ReadBytes('\n')

		if len(b) > 0 {
			e.output.Write(b)
		}

		if err != nil {
			break
		}
	}

	if reader.Buffered() > 0 {
		b, _ := ioutil.ReadAll(reader)
		e.output.Write(b)
	}

	<-execFinished

	return execError
}
