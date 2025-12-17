package request

import (
	"compress/gzip"
	"compress/zlib"
	"github.com/klauspost/compress/zstd"
	"github.com/yydsqu/tools/request/brotli"
	"io"
	"net/http"
	"strings"
)

var (
	AcceptEncoding = strings.Join([]string{"gzip", "deflate", "br", "zstd"}, ",")
)

func markUncompressed(resp *http.Response) {
	resp.Uncompressed = true
}

type readCloseWrapper struct {
	io.Reader
	Closer io.Closer
}

func (w *readCloseWrapper) Close() error {
	if c, ok := w.Reader.(io.Closer); ok {
		_ = c.Close()
	}
	return w.Closer.Close()
}

type Encoding struct {
	parent         http.RoundTripper
	AcceptEncoding string
}

func (encoding *Encoding) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("Accept-Encoding") == "" && req.Header.Get("Range") == "" && req.Method != "HEAD" {
		req.Header.Set("Accept-Encoding", encoding.AcceptEncoding)
	}

	resp, err := encoding.parent.RoundTrip(req)
	if err != nil || resp == nil || resp.Body == nil {
		return resp, err
	}

	switch strings.TrimSpace(resp.Header.Get("Content-Encoding")) {
	case "gzip":
		rc, err := gzip.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body = &readCloseWrapper{Reader: rc, Closer: resp.Body}
		markUncompressed(resp)
	case "deflate":
		rc, err := zlib.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body = &readCloseWrapper{Reader: rc, Closer: resp.Body}
		markUncompressed(resp)
	case "br":
		resp.Body = &readCloseWrapper{Reader: brotli.NewReader(resp.Body), Closer: resp.Body}
		markUncompressed(resp)
	case "zstd":
		enc, err := zstd.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body = &readCloseWrapper{Reader: enc.IOReadCloser(), Closer: resp.Body}
		markUncompressed(resp)
	}

	return resp, nil
}

func EncodingTransport(parent http.RoundTripper) http.RoundTripper {
	return &Encoding{
		parent:         parent,
		AcceptEncoding: AcceptEncoding,
	}
}
