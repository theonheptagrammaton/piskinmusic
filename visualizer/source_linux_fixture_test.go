//go:build linux

package visualizer

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jfreymuth/pulse"
)

// Isolated protocol fixture with null sinks and silence, not hardware audio or
// subscription evidence. Only the daemon created here is killed.
func TestPulseIsolatedLifecycle(t *testing.T) {
	if os.Getenv("AMTUI_TEST_PULSE_FIXTURE") != "1" {
		t.Skip("set AMTUI_TEST_PULSE_FIXTURE=1 for isolated PipeWire lifecycle checks")
	}
	for _, binary := range []string{"pipewire", "wireplumber", "pactl"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Fatalf("opt-in fixture requires %s: %v", binary, err)
		}
	}
	server := startPulseFixture(t)
	service := fixtureService(t)
	first := fixtureFrame(t, service, time.Time{})
	assertFixtureMonitor(t, "amtui_test_a.monitor")

	fixtureSetDefault(t, "amtui_test_b")
	// Existing captures select a monitor at open time. Reopening must select
	// the new default instead of silently reusing the previous source.
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	service = fixtureService(t)
	fixtureFrame(t, service, first.At)
	assertFixtureMonitor(t, "amtui_test_b.monitor")
	t.Log("reopen selected the changed default sink monitor")

	fixturePactl(t, "unload-module", server.moduleB)
	// Give policy time to move the stream or let it become stale. Both are
	// observable; the UI's delayed-read regression covers the STALE boundary.
	time.Sleep(600 * time.Millisecond)
	frame, _ := service.Latest()
	t.Logf("removed monitor: frame_age=%s terminal=%v", time.Since(frame.At), service.Err())
	if service.Err() == nil && time.Since(frame.At) <= 400*time.Millisecond {
		assertFixtureMonitor(t, "amtui_test_a.monitor")
		t.Log("policy moved capture to the surviving monitor")
	}
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	fixtureSetDefault(t, "amtui_test_a")
	service = fixtureService(t)
	fixtureFrame(t, service, time.Time{})
	assertFixtureMonitor(t, "amtui_test_a.monitor")

	if err := server.process.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(6 * time.Second)
	for service.Err() == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if service.Err() == nil {
		t.Fatal("server loss did not terminate capture")
	}
	t.Logf("isolated server loss surfaced: %v", service.Err())
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	startPulseFixture(t)
	service = fixtureService(t)
	fixtureFrame(t, service, time.Time{})
	assertFixtureMonitor(t, "amtui_test_a.monitor")
	t.Log("capture reopened on a fresh isolated server")
}

type pulseFixture struct {
	process *exec.Cmd
	moduleB string
}

func startPulseFixture(t *testing.T) pulseFixture {
	t.Helper()
	dir := t.TempDir()
	socket := filepath.Join(dir, "pulse")
	t.Setenv("PIPEWIRE_RUNTIME_DIR", dir)
	t.Setenv("PIPEWIRE_REMOTE", "fixture")
	t.Setenv("PULSE_SERVER", "unix:"+socket)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(dir, "state"))
	config, err := os.ReadFile("testdata/pulse-fixture.conf")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "fixture.conf")
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(string(config), "@PULSE_SOCKET@", socket)), 0o600); err != nil {
		t.Fatal(err)
	}
	start := func(binary string, args ...string) *exec.Cmd {
		cmd := exec.Command(binary, args...)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		})
		return cmd
	}
	process := start("pipewire", "-c", path)
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(socket); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("isolated Pulse socket did not appear")
		}
		time.Sleep(10 * time.Millisecond)
	}
	start("wireplumber", "-p", "policy") // policy only: no hardware discovery
	fixturePactl(t, "load-module", "module-null-sink", "sink_name=amtui_test_a")
	moduleB := fixturePactl(t, "load-module", "module-null-sink", "sink_name=amtui_test_b")
	deadline = time.Now().Add(3 * time.Second)
	for {
		sinks := fixturePactl(t, "list", "short", "sinks")
		if strings.Contains(sinks, "amtui_test_a") && strings.Contains(sinks, "amtui_test_b") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("isolated sink policy did not become ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	fixtureSetDefault(t, "amtui_test_a")

	return pulseFixture{process: process, moduleB: moduleB}
}

func fixtureSetDefault(t *testing.T, name string) {
	t.Helper()
	fixturePactl(t, "set-default-sink", name)
	deadline := time.Now().Add(3 * time.Second)
	for {
		client, err := pulse.NewClient(pulse.ClientTimeout(time.Second))
		var sink *pulse.Sink
		if err == nil {
			sink, err = client.DefaultSink()
			client.Close()
		}
		if err == nil && sink.ID() == name {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("isolated default sink %s did not become ready: %v", name, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func fixturePactl(t *testing.T, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "pactl", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("isolated pactl %v: %v: %s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func fixtureService(t *testing.T) *Service {
	t.Helper()
	source, err := OpenSystemSource()
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(source)
	if err != nil {
		_ = source.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := service.Close(); err != nil {
			t.Errorf("fixture service close: %v", err)
		}
	})
	return service
}

func fixtureFrame(t *testing.T, service *Service, after time.Time) Frame {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if err := service.Err(); err != nil {
			t.Fatal(err)
		}
		if frame, ok := service.Latest(); ok && frame.Live && frame.At.After(after) {
			return frame
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no fresh isolated capture frame")
	return Frame{}
}

func assertFixtureMonitor(t *testing.T, name string) {
	t.Helper()
	var sources []struct {
		Index int
		Name  string
	}
	var outputs []struct {
		Source     int
		Properties map[string]string
	}
	if err := json.Unmarshal([]byte(fixturePactl(t, "-f", "json", "list", "sources")), &sources); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(fixturePactl(t, "-f", "json", "list", "source-outputs")), &outputs); err != nil {
		t.Fatal(err)
	}
	for _, source := range sources {
		for _, output := range outputs {
			if source.Name == name && output.Properties["application.name"] == "amtui" && output.Source == source.Index {
				return
			}
		}
	}
	t.Fatalf("capture is not attached to isolated monitor %s", name)
}
