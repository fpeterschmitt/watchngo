package pkg_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Leryan/watchngo/pkg"
)

func TestUnixShellExec(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skipf("cannot run test requirering /bin/sh: %v", err)
	}

	out := bytes.Buffer{}
	exec := pkg.NewExecutorUnixShell(&out, "sleep 1")

	require.False(t, exec.Running())

	go func() {
		require.NoError(t, exec.Exec(pkg.NotificationEvent{}, "none"))
	}()

	time.Sleep(time.Millisecond * 100)
	require.True(t, exec.Running())
	time.Sleep(time.Millisecond * 1500)
	require.False(t, exec.Running())
}
