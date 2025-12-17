package request

import (
	"net/http"
)

type Chrome struct {
	parent http.RoundTripper
}

func (chrome *Chrome) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("Referer") == "" {
		req.Header.Set("Referer", req.URL.Scheme+"://"+req.URL.Host+"/")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json, text/plain, */*")
	}
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="143", "Chromium";v="143", "Not A(Brand";v="24"`)
	req.Header.Set("User-Agent", `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36`)
	return chrome.parent.RoundTrip(req)
}

func ChromeTransport(parent http.RoundTripper) http.RoundTripper {
	return &Chrome{
		parent: parent,
	}
}
