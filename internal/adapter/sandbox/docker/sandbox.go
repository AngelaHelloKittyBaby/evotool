// Package docker runs untrusted Python in disposable Linux containers.
package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/AngelaHelloKittyBaby/evotool/internal/ports"
)

// Sandbox requires a locally available image; it never pulls implicitly.
type Sandbox struct{ Image string }

// Run transfers source over stdin without mounting host directories or secrets.
func (s Sandbox) Run(ctx context.Context, r ports.SandboxRequest) (ports.SandboxResult, error) {
	if s.Image == "" || r.Runtime != "python" || len(r.Command) != 1 {
		return ports.SandboxResult{}, fmt.Errorf("image, python runtime and one entry point required")
	}
	l := r.Policy.ResourceLimits
	if l.MaxRuntimeMs <= 0 || l.MaxRuntimeMs > 30000 || l.MaxMemoryBytes < 16777216 || l.MaxMemoryBytes > 268435456 || l.MaxOutputBytes <= 0 || l.MaxOutputBytes > 1048576 {
		return ports.SandboxResult{}, fmt.Errorf("invalid sandbox resource limits")
	}
	p := r.Policy.Permissions
	if (p.Network != "" && p.Network != "none") || len(p.AllowedPaths)+len(p.AllowedCommands)+len(p.Environment) > 0 {
		return ports.SandboxResult{}, fmt.Errorf("privileged permissions unsupported")
	}
	payload, err := json.Marshal(map[string]any{"files": r.Files, "entry": r.Command[0], "input": r.Input})
	if err != nil {
		return ports.SandboxResult{}, err
	}
	if len(payload) > 1048576 {
		return ports.SandboxResult{}, fmt.Errorf("package and input exceed 1 MiB")
	}
	deadline, cancel := context.WithTimeout(ctx, time.Duration(l.MaxRuntimeMs)*time.Millisecond)
	defer cancel()
	// A unique name lets cleanup remove the container even after client timeout.
	name := "evotool-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = exec.CommandContext(cleanup, "docker", "rm", "-f", name).Run()
	}()
	args := []string{"run", "--rm", "--pull=never", "--name", name, "--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--pids-limit=32", "--cpus=1", "--memory=" + strconv.FormatInt(l.MaxMemoryBytes, 10), "--memory-swap=" + strconv.FormatInt(l.MaxMemoryBytes, 10), "--user=65534:65534", "--tmpfs=/work:rw,noexec,nosuid,size=16m,mode=1777", "-i", s.Image, "python", "-I", "-B", "-c", bootstrap}
	cmd := exec.CommandContext(deadline, "docker", args...)
	cmd.Stdin = bytes.NewReader(payload)
	out := &limitedBuffer{limit: int(l.MaxOutputBytes)}
	stderr := &limitedBuffer{limit: int(l.MaxOutputBytes)}
	cmd.Stdout = out
	cmd.Stderr = stderr
	err = cmd.Run()
	result := ports.SandboxResult{Success: err == nil, Output: out.String(), Error: stderr.String(), Metadata: map[string]string{"isolation": "docker", "image": s.Image}}
	if deadline.Err() != nil {
		return result, fmt.Errorf("sandbox deadline: %w", deadline.Err())
	}
	if out.exceeded || stderr.exceeded {
		result.Success = false
		return result, fmt.Errorf("sandbox output limit exceeded")
	}
	if err != nil {
		return result, fmt.Errorf("sandbox failed: %w: %s", err, stderr.String())
	}
	return result, nil
}

type limitedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	exceeded bool
}

func (b *limitedBuffer) String() string { return b.buffer.String() }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := b.limit - b.buffer.Len()
	if len(p) > remaining {
		b.exceeded = true
		p = p[:remaining]
	}
	_, _ = b.buffer.Write(p)
	return n, nil
}

const bootstrap = `import json, pathlib, sys, os, runpy, io
p=json.load(sys.stdin)
os.chdir('/work')
for f in p['files']:
 path=pathlib.PurePosixPath(f['Path'])
 if path.is_absolute() or '..' in path.parts or '\\' in f['Path']:
  raise ValueError('unsafe source path')
 target=pathlib.Path(*path.parts)
 target.parent.mkdir(parents=True,exist_ok=True)
 target.write_text(f['Content'],encoding='utf-8')
sys.path.insert(0,'/work')
sys.stdin=io.StringIO(json.dumps(p['input']))
runpy.run_path(p['entry'],run_name='__main__')
`
