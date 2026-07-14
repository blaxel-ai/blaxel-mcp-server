package live

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	hostedToolsets            = "agents,modelapis,mcpservers,sandboxes,jobs,integrations,users,serviceaccounts,runtime"
	standaloneStderrTailLimit = 64 << 10
	standaloneStderrChunkSize = 32 << 10
)

type stderrDrainResult struct {
	tail    []byte
	leakErr error
	readErr error
}

type protectedSecrets struct {
	mu        sync.Mutex
	values    []string
	spool     *os.File
	spoolPath string
	leakErr   error
}

func newProtectedSecrets(values ...string) (*protectedSecrets, error) {
	spool, err := os.CreateTemp("", "blaxel-mcp-stderr-*")
	if err != nil {
		return nil, fmt.Errorf("create stderr spool: %w", err)
	}
	if err := spool.Chmod(0o600); err != nil {
		_ = spool.Close()
		_ = os.Remove(spool.Name())
		return nil, fmt.Errorf("secure stderr spool: %w", err)
	}
	return &protectedSecrets{
		values:    append([]string(nil), values...),
		spool:     spool,
		spoolPath: spool.Name(),
	}, nil
}

func (s *protectedSecrets) protect(secret string) {
	if len(secret) < 8 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = append(s.values, secret)
	if s.leakErr != nil {
		return
	}
	leaked, err := spoolContainsSecret(s.spool, []byte(secret))
	if err != nil {
		s.leakErr = fmt.Errorf("scan stderr spool: %w", err)
	} else if leaked {
		s.leakErr = fmt.Errorf("contains a configured credential value")
	}
}

func (s *protectedSecrets) appendAndScan(data, window []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := s.spool.Write(data)
	if err != nil {
		return fmt.Errorf("write stderr spool: %w", err)
	}
	if n != len(data) {
		return fmt.Errorf("write stderr spool: %w", io.ErrShortWrite)
	}
	if s.leakErr == nil {
		s.leakErr = scanForLeaks(string(window), s.values)
	}
	return nil
}

func (s *protectedSecrets) snapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.values...)
}

func (s *protectedSecrets) leakError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.leakErr
}

func (s *protectedSecrets) closeAndRemove() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	closeErr := s.spool.Close()
	removeErr := os.Remove(s.spoolPath)
	if closeErr != nil {
		return closeErr
	}
	return removeErr
}

func spoolContainsSecret(spool *os.File, secret []byte) (found bool, err error) {
	if _, err := spool.Seek(0, io.SeekStart); err != nil {
		return false, err
	}
	defer func() {
		if _, seekErr := spool.Seek(0, io.SeekEnd); err == nil && seekErr != nil {
			found = false
			err = seekErr
		}
	}()

	chunk := make([]byte, standaloneStderrChunkSize)
	var tail []byte
	for {
		n, err := spool.Read(chunk)
		if n > 0 {
			window := make([]byte, 0, len(tail)+n)
			window = append(window, tail...)
			window = append(window, chunk[:n]...)
			if bytes.Contains(window, secret) {
				return true, nil
			}
			tail = boundedByteTail(window, len(secret)-1)
		}
		if err != nil {
			if err == io.EOF {
				return false, nil
			}
			return false, err
		}
	}
}

type strictClient struct {
	t          *testing.T
	client     *mcpclient.Client
	ctx        context.Context
	cancel     context.CancelFunc
	stderrDone chan stderrDrainResult
	secrets    *protectedSecrets
	authHome   string

	callsMu         sync.Mutex
	successfulCalls map[string]int
}

func newStrictClient(t *testing.T) *strictClient {
	t.Helper()
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(root, "build", "blaxel-mcp-server")
	if info, err := os.Stat(binary); err != nil || info.IsDir() {
		t.Fatalf("built standalone binary missing at %s; run make build", binary)
	}

	// Isolate the process from CLI/browser credential stores. The live lane must
	// authenticate only with the workspace and API key supplied in process env.
	authHome := t.TempDir()
	env := environmentWithout("HOME", "XDG_CONFIG_HOME", "BL_CLIENT_ID", "BL_CLIENT_SECRET", "BLAXEL_CLIENT_ID", "BLAXEL_CLIENT_SECRET", "OPENAI_API_KEY", "ANTHROPIC_API_KEY")
	env = append(env,
		"HOME="+authHome,
		"XDG_CONFIG_HOME="+filepath.Join(authHome, ".config"),
		"BL_WORKSPACE="+liveConfig.Workspace,
		"BL_READ_ONLY=false",
		"BL_DEBUG=false",
	)
	secrets, err := newProtectedSecrets(credentialValues()...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := secrets.closeAndRemove(); err != nil {
			t.Errorf("clean up stderr spool: %v", err)
		}
	})
	c, err := mcpclient.NewStdioMCPClient(binary, env, "--toolsets", hostedToolsets)
	if err != nil {
		t.Fatalf("launch standalone MCP binary: %v", err)
	}

	stderrDone := make(chan stderrDrainResult, 1)
	if stderr, ok := mcpclient.GetStderr(c); ok {
		go func() {
			stderrDone <- drainStandaloneStderr(stderr, secrets)
		}()
	} else {
		stderrDone <- stderrDrainResult{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	_, err = c.Initialize(ctx, mcp.InitializeRequest{Params: mcp.InitializeParams{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		ClientInfo:      mcp.Implementation{Name: "blaxel-mcp-strict-live", Version: "1"},
	}})
	if err != nil {
		cancel()
		_ = c.Close()
		t.Fatalf("initialize official MCP client: %v", err)
	}

	sc := &strictClient{
		t: t, client: c, ctx: ctx, cancel: cancel, stderrDone: stderrDone,
		secrets: secrets, authHome: authHome, successfulCalls: make(map[string]int),
	}
	t.Cleanup(sc.close)
	return sc
}

func (c *strictClient) close() {
	c.cancel()
	_ = c.client.Close()
	select {
	case stderr := <-c.stderrDone:
		if stderr.readErr != nil {
			c.t.Errorf("standalone server stderr drain failed: %v", stderr.readErr)
		}
		if stderr.leakErr != nil {
			c.t.Errorf("standalone server stderr failed leak scan: %v", stderr.leakErr)
		} else if err := c.secrets.leakError(); err != nil {
			c.t.Errorf("standalone server stderr failed leak scan: %v", err)
		} else if len(stderr.tail) > 0 {
			// Scan the retained tail again for secrets registered after the bytes
			// were streamed, while keeping stderr memory use bounded.
			if err := scanForLeaks(string(stderr.tail), c.sensitiveValues()); err != nil {
				c.t.Errorf("standalone server stderr failed leak scan: %v", err)
			}
		}
	case <-time.After(2 * time.Second):
		c.t.Error("timed out draining standalone server stderr")
	}
	if err := scanDirectoryForSecrets(c.authHome, c.sensitiveValues()); err != nil {
		c.t.Errorf("standalone auth home persisted a secret: %v", err)
	}
}

func (c *strictClient) listTools() *mcp.ListToolsResult {
	c.t.Helper()
	result, err := c.client.ListTools(c.ctx, mcp.ListToolsRequest{})
	if err != nil {
		c.t.Fatalf("tools/list transport failure: %v", err)
	}
	return result
}

func (c *strictClient) callRaw(name string, args map[string]any) (*mcp.CallToolResult, string) {
	c.t.Helper()
	return c.callRawContext(c.ctx, name, args)
}

func (c *strictClient) callRawContext(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, string) {
	c.t.Helper()
	result, text, err := c.callRawResult(ctx, name, args)
	if err != nil {
		c.t.Fatalf("%s failed: %v", name, err)
	}
	return result, text
}

// callRawResult returns errors instead of failing the test so cleanup can keep
// unwinding every ledger entry after one transport, response, or leak failure.
func (c *strictClient) callRawResult(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, string, error) {
	result, err := c.client.CallTool(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Name: name, Arguments: args}})
	if err != nil {
		return nil, "", fmt.Errorf("transport failure: %s", redactedSummaryWithSecrets(err.Error(), c.sensitiveValues()))
	}
	if result == nil {
		return nil, "", fmt.Errorf("nil result")
	}
	text := resultText(result)
	if err := scanCallToolResult(result, c.sensitiveValues()); err != nil {
		return result, text, fmt.Errorf("result failed leak scan: %w", err)
	}
	if !result.IsError {
		c.callsMu.Lock()
		c.successfulCalls[name]++
		c.callsMu.Unlock()
	}
	return result, text, nil
}

func (c *strictClient) calledSuccessfully(name string) bool {
	c.callsMu.Lock()
	defer c.callsMu.Unlock()
	return c.successfulCalls[name] > 0
}

func drainStandaloneStderr(reader io.Reader, secrets *protectedSecrets) stderrDrainResult {
	var result stderrDrainResult
	chunk := make([]byte, standaloneStderrChunkSize)
	for {
		n, err := reader.Read(chunk)
		if n > 0 {
			// Include the previous tail in each scan so forbidden details split
			// across reads are still detected. Only the bounded tail is retained.
			window := make([]byte, 0, len(result.tail)+n)
			window = append(window, result.tail...)
			window = append(window, chunk[:n]...)
			if scanErr := secrets.appendAndScan(chunk[:n], window); scanErr != nil && result.readErr == nil {
				result.readErr = scanErr
			}
			result.tail = boundedByteTail(window, standaloneStderrTailLimit)
		}
		if err != nil {
			// Closing the MCP client closes its stderr pipe while the drain is
			// waiting. That is normal teardown, not a truncated-read failure.
			if err != io.EOF && !errors.Is(err, os.ErrClosed) && result.readErr == nil {
				result.readErr = err
			}
			result.leakErr = secrets.leakError()
			return result
		}
	}
}

func boundedByteTail(data []byte, limit int) []byte {
	if len(data) > limit {
		data = data[len(data)-limit:]
	}
	return append([]byte(nil), data...)
}

func scanCallToolResult(result *mcp.CallToolResult, secrets []string) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode full result for leak scan: %w", err)
	}
	return scanForLeaks(resultText(result)+"\n"+string(raw), secrets)
}

func (c *strictClient) protectSecret(secret string) {
	c.secrets.protect(secret)
}

func (c *strictClient) sensitiveValues() []string {
	return c.secrets.snapshot()
}

func (c *strictClient) call(name string, args map[string]any) string {
	c.t.Helper()
	result, text := c.callRaw(name, args)
	if result == nil {
		c.t.Fatalf("%s returned nil result", name)
	}
	if result.IsError {
		c.t.Fatalf("%s returned tool error: %s", name, redactedSummary(text))
	}
	return text
}

func resultText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	var parts []string
	for _, content := range result.Content {
		if text, ok := mcp.AsTextContent(content); ok {
			parts = append(parts, text.Text)
			continue
		}
		if data, err := json.Marshal(content); err == nil {
			parts = append(parts, string(data))
		}
	}
	return strings.Join(parts, "\n")
}

func scanForLeaks(text string, credentials []string) error {
	for _, pattern := range []*regexp.Regexp{
		regexp.MustCompile(`(?i)traceback \(most recent call last\)`),
		regexp.MustCompile(`(?i)runtime/debug\.stack`),
		regexp.MustCompile(`(?i)/go/pkg/mod/`),
		regexp.MustCompile(`(?i)panic:`),
		regexp.MustCompile(`(?m)goroutine [0-9]+ \[`),
		regexp.MustCompile(`(?i)\.go:\d+`),
		regexp.MustCompile(`(?i)github\.com/blaxel-ai/controlplane`),
		regexp.MustCompile(`(?m)\t+at\s+`),
		regexp.MustCompile(`(?i)arn:(aws|aws-us-gov|aws-cn):`),
		regexp.MustCompile(`(?i)\b(aws|amazon)[ ._/-]?(ses|simple email service)\b`),
		regexp.MustCompile(`(?i)\bses:\s*sendemail\b|\bsimple email service\b`),
		regexp.MustCompile(`(?i)\b(request|message)[ _-]?id\b\s*[:=]`),
		regexp.MustCompile(`(?i)\bx-amzn-(requestid|trace-id)\b`),
		regexp.MustCompile(`(?i)<!doctype\s+html|<html(\s|>)`),
	} {
		if pattern.MatchString(text) {
			return fmt.Errorf("contains forbidden internal detail matching %q", pattern.String())
		}
	}
	for _, secret := range credentials {
		if len(secret) >= 8 && strings.Contains(text, secret) {
			return fmt.Errorf("contains a configured credential value")
		}
	}
	return nil
}

func environmentWithout(keys ...string) []string {
	excluded := make(map[string]bool, len(keys))
	for _, key := range keys {
		excluded[key] = true
	}
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !excluded[key] {
			env = append(env, entry)
		}
	}
	return env
}

func scanDirectoryForSecrets(root string, secrets []string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, secret := range secrets {
			if len(secret) >= 8 && bytes.Contains(data, []byte(secret)) {
				return fmt.Errorf("credential material found in %s", filepath.Base(path))
			}
		}
		return nil
	})
}

func credentialValues() []string {
	keys := []string{"BL_API_KEY", "BL_CLIENT_SECRET", "BLAXEL_CLIENT_SECRET", "OPENAI_API_KEY", "ANTHROPIC_API_KEY"}
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func redactedSummary(text string) string {
	return redactedSummaryWithSecrets(text, credentialValues())
}

func redactedSummaryWithSecrets(text string, secrets []string) string {
	text = strings.TrimSpace(text)
	for _, secret := range secrets {
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "[REDACTED]")
		}
	}
	if text == "" {
		return "empty error result"
	}
	if len(text) > 240 {
		text = text[:240] + "…"
	}
	return text
}

func TestPM2611APIKeyWorkspaceAuthentication(t *testing.T) {
	if os.Getenv("BLAXEL_E2E_REVIEWER_API_KEY_CONFIRMED") != "true" {
		t.Fatal("PM-2611 reviewer lane requires BLAXEL_E2E_REVIEWER_API_KEY_CONFIRMED=true to confirm BL_API_KEY is an API key, not a developer token")
	}
	client := newStrictClient(t)
	if len(client.listTools().Tools) != 41 {
		t.Fatal("API-key-authenticated workspace session did not expose the hosted tool contract")
	}
}

func newTestProtectedSecrets(t *testing.T, values ...string) *protectedSecrets {
	t.Helper()
	secrets, err := newProtectedSecrets(values...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := secrets.closeAndRemove(); err != nil {
			t.Errorf("clean up stderr spool: %v", err)
		}
	})
	return secrets
}

func TestStandaloneStderrDrainScansBeyondFirstMiBAndBoundsRetention(t *testing.T) {
	credential := "credential-value-123"
	payload := bytes.Repeat([]byte("safe stderr output\n"), (1<<20)/19+100)
	payload = append(payload, credential...)
	// Move the leak outside the retained tail to prove the streaming scan, not
	// only the final bounded-tail scan, remembers it.
	payload = append(payload, bytes.Repeat([]byte("safe trailing output\n"), standaloneStderrTailLimit/21+100)...)
	reader := bytes.NewReader(payload)

	result := drainStandaloneStderr(reader, newTestProtectedSecrets(t, credential))

	if result.readErr != nil {
		t.Fatalf("drainStandaloneStderr() read error = %v", result.readErr)
	}
	if reader.Len() != 0 {
		t.Fatalf("drainStandaloneStderr() left %d bytes unread", reader.Len())
	}
	if result.leakErr == nil {
		t.Fatal("drainStandaloneStderr() missed forbidden output beyond the first MiB")
	}
	if len(result.tail) > standaloneStderrTailLimit {
		t.Fatalf("retained stderr tail = %d bytes, want at most %d", len(result.tail), standaloneStderrTailLimit)
	}
}

func TestStandaloneStderrDrainRetrospectivelyScansSecret(t *testing.T) {
	credential := "late-credential-value-123"
	secrets := newTestProtectedSecrets(t)
	info, err := secrets.spool.Stat()
	if err != nil {
		t.Fatalf("stat stderr spool: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("stderr spool permissions = %o, want 600", got)
	}
	reader, writer := io.Pipe()
	defer func() { _ = reader.Close() }()
	defer func() { _ = writer.Close() }()

	done := make(chan stderrDrainResult, 1)
	go func() {
		done <- drainStandaloneStderr(reader, secrets)
	}()

	prefix := bytes.Repeat([]byte("safe stderr output\n"), (1<<20)/19+100)
	if _, err := writer.Write(prefix); err != nil {
		t.Fatalf("write stderr prefix: %v", err)
	}
	if _, err := io.WriteString(writer, credential); err != nil {
		t.Fatalf("write unregistered credential: %v", err)
	}
	trailing := bytes.Repeat([]byte("safe trailing output\n"), standaloneStderrTailLimit/21+100)
	if _, err := writer.Write(trailing); err != nil {
		t.Fatalf("write trailing stderr: %v", err)
	}
	secrets.protect(credential)
	if err := writer.Close(); err != nil {
		t.Fatalf("close stderr writer: %v", err)
	}

	result := <-done
	if result.readErr != nil {
		t.Fatalf("drainStandaloneStderr() read error = %v", result.readErr)
	}
	if result.leakErr == nil {
		t.Fatal("drainStandaloneStderr() missed a secret registered after it left the retained tail")
	}
	if bytes.Contains(result.tail, []byte(credential)) {
		t.Fatal("test credential remains in retained tail; regression did not exercise streaming scan")
	}
	if len(result.tail) > standaloneStderrTailLimit {
		t.Fatalf("retained stderr tail = %d bytes, want at most %d", len(result.tail), standaloneStderrTailLimit)
	}
}

func TestLeakScannerRegression(t *testing.T) {
	credential := "credential-value-123"
	for _, leaked := range []string{
		"handler.go:42", "github.com/blaxel-ai/controlplane/api", "\tat handler",
		"arn:aws:iam::123456789012:role/test", "AWS SES request failed", "SES: SendEmail",
		"RequestId: abc-123", "x-amzn-requestid: abc-123", "<!doctype html><html>", credential,
	} {
		t.Run(leaked, func(t *testing.T) {
			if err := scanForLeaks(leaked, []string{credential}); err == nil {
				t.Fatal("leak scanner accepted forbidden output")
			}
		})
	}
	if err := scanForLeaks("Internal Server Error", []string{credential}); err != nil {
		t.Fatalf("generic safe error text must not be classified as a leak: %v", err)
	}
	structured := mcp.NewToolResultText("safe rendered content")
	structured.StructuredContent = map[string]any{"credential": credential}
	if err := scanCallToolResult(structured, []string{credential}); err == nil {
		t.Fatal("full-result scanner missed a credential in structuredContent")
	}
}

func parseJSONObject(t *testing.T, text string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewBufferString(text))
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("expected JSON object result: %v", err)
	}
	return value
}

func findNestedString(value map[string]any, keys ...string) string {
	var current any = value
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = object[key]
	}
	result, _ := current.(string)
	return result
}
