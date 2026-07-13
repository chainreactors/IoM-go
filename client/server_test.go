package client

import (
	"strings"
	"sync"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func remPipeline(name, listenerID, link string) *clientpb.Pipeline {
	return &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{Rem: &clientpb.REM{
			Name:       name,
			ListenerId: listenerID,
			Link:       link,
		}},
	}
}

func TestFindCachedPipelineFallsBackToEligibleScopedEntry(t *testing.T) {
	state := &ServerState{Pipelines: map[string]*clientpb.Pipeline{
		"shared":            remPipeline("shared", "listener-old", ""),
		"listener-a:shared": remPipeline("shared", "listener-a", "tcp://127.0.0.1:19966"),
	}}

	got, err := state.FindCachedPipeline("shared", func(pipeline *clientpb.Pipeline) bool {
		return pipeline.GetRem() != nil && pipeline.GetRem().GetLink() != ""
	})
	if err != nil {
		t.Fatalf("FindCachedPipeline returned error: %v", err)
	}
	if got.GetListenerId() != "listener-a" {
		t.Fatalf("listener id = %q, want listener-a", got.GetListenerId())
	}

	got.GetRem().Link = "mutated"
	if cached := state.Pipelines["listener-a:shared"].GetRem().GetLink(); cached != "tcp://127.0.0.1:19966" {
		t.Fatalf("returned pipeline aliases cache: cached link = %q", cached)
	}
}

func TestFindCachedPipelineRejectsAmbiguousScopedEntries(t *testing.T) {
	state := &ServerState{Pipelines: map[string]*clientpb.Pipeline{
		"listener-a:shared": remPipeline("shared", "listener-a", "tcp://127.0.0.1:19966"),
		"listener-b:shared": remPipeline("shared", "listener-b", "tcp://127.0.0.1:29966"),
	}}

	_, err := state.FindCachedPipeline("shared", nil)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("FindCachedPipeline error = %v, want ambiguity error", err)
	}
}

func TestSnapshotPipelinesDoesNotAliasCache(t *testing.T) {
	state := &ServerState{Pipelines: map[string]*clientpb.Pipeline{
		"shared": remPipeline("shared", "listener-a", "tcp://127.0.0.1:19966"),
	}}

	snapshot := state.SnapshotPipelines()
	snapshot["shared"].GetRem().Link = "mutated"
	delete(snapshot, "shared")

	if got := state.Pipelines["shared"].GetRem().GetLink(); got != "tcp://127.0.0.1:19966" {
		t.Fatalf("snapshot aliases cache: cached link = %q", got)
	}
}

func TestSnapshotPipelinesConcurrentWithEventReconciliation(t *testing.T) {
	state := &ServerState{Pipelines: make(map[string]*clientpb.Pipeline)}
	const iterations = 2000
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			pipeline := remPipeline("shared", "listener-a", "tcp://127.0.0.1:19966")
			state.ReconcileEvent(&clientpb.Event{
				Type: consts.EventJob,
				Op:   consts.CtrlPipelineStart,
				Job:  &clientpb.Job{Pipeline: pipeline},
			})
			state.ReconcileEvent(&clientpb.Event{
				Type: consts.EventJob,
				Op:   consts.CtrlPipelineStop,
				Job:  &clientpb.Job{Pipeline: pipeline},
			})
		}
	}()

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			_ = state.SnapshotPipelines()
		}
	}()

	close(start)
	wg.Wait()
}

func TestStateSnapshotsDoNotAliasCaches(t *testing.T) {
	state := &ServerState{
		Sessions: map[string]*Session{
			"session-1": {Session: &clientpb.Session{SessionId: "session-1", Note: "original"}},
		},
		Listeners: map[string]*clientpb.Listener{
			"listener-1": {Id: "listener-1", Ip: "original"},
		},
		Clients: []*clientpb.Client{{ID: 1, Name: "original"}},
	}

	sessions := state.SnapshotSessions()
	listeners := state.SnapshotListeners()
	clients := state.SnapshotClients()
	sessions["session-1"].Note = "mutated"
	listeners["listener-1"].Ip = "mutated"
	clients[0].Name = "mutated"

	if state.Sessions["session-1"].Note != "original" {
		t.Fatal("session snapshot aliases the state cache")
	}
	if state.Listeners["listener-1"].Ip != "original" {
		t.Fatal("listener snapshot aliases the state cache")
	}
	if state.Clients[0].Name != "original" {
		t.Fatal("client snapshot aliases the state cache")
	}
}

func TestSessionSnapshotConcurrentWithEventReconciliation(t *testing.T) {
	oldLogDir := LogDir
	LogDir = t.TempDir()
	t.Cleanup(func() { LogDir = oldLogDir })

	state := &ServerState{Sessions: make(map[string]*Session)}
	const iterations = 2000
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			state.ReconcileEvent(&clientpb.Event{
				Type:    consts.EventSession,
				Op:      consts.CtrlSessionUpdate,
				Session: &clientpb.Session{SessionId: "session-1"},
			})
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			_ = state.SnapshotSessions()
		}
	}()

	close(start)
	wg.Wait()
}

func TestRemoveLocalSessionUsesStateLock(t *testing.T) {
	state := &ServerState{Sessions: map[string]*Session{
		"session-1": {Session: &clientpb.Session{SessionId: "session-1"}},
	}}
	state.RemoveLocalSession("session-1")
	if _, ok := state.GetLocalSession("session-1"); ok {
		t.Fatal("RemoveLocalSession left the session in the cache")
	}
}

func TestFindPipelineLockedDoesNotFallbackToBareNameForDifferentListener(t *testing.T) {
	state := &ServerState{
		Pipelines: map[string]*clientpb.Pipeline{
			"site": {
				Name:       "site",
				ListenerId: "listener-a",
				Type:       consts.WebsitePipeline,
				Body: &clientpb.Pipeline_Web{
					Web: &clientpb.Website{
						Contents: map[string]*clientpb.WebContent{
							"/a": {Path: "/a"},
						},
					},
				},
			},
		},
	}

	incoming := &clientpb.Pipeline{
		Name:       "site",
		ListenerId: "listener-b",
		Type:       consts.WebsitePipeline,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Contents: map[string]*clientpb.WebContent{
					"/b": {Path: "/b"},
				},
			},
		},
	}

	current, ok := state.findPipelineLocked(incoming)
	if ok || current != nil {
		t.Fatalf("findPipelineLocked returned %#v, want miss for different listener", current)
	}
}

func TestEventCallbackConcurrentAccess(t *testing.T) {
	state := &ServerState{EventCallback: make(map[string]func(*clientpb.Event))}
	callback := func(*clientpb.Event) {}
	const iterations = 2000

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			state.SetEventCallback("test", callback)
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			_, _ = state.GetEventCallback("test")
		}
	}()

	close(start)
	wg.Wait()
	if _, ok := state.GetEventCallback("test"); !ok {
		t.Fatal("event callback was not registered")
	}
}
