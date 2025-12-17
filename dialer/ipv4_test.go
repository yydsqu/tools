package dialer

import (
	"fmt"
	"testing"
)

func TestLoadLocalIps(t *testing.T) {
	ips, err := LoadLocalIPV4()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(len(ips))
	fmt.Println(ips)
}

func TestLoadReachableIPV4(t *testing.T) {
	ips, err := LoadReachableIPV4()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("IP地址:", len(ips))
	fmt.Println(ips)
}
