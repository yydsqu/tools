package request

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func server() {
	http.ListenAndServe(":5566", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.Header)
		io.WriteString(w, r.RemoteAddr)
	}))
}

func TestEncodingTransport(t *testing.T) {
	client := http.Client{
		Transport: ChromeTransport(http.DefaultTransport),
	}
	resp, err := client.Get("https://www.google.com/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)
}

func IP(cli *http.Client, url string) {
	resp, err := cli.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	io.Copy(os.Stdout, resp.Body)
	fmt.Println("\n=================")
}

func TestRoundRobinProxy(t *testing.T) {
	go server()
	time.Sleep(time.Second)
	transport, err := LoadLocalDialerTransport(http.DefaultTransport.(*http.Transport), EncodingTransport, ChromeTransport)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{
		Transport: transport,
	}
	// 获取IP地址
	for i := 0; i < 100; i++ {
		time.Sleep(time.Second)
		IP(client, "http://192.168.1.6:5566")
	}
}
