package pkg_test

import (
	"bytes"
	"log"
	"os"
	"path"
	"strings"
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
	makefile(tempdir, "sub1", "ex1")
	makefile(tempdir, "sub2", "f1")
	makefile(tempdir, "sub2", "f2")
	makefile(tempdir, "sub2", "ex2")

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
	t.NoError(os.RemoveAll(t.tempdir), "cleanup")
}

type testCase struct {
	appendFile     string
	executorRan    int
	executorParams []string
}

func (t *testWatcher) runTestCase(testCase testCase) {
	t.executor.reset()
	t.append(testCase.appendFile)
	time.Sleep(time.Millisecond * 500)

	if t.Equal(testCase.executorRan, t.executor.ran) && testCase.executorRan > 0 {
		t.Equal(testCase.executorParams, t.executor.params)
	}
}

func (t *testWatcher) TestAllSubAllFiles() {
	logout := bytes.Buffer{}
	logger := log.New(&logout, "", log.LstdFlags)
	watcher, err := pkg.NewWatcher(t.T().Name(), t.tempdir, "f.*", "ls %event.file", t.executor, false, logger)
	t.Require().NoError(err)

	go func() { t.Require().NoError(watcher.Work()) }()
	time.Sleep(time.Millisecond * 200)

	testCases := []testCase{
		{
			appendFile:     t.tempfiles[0], // f1
			executorRan:    1,
			executorParams: []string{"ls " + t.tempfiles[0]},
		},
		{
			appendFile:  t.tempfiles[2], // ex1
			executorRan: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.appendFile, func() {
			t.runTestCase(tc)
		})
	}

	t.Run("logs", func() {
		t.Len(strings.Split(logout.String(), "\n"), 4)
		t.Contains(logout.String(), "running watcher")
		t.Contains(logout.String(), "running command on watcher")
		t.Contains(logout.String(), "finished running command on watcher")
	})
}
