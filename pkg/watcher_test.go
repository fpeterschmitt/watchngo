package pkg_test

import (
	"bytes"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/Leryan/watchngo/pkg"
	"github.com/stretchr/testify/suite"

	"github.com/stretchr/testify/require"
)

type mockExecutor struct {
	running bool
	params  []string
	ran     int
}

func (m *mockExecutor) Running() bool {
	return m.running
}

func (m *mockExecutor) Exec(params ...string) error {
	m.params = params
	m.ran += 1
	return nil
}

func (m *mockExecutor) reset() {
	m.params = nil
	m.ran = 0
}

func setupTempFiles(t *testing.T) (string, []string) {
	tempdir, err := os.MkdirTemp(".", "watcher_test.go-")
	require.NoError(t, err)

	require.NoError(t, os.MkdirAll(path.Join(tempdir, "sub1"), 0770))
	require.NoError(t, os.MkdirAll(path.Join(tempdir, "sub2"), 0770))

	files := make([]string, 0)

	makefile := func(paths ...string) {
		fpath := path.Join(paths...)
		fh, err := os.Create(fpath)
		require.NoError(t, err)
		defer fh.Close()
		_, err = fh.WriteString(fpath + "\n")
		require.NoError(t, err)

		files = append(files, fpath)
	}

	makefile(tempdir, "sub1", "f1")
	makefile(tempdir, "sub1", "f2")
	makefile(tempdir, "sub2", "f1")
	makefile(tempdir, "sub2", "f2")

	return tempdir, files
}

type testWatcher struct {
	suite.Suite
	executor  *mockExecutor
	tempdir   string
	tempfiles []string
}

func TestWatcher(t *testing.T) {
	suite.Run(t, &testWatcher{})
}

func (t *testWatcher) append(file string) {
	fh, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0)
	t.Require().NoError(err, "opening file %s", file)
	defer fh.Close()
	_, err = fh.WriteString(file + " - append\n")
	t.Require().NoError(err, "writing to file %s", file)
	t.Require().NoError(fh.Sync())
	os.Chtimes(file, time.Now().Local(), time.Now().Local())
}

func (t *testWatcher) SetupTest() {
	t.executor = &mockExecutor{}
	t.tempdir, t.tempfiles = setupTempFiles(t.T())
}

func (t *testWatcher) TearDownTest() {
	os.RemoveAll(t.tempdir)
}

func (t *testWatcher) TestAllSubAllFiles() {
	logout := bytes.Buffer{}
	logger := log.New(&logout, "", log.LstdFlags)
	watcher, err := pkg.NewWatcher(t.T().Name(), t.tempdir, ".*", "ls %event.file", t.executor, false, logger)
	t.Require().NoError(err)
	t.Require().NoError(watcher.Find())

	go func() { t.Require().NoError(err, watcher.Work()) }()
	time.Sleep(time.Millisecond * 200)

	for _, f := range t.tempfiles {
		t.Run(f, func() {
			t.executor.reset()

			t.append(f)
			time.Sleep(time.Millisecond * 500)
			t.Equal(1, t.executor.ran)
			if t.Len(t.executor.params, 1) {
				t.Equal("ls "+f, t.executor.params[0])
			}
		})
	}
}
