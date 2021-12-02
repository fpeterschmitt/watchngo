package pkg_test

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/Leryan/watchngo/pkg"

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
	makefile(tempdir, "sub2", "f1")

	return tempdir, files
}

type testNotifier struct {
	suite.Suite
	tempdir   string
	tempfiles []string
	notifier  pkg.Notifier
}

func TestNotifiers(t *testing.T) {
	suite.Run(t, &testNotifier{})
}

func (t *testNotifier) SetupTest() {
	t.tempdir, t.tempfiles = setupTempFiles(t.T())
}

func (t *testNotifier) TearDownTest() {
	t.NoError(os.RemoveAll(t.tempdir), "cleanup")
}

func (t *testNotifier) TestNotifier() {
	var events <-chan pkg.NotificationEvent
	watchedFile := t.tempfiles[0]
	freeFile := t.tempfiles[1]

	writeEvent := func(file string) {
		fh, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0660)
		t.Require().NoError(err, "opening file %s", file)
		defer fh.Close()
		_, err = fh.WriteString(file + " - append\n")
		t.Require().NoError(err, "writing to file %s", file)
	}

	zeroEvents := func() {
		select {
		case event := <-events:
			t.FailNow("must not have an event: %v", event)
		case tt := <-time.NewTimer(time.Millisecond * 500).C:
			t.NotEmpty(tt)
		}
	}

	mustPullEvent := func() pkg.NotificationEvent {
		select {
		case event := <-events:
			return event
		case <-time.NewTimer(time.Second).C:
			t.FailNow("no event available after waiting 1 second")
		}
		return pkg.NotificationEvent{Error: fmt.Errorf("no event pulled, should never arrive here")}
	}

	t.Run("fsnotify", func() {
		t.notifier = pkg.NewFSNotifyNotifier()
		defer t.notifier.Close()
		t.Require().NoError(t.notifier.Add(watchedFile))
		events = t.notifier.Events()

		t.Run("boot", func() {
			zeroEvents()
		})

		t.Run("no event on unwatched file", func() {
			writeEvent(freeFile)
			zeroEvents()
		})

		t.Run("write event", func() {
			writeEvent(watchedFile)
			event := mustPullEvent()
			t.Equal(pkg.NotificationWrite, event.Notification)
			t.Equal(watchedFile, event.Path)
			t.NoError(event.Error)
			t.Equal(pkg.FileTypeFile, event.FileType)
		})

		t.Run("chmod event", func() {
			t.NoError(os.Chmod(watchedFile, 0640))
			event := mustPullEvent()
			t.Equal(pkg.NotificationChmod, event.Notification)
			t.Equal(watchedFile, event.Path)
			t.NoError(event.Error)
			t.Equal(pkg.FileTypeFile, event.FileType)
		})

		t.Run("rename event", func() {
			t.NoError(os.Rename(watchedFile, watchedFile+".renamed"))
			event := mustPullEvent()
			t.Equal(pkg.NotificationRename, event.Notification)
			t.Equal(watchedFile, event.Path)
			t.NoError(event.Error)
			t.Equal(pkg.FileTypeFile, event.FileType)

			writeEvent(watchedFile + ".renamed")
			zeroEvents()

			watchedFile = watchedFile + ".renamed"
			t.Require().NoError(t.notifier.Add(watchedFile))
		})

		t.Run("remove event", func() {
			t.NoError(os.Remove(watchedFile))
			event := mustPullEvent()
			t.Equal(pkg.NotificationRemove, event.Notification)
			t.Equal(watchedFile, event.Path)
			t.NoError(event.Error)
			t.Equal(pkg.FileTypeFile, event.FileType)

			writeEvent(watchedFile)
			zeroEvents()
		})

		t.Run("create event - not watched", func() {
			writeEvent(watchedFile + ".unwatched")
			zeroEvents()
		})

		t.Run("create event - watch top directory, write in subdir", func() {
			t.Require().NoError(t.notifier.Add(t.tempdir))
			writeEvent(watchedFile + ".dirwatched")
			zeroEvents()
		})

		t.Run("create event - watch subdir, create in subdir", func() {
			watchedDir := path.Join(t.tempdir, "sub1")
			t.T().Logf("%s -> %s", watchedDir, watchedFile)
			t.Require().NoError(t.notifier.Add(watchedDir))

			fh, err := os.OpenFile(watchedFile+".dirwatched", os.O_CREATE|os.O_WRONLY, 0660)
			t.Require().NoError(err, "create file")
			fh.Write([]byte("coucou"))
			fh.Close()
			event := mustPullEvent()

			t.Equal(pkg.NotificationWrite, event.Notification) // didn't manage to actually get a CREATE event with fsnotify on linux.
			t.Equal(watchedFile+".dirwatched", event.Path)
			t.NoError(event.Error)
			t.Equal(pkg.FileTypeFile, event.FileType)
		})
	})
}
