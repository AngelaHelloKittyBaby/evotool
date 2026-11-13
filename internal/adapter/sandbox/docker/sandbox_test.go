package docker

import (
	"context"
	"github.com/AngelaHelloKittyBaby/evotool/internal/domain"
	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
	"io"
	"os"
	"strings"
	"testing"
)

func TestLimitedBuffer(t *testing.T) {
	b := &limitedBuffer{limit: 3}
	n, err := b.Write([]byte("abcdef"))
	if n != 6 || err != nil || b.String() != "abc" || !b.exceeded {
		t.Fatalf("buffer=%+v n=%d err=%v", b, n, err)
	}
}

func TestLimitedBufferCopyCannotBypassLimit(t *testing.T) {
	b := &limitedBuffer{limit: 3}
	_, err := io.Copy(b, io.LimitReader(strings.NewReader("abcdef"), 6))
	if err != nil || b.String() != "abc" || !b.exceeded {
		t.Fatalf("output=%q exceeded=%v err=%v", b.String(), b.exceeded, err)
	}
}

func TestSandboxRejectsInvalidRequest(t *testing.T) {
	if _, err := (Sandbox{Image: "python:3.13-alpine"}).Run(context.Background(), ports.SandboxRequest{Runtime: "shell"}); err == nil {
		t.Fatal("unsupported runtime accepted")
	}
}

func TestDockerIsolation(t *testing.T) {
	if os.Getenv("EVOTOOL_DOCKER_TEST") != "1" {
		t.Skip("optional: set EVOTOOL_DOCKER_TEST=1 and prepare python:3.13-alpine")
	}
	for _, tc := range []struct {
		name, code string
		fail       bool
		output     int64
	}{
		{"isolation", "import os, socket\nassert os.getuid() == 65534\nassert 'EVOTOOL_HOST_SECRET' not in os.environ\ntry:\n open('/etc/evotool-write-test','w')\n raise AssertionError('root filesystem writable')\nexcept OSError:\n pass\ns=socket.socket(); s.settimeout(1)\ntry:\n s.connect(('1.1.1.1',443))\n raise AssertionError('network available')\nexcept OSError:\n pass\nprint('isolated')", false, 4096},
		{"timeout", "while True: pass", true, 4096},
		{"output limit", "print('x'*10000)", true, 100},
		{"test failure", "assert False", true, 4096},
		{"memory limit", "x = bytearray(256 * 1024 * 1024)", true, 4096},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limit := int64(30000)
			if tc.name == "timeout" {
				limit = 5000
			}
			r, err := (Sandbox{Image: "python:3.13-alpine"}).Run(context.Background(), ports.SandboxRequest{Runtime: "python", Files: []domain.SourceFile{{Path: "main.py", Content: tc.code}}, Command: []string{"main.py"}, Policy: domain.ToolPolicy{ResourceLimits: domain.ResourceLimits{MaxRuntimeMs: limit, MaxMemoryBytes: 67108864, MaxOutputBytes: tc.output}}})
			if tc.fail {
				if err == nil {
					t.Fatal("unsafe/failed run succeeded")
				}
				if tc.name != "timeout" && strings.Contains(err.Error(), "deadline") {
					t.Fatalf("unexpected infrastructure timeout: %v", err)
				}
				if tc.name == "timeout" && !strings.Contains(err.Error(), "deadline") {
					t.Fatalf("expected timeout, got %v", err)
				}
				if tc.name == "output limit" && !strings.Contains(err.Error(), "output limit exceeded") {
					t.Fatalf("expected output rejection, got %v", err)
				}
			} else if err != nil || !strings.Contains(r.Output, "isolated") {
				t.Fatalf("result=%+v err=%v", r, err)
			}
		})
	}
}
