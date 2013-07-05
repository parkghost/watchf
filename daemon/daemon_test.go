package daemon

import (
	"os"
	"testing"
)

type DummyService struct {
	running bool
}

func (s *DummyService) Start() error {
	s.running = true
	return nil
}

func (s *DummyService) Stop() error {
	s.running = false
	return nil
}

func TestForegroundDaemonStartAndStop(t *testing.T) {
	dummyService := &DummyService{}
	dmon := NewDaemon("dummy", dummyService)

	var err error
	err = dmon.Start()
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(dmon.getPidFilename())
	if err != nil {
		t.Fatal("started: pid file not found")
	}

	if dummyService.running != dmon.running {
		t.Fatal("started: service and daemon have different running state")
	}

	err = dmon.Stop()
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(dmon.getPidFilename())
	if err == nil {
		t.Fatal("stopped: pid file should be removed")
	}

	if dummyService.running != dmon.running {
		t.Fatal("stopped: service and daemon have different running state")
	}
}
