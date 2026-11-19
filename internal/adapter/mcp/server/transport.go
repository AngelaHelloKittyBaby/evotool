package server

import (
	"bufio"
	"context"
	"io"
	"net/http"
)

// ServeStdio serves newline-delimited JSON-RPC requests until input ends.
// It is suitable for MCP clients that launch EvoTool as a child process.
func (s Server) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	// Generated manifests can be sizeable; allow up to 2 MiB per request.
	scanner.Buffer(make([]byte, 4096), 2*1024*1024)
	writer := bufio.NewWriter(out)
	defer writer.Flush()
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := writer.Write(append(s.Handle(ctx, scanner.Bytes()), '\n')); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// ServeHTTP exposes one JSON-RPC request per POST. GET is intentionally not
// supported so clients cannot accidentally invoke tools from a browser.
func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if r.ContentLength > 2*1024*1024 {
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 2*1024*1024+1))
	if err != nil {
		http.Error(w, "read request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(body) > 2*1024*1024 {
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(s.Handle(r.Context(), body))
}
