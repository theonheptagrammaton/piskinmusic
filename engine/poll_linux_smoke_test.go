//go:build linux

package engine

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// Live headed browser in a disposable profile. Never signs out the user's
// persistent profile or enters credentials; separate from subscription proof.
func TestLinuxHeadedLoginCancellationLive(t *testing.T) {
	if os.Getenv("AMTUI_TEST_LINUX_BROWSER") != "1" {
		t.Skip("set AMTUI_TEST_LINUX_BROWSER=1 for disposable headed browser checks")
	}
	browserCtx, cancels, err := open(t.TempDir(), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeAll(cancels) })
	ctx, cancel := context.WithTimeout(browserCtx, 30*time.Second)
	defer cancel()
	if err := poll(ctx, mkReady, 25*time.Second, "MusicKit"); err != nil {
		t.Fatal(err)
	}
	var authorized bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(mkAuthorized, &authorized)); err != nil {
		t.Fatal(err)
	}
	if authorized {
		t.Fatal("disposable profile unexpectedly authorized")
	}
	started := time.Now()
	err = poll(ctx, "false", 150*time.Millisecond, "test login timeout")
	if err == nil || !strings.Contains(err.Error(), "timed out waiting for test login timeout") || time.Since(started) > time.Second {
		t.Fatalf("bounded login timeout: elapsed=%s err=%v", time.Since(started), err)
	}
	if h, ok := newWindowController(browserPID(ctx)).(hyprlandWindowController); ok {
		if err := h.parkOffscreen(ctx); err != nil {
			t.Fatal(err)
		}
		if parked, known := h.parked(ctx); !known || !parked {
			t.Fatal("headed test browser was not verified on special:amtui")
		}
		t.Log("disposable headed browser verified on special:amtui")
	}
	result := make(chan error, 1)
	go func() { result <- poll(ctx, mkAuthorized, 5*time.Minute, "login") }()
	process := chromedp.FromContext(ctx).Browser.Process()
	if process == nil {
		t.Fatal("missing owned browser process")
	}
	started = time.Now()
	if err := process.Kill(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("login poll after browser exit: %v", err)
		}
		t.Logf("owned browser exit stopped pending login in %s", time.Since(started))
	case <-time.After(2 * time.Second):
		t.Fatal("login poll did not stop after browser exit")
	}
}
