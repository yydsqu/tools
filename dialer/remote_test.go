package dialer

import (
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestRemote(t *testing.T) {
	dial, err := NewRemote("socks5://127.0.0.1:1080")
	if err != nil {
		t.Fatal(err)
	}
	client := http.Client{
		Transport: &http.Transport{
			Dial: dial.Dial,
		},
	}
	resp, err := client.Get("https://api.ipify.org/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(body))
}
