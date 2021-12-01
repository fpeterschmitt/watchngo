package pkg_test

import (
	"bytes"
	"log"
	"testing"
	"time"

	"github.com/Leryan/watchngo/pkg"
	"github.com/stretchr/testify/suite"
)

type mockExecutor struct {
	running bool
	params  []struct {
		Event pkg.NotificationEvent
		File  string
	}
}

func (m *mockExecutor) Running() bool {
	return m.running
}

func (m *mockExecutor) Exec(event pkg.NotificationEvent, eventFile string) error {
	m.params = append(m.params, struct {
		Event pkg.NotificationEvent
		File  string
	}{Event: event, File: eventFile})
	return nil
}

func (m *mockExecutor) reset() {
	m.params = nil
}

type mockNotifier struct {
	events chan pkg.NotificationEvent
}

func (m *mockNotifier) Events() <-chan pkg.NotificationEvent {
	return m.events
}

func (m *mockNotifier) Add(file string) error {
	return nil
}

func (m *mockNotifier) Remove(file string) error {
	return nil
}

func (m *mockNotifier) Close() error {
	close(m.events)
	return nil
}

type testWatcher struct {
	suite.Suite
	executor *mockExecutor
	notifier *mockNotifier
}

func TestWatcher(t *testing.T) {
	suite.Run(t, &testWatcher{})
}

func (t *testWatcher) SetupTest() {
	t.executor = &mockExecutor{}
}

type testCase struct {
	appendFile     string
	executorRan    int
	executorParams []string
}

func (t *testWatcher) TestAllSubAllFiles() {
	logout := bytes.Buffer{}
	logger := pkg.InfoLogger{Logger: log.New(&logout, "", log.LstdFlags)}

	var finder pkg.Finder
	var notifier pkg.Notifier
	var filter pkg.FilterRegexp

	watcher, err := pkg.NewWatcher(t.T().Name(), finder, filter, notifier, t.executor, logger)
	t.Require().NoError(err)

	go func() { t.Require().NoError(watcher.Work()) }()
	time.Sleep(time.Millisecond * 200)
}
