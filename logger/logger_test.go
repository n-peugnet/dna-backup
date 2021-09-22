package logger

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLoggingBeforeInit(t *testing.T) {
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stderr = w
	// Reset
	initialize()

	info := "info log"
	warning := "warning log"
	errL := "error log"
	fatal := "fatal log"

	Info(info)
	Warning(warning)
	Error(errL)
	// We don't want os.Exit in a test
	defaultLogger.output(sFatal, 0, fatal)

	w.Close()
	os.Stderr = old

	var b bytes.Buffer
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		b.Write(scanner.Bytes())
	}

	out := b.String()

	for _, txt := range []string{info, warning, errL, fatal} {
		if !strings.Contains(out, txt) {
			t.Errorf("log output %q does not contain expected text: %q", out, txt)
		}
	}
}

func TestInit(t *testing.T) {
	var buf1 bytes.Buffer
	l1 := Init(3)
	l1.SetOutput(&buf1)
	if !reflect.DeepEqual(l1, defaultLogger) {
		t.Fatal("defaultLogger does not match logger returned by Init")
	}

	// Subsequent runs of Init shouldn't change defaultLogger.
	var buf2 bytes.Buffer
	l2 := Init(3)
	l2.SetOutput(&buf2)
	if !reflect.DeepEqual(l1, defaultLogger) {
		t.Error("defaultLogger should not have changed")
	}

	// Check log output.
	l1.Info("logger #1")
	l2.Info("logger #2")
	defaultLogger.Info("default logger")

	tests := []struct {
		out  string
		want int
	}{
		{buf1.String(), 2},
		{buf2.String(), 1},
	}

	for i, tt := range tests {
		got := len(strings.Split(strings.TrimSpace(tt.out), "\n"))
		if got != tt.want {
			t.Errorf("logger %d wrong number of lines, want %d, got %d", i+1, tt.want, got)
		}
	}
}

func TestLevel(t *testing.T) {
	initialize()
	var buf bytes.Buffer
	Init(1)
	SetOutput(&buf)

	Infof("info %d", sInfo)
	Warningf("warning %d", sWarning)
	Errorf("error %d", sError)
	s := buf.String()
	if strings.Contains(s, "info") {
		t.Errorf("log output %q should not contain: info", s)
	}
	if strings.Contains(s, "warning") {
		t.Errorf("log output %q should not contain: warning", s)
	}
	if !strings.Contains(s, "error") {
		t.Errorf("log output %q should contain: error", s)
	}
}

func TestFlags(t *testing.T) {
	initialize()
	l := Init(3)
	SetFlags(log.Llongfile)
	var buf bytes.Buffer
	SetOutput(&buf)
	l.Infof("info %d", sInfo)
	s := buf.String()
	if !strings.Contains(s, "info 1") {
		t.Errorf("log output %q should contain: info 1", s)
	}
	path := "logger/logger_test.go:117"
	if !strings.Contains(s, path) {
		t.Errorf("log output %q should contain: %s", s, path)
	}

	// bonus for coverage
	l.Warning("warning")
	l.Warningf("warning %d", sWarning)
	l.Error("error")
	l.Errorf("error %d", sError)
	s = buf.String()
	if !strings.Contains(s, "warning") {
		t.Errorf("log output %q should contain: warning", s)
	}
	if !strings.Contains(s, "warning 2") {
		t.Errorf("log output %q should contain: warning 2", s)
	}
	if !strings.Contains(s, "error") {
		t.Errorf("log output %q should contain: error", s)
	}
	if !strings.Contains(s, "error 3") {
		t.Errorf("log output %q should contain: error 3", s)
	}
}
