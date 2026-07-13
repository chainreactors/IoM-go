package client

import (
	"sync"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func TestActiveTargetConcurrentAccess(t *testing.T) {
	target := &ActiveTarget{}
	session := &Session{Session: &clientpb.Session{SessionId: "session-1"}}
	const iterations = 2000

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			target.Set(session)
			target.Background()
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			_ = target.Get()
		}
	}()

	close(start)
	wg.Wait()
	target.Set(session)
	if got := target.Get(); got != session {
		t.Fatalf("active session = %p, want %p", got, session)
	}
}
