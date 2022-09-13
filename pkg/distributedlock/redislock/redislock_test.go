package redislock

import "testing"

func TestBase(t *testing.T) {
	l := NewDisLock(nil, "distributed_key")
	if l == nil {
		t.Fatal("")
	}
}
