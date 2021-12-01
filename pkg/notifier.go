//go:generate stringer -type=Notification -trimprefix=Notification

package pkg

import (
	"os"
	"path"

	"github.com/fsnotify/fsnotify"
)

type Notification int

const (
	NotificationRemove Notification = 1 << iota
	NotificationCreate
	NotificationWrite
	NotificationRename
	NotificationChmod
	NotificationError
)

type FileType int

const (
	FileTypeDir FileType = iota
	FileTypeFile
)

type NotificationEvent struct {
	Path         string
	Notification Notification
	FileType     FileType
	Error        error
}

type Notifier interface {
	// Events should not be called more than once, and the returned channel is to be reused.
	Events() <-chan NotificationEvent
	Add(file string) error
	Remove(file string) error
	Close() error
}

type fsnotifyNotifier struct {
	FSWatcher *fsnotify.Watcher
}

func (f fsnotifyNotifier) handleEvent(event fsnotify.Event) NotificationEvent {
	var n Notification
	if fsnotify.Write&event.Op > 0 {
		n |= NotificationWrite
	}
	if fsnotify.Chmod&event.Op > 0 {
		n |= NotificationChmod
	}
	if fsnotify.Rename&event.Op > 0 {
		n |= NotificationRename
	}
	if fsnotify.Remove&event.Op > 0 {
		n |= NotificationRemove
	}
	if fsnotify.Create&event.Op > 0 {
		n |= NotificationCreate
	}

	fpath := path.Clean(event.Name)
	ft := FileTypeFile

	fi, err := os.Stat(fpath)
	if err == nil {
		if fi.IsDir() {
			ft = FileTypeDir
		}
	} else {
		n |= NotificationError
	}

	return NotificationEvent{
		Path:         fpath,
		Notification: n,
		FileType:     ft,
		Error:        err,
	}
}

func (f fsnotifyNotifier) Events() <-chan NotificationEvent {
	out := make(chan NotificationEvent)

	go func() {
		defer close(out)

		for {
			select {
			case event := <-f.FSWatcher.Events:
				out <- f.handleEvent(event)
			case err := <-f.FSWatcher.Errors:
				out <- NotificationEvent{
					Notification: NotificationError,
					Error:        err,
				}
			}
		}
	}()

	return out
}

func (f fsnotifyNotifier) Add(file string) error {
	return f.FSWatcher.Add(file)
}

func (f fsnotifyNotifier) Remove(file string) error {
	return f.FSWatcher.Remove(file)
}

func (f fsnotifyNotifier) Close() error {
	return f.FSWatcher.Close()
}

func NewFSNotifyNotifier() Notifier {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	return fsnotifyNotifier{FSWatcher: fsw}
}
