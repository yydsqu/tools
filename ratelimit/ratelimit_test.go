package ratelimit

import (
	"fmt"
	"github.com/yydsqu/tools/log"
	"testing"
	"time"
)

func TestSmoothLimiter(t *testing.T) {
	lim, _ := NewSmoothLimiter(5, 5)
	for i := 0; i < 10; i++ {
		if lim.Allow() {
			log.Info("Allow")
		} else {
			log.Info("NotAllow")
		}
	}
	time.Sleep(2 * time.Second)
	fmt.Println("========================================")
	for i := 0; i < 10; i++ {
		if lim.Allow() {
			log.Info("Allow")
		} else {
			log.Info("NotAllow")
		}
	}
}
