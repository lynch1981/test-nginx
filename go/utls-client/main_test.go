package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestResolveClientAliases(t *testing.T) {
	cases := map[string]string{
		"chrome":          "HelloChrome_Auto",
		"Chrome":          "HelloChrome_Auto",
		"firefox":         "HelloFirefox_Auto",
		"safari":          "HelloSafari_Auto",
		"ios":             "HelloIOS_Auto",
		"android":         "HelloAndroid_11_OkHttp",
		"edge":            "HelloEdge_Auto",
		"randomized":      "HelloRandomizedALPN",
		"golang":          "HelloGolang",
		"HelloChrome_120": "HelloChrome_120",
		"hellochrome_120": "HelloChrome_120",
		"Chrome_120":      "HelloChrome_120",
	}
	for in, want := range cases {
		id, err := resolveClient(in)
		if err != nil {
			t.Errorf("resolveClient(%q): %v", in, err)
			continue
		}
		got, err := resolveClient(want)
		if err != nil {
			t.Errorf("resolveClient(%q): %v", want, err)
			continue
		}
		if id.Client != got.Client || id.Version != got.Version {
			t.Errorf("resolveClient(%q) = %+v, want %+v", in, id, got)
		}
	}
}

func TestResolveClientUnknown(t *testing.T) {
	_, err := resolveClient("not-a-browser")
	if err == nil {
		t.Fatal("expected error for unknown client")
	}
	if !strings.Contains(err.Error(), "unknown client") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "aliases:") {
		t.Fatalf("error should list clients: %v", err)
	}
}

func TestSplitALPN(t *testing.T) {
	got := splitALPN("h2, http/1.1")
	if len(got) != 2 || got[0] != "h2" || got[1] != "http/1.1" {
		t.Fatalf("splitALPN = %#v", got)
	}
	if splitALPN("  ") != nil {
		t.Fatal("expected nil for blank ALPN")
	}
}

func TestListClients(t *testing.T) {
	text := listClientsText()
	for _, want := range []string{"chrome = HelloChrome_Auto", "HelloChrome_Auto", "HelloGolang"} {
		if !strings.Contains(text, want) {
			t.Errorf("listClientsText missing %q", want)
		}
	}
}

func TestDoRequestHTTP1(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/t" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("X-Foo") != "bar" {
			t.Errorf("X-Foo = %q", r.Header.Get("X-Foo"))
		}
		w.Header().Set("X-Reply", "ok")
		w.WriteHeader(200)
		_, _ = io.WriteString(w, "hello utls")
	}))
	defer ts.Close()

	host := ts.Listener.Addr().String()
	raw := []byte("GET /t HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nX-Foo: bar\r\n\r\n")
	out, err := doRequest(options{
		client:    "golang",
		addr:      host,
		sni:       "localhost",
		insecure:  true,
		timeout:   5e9,
		http2Mode: "never",
	}, raw)
	if err != nil {
		t.Fatal(err)
	}
	body := string(out)
	if !strings.Contains(body, "200") {
		t.Fatalf("missing 200: %s", body)
	}
	if !strings.Contains(body, "hello utls") {
		t.Fatalf("missing body: %s", body)
	}
	if !strings.Contains(body, "X-Reply") {
		t.Fatalf("missing header: %s", body)
	}
}

func TestDoRequestChromeHandshake(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	defer ts.Close()

	raw := []byte("GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")
	out, err := doRequest(options{
		client:    "chrome",
		addr:      ts.Listener.Addr().String(),
		sni:       "localhost",
		insecure:  true,
		timeout:   5e9,
		http2Mode: "auto",
	}, raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "ok") {
		t.Fatalf("response = %s", out)
	}
}

func TestDoRequestHTTP2(t *testing.T) {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			t.Errorf("expected HTTP/2, got %s", r.Proto)
		}
		w.Header().Set("X-Proto", r.Proto)
		_, _ = io.WriteString(w, "h2-ok")
	}))
	ts.EnableHTTP2 = true
	ts.TLS = &tls.Config{NextProtos: []string{"h2", "http/1.1"}}
	ts.StartTLS()
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	raw := []byte("GET /t HTTP/1.1\r\nHost: localhost\r\n\r\n")
	out, err := doRequest(options{
		client:    "golang",
		addr:      u.Host,
		sni:       "localhost",
		insecure:  true,
		timeout:   5e9,
		http2Mode: "require",
	}, raw)
	if err != nil {
		t.Fatal(err)
	}
	body := string(out)
	if !strings.Contains(body, "HTTP/1.1") {
		t.Fatalf("expected HTTP/1.1 dump: %s", body)
	}
	if !strings.Contains(body, "h2-ok") {
		t.Fatalf("missing body: %s", body)
	}
}

func TestRunCLIListClients(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"--list-clients"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "chrome = HelloChrome_Auto") {
		t.Fatalf("stdout=%s", stdout.String())
	}
}

func TestRunCLIRequiresAddr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"--client", "chrome"}, strings.NewReader("GET / HTTP/1.1\r\n\r\n"), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--addr") {
		t.Fatalf("stderr=%s", stderr.String())
	}
}

func TestRunCLIE2E(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "cli-ok")
	}))
	defer ts.Close()

	var stdout, stderr bytes.Buffer
	req := "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	code := runCLI([]string{
		"--client", "golang",
		"--addr", ts.Listener.Addr().String(),
		"--sni", "localhost",
		"--insecure",
		"--http2", "never",
		"--timeout", "5",
	}, strings.NewReader(req), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "cli-ok") {
		t.Fatalf("stdout=%s", stdout.String())
	}
}
