package pkg_test

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/Leryan/watchngo/pkg"
	"github.com/stretchr/testify/suite"
)

type testWatcher struct {
	suite.Suite
	ctrl     *gomock.Controller
	finder   *MockFinder
	filter   *MockFilter
	notifier *MockNotifier
	executor *MockExecutor
	logger   *MockLogger
	watcher  *pkg.Watcher
}

func TestWatcher(t *testing.T) {
	suite.Run(t, &testWatcher{})
}

func (t *testWatcher) SetupTest() {
	t.ctrl = gomock.NewController(t.T())
	t.finder = NewMockFinder(t.ctrl)
	t.filter = NewMockFilter(t.ctrl)
	t.notifier = NewMockNotifier(t.ctrl)
	t.executor = NewMockExecutor(t.ctrl)
	t.logger = NewMockLogger(t.ctrl)

	var err error
	t.watcher, err = pkg.NewWatcher(t.T().Name(), t.finder, t.filter, t.notifier, t.executor, t.logger)
	t.Require().NoError(err, "init watcher")
}

func (t *testWatcher) TearDownTest() {
	t.ctrl.Finish()
}

type testCase struct {
	appendFile     string
	executorRan    int
	executorParams []string
}

func (t *testWatcher) TestMatchFilterLogExecute() {
	notifications := make(chan pkg.NotificationEvent, 1)

	t.logger.EXPECT().Log(gomock.Any(), gomock.Any()).AnyTimes()
	t.logger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()

	gomock.InOrder(
		t.finder.EXPECT().Find().Return(&pkg.FinderResults{Locations: []string{"sub1/f1", "sub2/f1", "sub1/f2"}}, nil),
		t.notifier.EXPECT().Add("sub1/f1").Return(nil),
		t.notifier.EXPECT().Add("sub2/f1").Return(nil),
		t.notifier.EXPECT().Add("sub1/f2").Return(nil),

		t.notifier.EXPECT().Events().Return(notifications),

		t.filter.EXPECT().MatchString("sub1/f1").Return(true),
		t.executor.EXPECT().Running().Return(false),
		t.executor.EXPECT().Exec(gomock.Any(), "sub1/f1").Times(1),
	)

	go func() { t.Require().NoError(t.watcher.Work()) }()
	time.Sleep(time.Millisecond * 200)

	notifications <- pkg.NotificationEvent{
		Path:         "sub1/f1",
		Notification: pkg.NotificationWrite,
		FileType:     pkg.FileTypeFile,
		Error:        nil,
	}

	time.Sleep(time.Millisecond * 500)
}
