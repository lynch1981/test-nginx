// Command test-nginx-utls is a TLS HTTP client that parrots browser
// ClientHello fingerprints via refraction-networking/utls.
//
// It reads a raw HTTP/1.1 request from stdin, dials --addr with the
// chosen fingerprint, and writes a curl -i style HTTP/1.1 dump to stdout
// so Test::Nginx::Socket can parse the response unchanged.
package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"sort"
	"strings"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

var helloIDs = map[string]utls.ClientHelloID{
	"HelloGolang":                      utls.HelloGolang,
	"HelloCustom":                      utls.HelloCustom,
	"HelloRandomized":                  utls.HelloRandomized,
	"HelloRandomizedALPN":              utls.HelloRandomizedALPN,
	"HelloRandomizedNoALPN":            utls.HelloRandomizedNoALPN,
	"HelloFirefox_Auto":                utls.HelloFirefox_Auto,
	"HelloFirefox_55":                  utls.HelloFirefox_55,
	"HelloFirefox_56":                  utls.HelloFirefox_56,
	"HelloFirefox_63":                  utls.HelloFirefox_63,
	"HelloFirefox_65":                  utls.HelloFirefox_65,
	"HelloFirefox_99":                  utls.HelloFirefox_99,
	"HelloFirefox_102":                 utls.HelloFirefox_102,
	"HelloFirefox_105":                 utls.HelloFirefox_105,
	"HelloFirefox_120":                 utls.HelloFirefox_120,
	"HelloChrome_Auto":                 utls.HelloChrome_Auto,
	"HelloChrome_58":                   utls.HelloChrome_58,
	"HelloChrome_62":                   utls.HelloChrome_62,
	"HelloChrome_70":                   utls.HelloChrome_70,
	"HelloChrome_72":                   utls.HelloChrome_72,
	"HelloChrome_83":                   utls.HelloChrome_83,
	"HelloChrome_87":                   utls.HelloChrome_87,
	"HelloChrome_96":                   utls.HelloChrome_96,
	"HelloChrome_100":                  utls.HelloChrome_100,
	"HelloChrome_102":                  utls.HelloChrome_102,
	"HelloChrome_106_Shuffle":          utls.HelloChrome_106_Shuffle,
	"HelloChrome_100_PSK":              utls.HelloChrome_100_PSK,
	"HelloChrome_112_PSK_Shuf":         utls.HelloChrome_112_PSK_Shuf,
	"HelloChrome_114_Padding_PSK_Shuf": utls.HelloChrome_114_Padding_PSK_Shuf,
	"HelloChrome_115_PQ":               utls.HelloChrome_115_PQ,
	"HelloChrome_115_PQ_PSK":           utls.HelloChrome_115_PQ_PSK,
	"HelloChrome_120":                  utls.HelloChrome_120,
	"HelloChrome_120_PQ":               utls.HelloChrome_120_PQ,
	"HelloChrome_131":                  utls.HelloChrome_131,
	"HelloChrome_133":                  utls.HelloChrome_133,
	"HelloIOS_Auto":                    utls.HelloIOS_Auto,
	"HelloIOS_11_1":                    utls.HelloIOS_11_1,
	"HelloIOS_12_1":                    utls.HelloIOS_12_1,
	"HelloIOS_13":                      utls.HelloIOS_13,
	"HelloIOS_14":                      utls.HelloIOS_14,
	"HelloAndroid_11_OkHttp":           utls.HelloAndroid_11_OkHttp,
	"HelloEdge_Auto":                   utls.HelloEdge_Auto,
	"HelloEdge_85":                     utls.HelloEdge_85,
	"HelloEdge_106":                    utls.HelloEdge_106,
	"HelloSafari_Auto":                 utls.HelloSafari_Auto,
	"HelloSafari_16_0":                 utls.HelloSafari_16_0,
	"Hello360_Auto":                    utls.Hello360_Auto,
	"Hello360_7_5":                     utls.Hello360_7_5,
	"Hello360_11_0":                    utls.Hello360_11_0,
	"HelloQQ_Auto":                     utls.HelloQQ_Auto,
	"HelloQQ_11_1":                     utls.HelloQQ_11_1,
}

var aliases = map[string]string{
	"chrome":     "HelloChrome_Auto",
	"firefox":    "HelloFirefox_Auto",
	"safari":     "HelloSafari_Auto",
	"ios":        "HelloIOS_Auto",
	"android":    "HelloAndroid_11_OkHttp",
	"edge":       "HelloEdge_Auto",
	"randomized": "HelloRandomizedALPN",
	"golang":     "HelloGolang",
}

type options struct {
	client    string
	addr      string
	sni       string
	insecure  bool
	timeout   time.Duration
	alpn      []string
	http2Mode string
	verbose   bool
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func runCLI(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("test-nginx-utls", flag.ContinueOnError)
	fs.SetOutput(stderr)

	client := fs.String("client", "chrome", "fingerprint alias or Hello* ClientHelloID")
	addr := fs.String("addr", "", "host:port to dial")
	sni := fs.String("sni", "localhost", "TLS server name")
	insecure := fs.Bool("insecure", false, "skip TLS certificate verification")
	timeoutSec := fs.Float64("timeout", 3, "deadline in seconds")
	alpnFlag := fs.String("alpn", "", "comma-separated ALPN override (empty = parrot default)")
	http2Mode := fs.String("http2", "auto", "auto, require, or never")
	listClients := fs.Bool("list-clients", false, "print supported client names and exit")
	verbose := fs.Bool("verbose", false, "log handshake details to stderr")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *listClients {
		fmt.Fprint(stdout, listClientsText())
		return 0
	}

	if *addr == "" {
		fmt.Fprintln(stderr, "test-nginx-utls: --addr is required")
		return 2
	}

	mode := strings.ToLower(strings.TrimSpace(*http2Mode))
	switch mode {
	case "auto", "require", "never":
	default:
		fmt.Fprintf(stderr, "test-nginx-utls: invalid --http2 %q (want auto, require, or never)\n", *http2Mode)
		return 2
	}

	raw, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "test-nginx-utls: reading request: %v\n", err)
		return 1
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		fmt.Fprintln(stderr, "test-nginx-utls: empty request on stdin")
		return 1
	}

	opts := options{
		client:    *client,
		addr:      *addr,
		sni:       *sni,
		insecure:  *insecure,
		timeout:   time.Duration(*timeoutSec * float64(time.Second)),
		alpn:      splitALPN(*alpnFlag),
		http2Mode: mode,
		verbose:   *verbose,
	}

	out, err := doRequest(opts, raw)
	if err != nil {
		fmt.Fprintf(stderr, "test-nginx-utls: %v\n", err)
		return 1
	}
	if _, err := stdout.Write(out); err != nil {
		fmt.Fprintf(stderr, "test-nginx-utls: writing response: %v\n", err)
		return 1
	}
	return 0
}

func splitALPN(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func resolveClient(name string) (utls.ClientHelloID, error) {
	raw := strings.TrimSpace(name)
	if raw == "" {
		return utls.ClientHelloID{}, fmt.Errorf("empty client name")
	}

	if canon, ok := aliases[strings.ToLower(raw)]; ok {
		raw = canon
	}

	for key, id := range helloIDs {
		if strings.EqualFold(key, raw) {
			return id, nil
		}
	}

	if !strings.HasPrefix(strings.ToLower(raw), "hello") {
		return resolveClient("Hello" + raw)
	}

	return utls.ClientHelloID{}, fmt.Errorf("unknown client %q\n%s", name, listClientsText())
}

func listClientsText() string {
	var aliasNames []string
	for a := range aliases {
		aliasNames = append(aliasNames, a)
	}
	sort.Strings(aliasNames)

	var ids []string
	for id := range helloIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var b strings.Builder
	b.WriteString("aliases:\n")
	for _, a := range aliasNames {
		fmt.Fprintf(&b, "  %s = %s\n", a, aliases[a])
	}
	b.WriteString("ids:\n")
	for _, id := range ids {
		fmt.Fprintf(&b, "  %s\n", id)
	}
	return b.String()
}

func alpnOverride(opts options) []string {
	if len(opts.alpn) > 0 {
		return opts.alpn
	}
	switch opts.http2Mode {
	case "require":
		return []string{"h2"}
	case "never":
		return []string{"http/1.1"}
	default:
		return nil
	}
}

func applyALPN(spec *utls.ClientHelloSpec, protos []string) {
	if len(protos) == 0 {
		return
	}
	replaced := false
	for i, ext := range spec.Extensions {
		if _, ok := ext.(*utls.ALPNExtension); ok {
			spec.Extensions[i] = &utls.ALPNExtension{AlpnProtocols: protos}
			replaced = true
		}
	}
	if !replaced {
		spec.Extensions = append(spec.Extensions, &utls.ALPNExtension{AlpnProtocols: protos})
	}
}

func handshake(opts options) (net.Conn, string, error) {
	id, err := resolveClient(opts.client)
	if err != nil {
		return nil, "", err
	}

	dialer := net.Dialer{Timeout: opts.timeout}
	tcpConn, err := dialer.Dial("tcp", opts.addr)
	if err != nil {
		return nil, "", fmt.Errorf("dial %s: %w", opts.addr, err)
	}
	if err := tcpConn.SetDeadline(time.Now().Add(opts.timeout)); err != nil {
		tcpConn.Close()
		return nil, "", err
	}

	nextProtos := alpnOverride(opts)
	cfg := &utls.Config{
		ServerName:         opts.sni,
		InsecureSkipVerify: opts.insecure,
		NextProtos:         nextProtos,
		MinVersion:         tls.VersionTLS12,
	}

	var uconn *utls.UConn
	if id.Client == utls.HelloGolang.Client && id.Version == utls.HelloGolang.Version {
		if len(cfg.NextProtos) == 0 {
			switch opts.http2Mode {
			case "require":
				cfg.NextProtos = []string{"h2"}
			case "never":
				cfg.NextProtos = []string{"http/1.1"}
			default:
				cfg.NextProtos = []string{"h2", "http/1.1"}
			}
		}
		uconn = utls.UClient(tcpConn, cfg, utls.HelloGolang)
	} else {
		spec, specErr := utls.UTLSIdToSpec(id)
		if specErr != nil {
			uconn = utls.UClient(tcpConn, cfg, id)
		} else {
			applyALPN(&spec, nextProtos)
			uconn = utls.UClient(tcpConn, cfg, utls.HelloCustom)
			if err := uconn.ApplyPreset(&spec); err != nil {
				tcpConn.Close()
				return nil, "", fmt.Errorf("apply fingerprint %s: %w", opts.client, err)
			}
		}
	}

	if err := uconn.Handshake(); err != nil {
		uconn.Close()
		return nil, "", fmt.Errorf("handshake as %s: %w", opts.client, err)
	}

	alpn := uconn.ConnectionState().NegotiatedProtocol
	if opts.verbose {
		fmt.Fprintf(os.Stderr, "test-nginx-utls: client=%s alpn=%q version=0x%x\n",
			opts.client, alpn, uconn.ConnectionState().Version)
	}

	if opts.http2Mode == "require" && alpn != "h2" {
		uconn.Close()
		return nil, "", fmt.Errorf("HTTP/2 required, negotiated ALPN %q", alpn)
	}
	if opts.http2Mode == "never" && alpn == "h2" {
		uconn.Close()
		return nil, "", fmt.Errorf("HTTP/2 forbidden, negotiated ALPN %q", alpn)
	}

	return uconn, alpn, nil
}

func doRequest(opts options, raw []byte) ([]byte, error) {
	conn, alpn, err := handshake(opts)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if alpn == "h2" {
		return doHTTP2(conn, raw)
	}
	return doHTTP1(conn, raw)
}

func doHTTP1(conn net.Conn, raw []byte) ([]byte, error) {
	if _, err := conn.Write(raw); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}
	resp, err := io.ReadAll(conn)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if len(resp) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	return resp, nil
}

func doHTTP2(conn net.Conn, raw []byte) ([]byte, error) {
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(raw)))
	if err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}
	req.RequestURI = ""
	if req.URL != nil {
		req.URL.Scheme = "https"
		if req.URL.Host == "" {
			req.URL.Host = req.Host
		}
	}
	req.Proto = "HTTP/2.0"
	req.ProtoMajor = 2
	req.ProtoMinor = 0
	req.Header.Del("Connection")
	req.Header.Del("Transfer-Encoding")
	req.Header.Del("Keep-Alive")
	req.Header.Del("Upgrade")
	req.Header.Del("Proxy-Connection")

	tr := &http2.Transport{}
	h2conn, err := tr.NewClientConn(conn)
	if err != nil {
		return nil, fmt.Errorf("http2 client: %w", err)
	}
	defer h2conn.Close()

	resp, err := h2conn.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("http2 roundtrip: %w", err)
	}
	defer resp.Body.Close()

	resp.Proto = "HTTP/1.1"
	resp.ProtoMajor = 1
	resp.ProtoMinor = 1
	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, fmt.Errorf("dump response: %w", err)
	}
	return dump, nil
}
