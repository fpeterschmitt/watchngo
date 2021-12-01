//go:generate mockgen -source=executor.go -destination=mock_executor_test.go -package=pkg_test Executor

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
	Exec(event NotificationEvent, eventFile string) error
}

func MakeCommand(cmdTemplate string, event NotificationEvent, eventFile string) string {
	command := strings.Replace(cmdTemplate, "%event.file", eventFile, -1)
	command = strings.Replace(command, "%event.op", event.Notification.String(), -1)
	return command
}

// NewExecutorPrintPath only prints to stdout the full file path that triggered an
// event, so you can pipe the output and do whatever you want with it.
func NewExecutorPrintPath(output io.Writer) Executor {
	return &printExec{output: output}
}

type printExec struct {
	output io.Writer
}

func (e *printExec) Running() bool {
	return false
}

func (e *printExec) Exec(event NotificationEvent, eventFile string) error {
	_, err := e.output.Write([]byte(eventFile + "\n"))
	return err
}

// NewExecutorUnixShell returns an executor that will run your command through
// /bin/sh -c "<command>". Your command will be quoted before to avoid any
// problems.
func NewExecutorUnixShell(output io.Writer, commandTemplate string) Executor {
	return &unixShellExec{
		rawExec:         NewExecutorRaw(output, "").(*rawExec),
		commandTemplate: commandTemplate,
	}
}

type unixShellExec struct {
	rawExec         *rawExec
	commandTemplate string
}

func (e *unixShellExec) Exec(event NotificationEvent, eventFile string) error {
	cmd := MakeCommand(e.commandTemplate, event, eventFile)
	return e.rawExec.ExecCommand([]string{"/bin/sh", "-c", cmd}...)
}

func (e *unixShellExec) Running() bool {
	return e.rawExec.Running()
}

// NewExecutorRaw will run your command without shell.
// FIXME: commands with arguments are NOT supported for now.
func NewExecutorRaw(output io.Writer, commandTemplate string) Executor {
	return &rawExec{output: output, commandTemplate: commandTemplate}
}

type rawExec struct {
	commandTemplate string
	lock            sync.RWMutex
	executing       bool
	output          io.Writer
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

func (e *rawExec) ExecCommand(params ...string) error {
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

func (e *rawExec) Exec(event NotificationEvent, eventFile string) error {
	e.setExecuting(true)
	defer e.setExecuting(false)

	params := strings.SplitN(MakeCommand(e.commandTemplate, event, eventFile), " ", 1)

	return e.ExecCommand(params...)
}
