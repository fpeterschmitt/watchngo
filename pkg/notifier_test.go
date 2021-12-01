package pkg_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/stretchr/testify/require"
)

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

type testNotifier struct {
	suite.Suite
	tempdir   string
	tempfiles []string
}

func (t *testNotifier) append(file string) {
	fh, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0)
	t.Require().NoError(err, "opening file %s", file)
	defer fh.Close()
	_, err = fh.WriteString(file + " - append\n")
	t.Require().NoError(err, "writing to file %s", file)
	t.Require().NoError(fh.Sync())
	os.Chtimes(file, time.Now().Local(), time.Now().Local())
}

func (t *testNotifier) SetupTest() {
	t.tempdir, t.tempfiles = setupTempFiles(t.T())
}

func (t *testNotifier) TearDownTest() {
	t.NoError(os.RemoveAll(t.tempdir), "cleanup")
}

func (t *testNotifier) runTestCase(testCase testCase) {
	t.append(testCase.appendFile)
	time.Sleep(time.Millisecond * 500)
}
