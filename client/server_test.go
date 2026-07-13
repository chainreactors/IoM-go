package client

import (
	"strings"
	"sync"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"google.golang.org/protobuf/proto"
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

func TestReconcileWebsiteContentUpdateRefreshesCachedContent(t *testing.T) {
	state := &ServerState{Pipelines: map[string]*clientpb.Pipeline{
		"site": {
			Name:       "site",
			ListenerId: "listener-a",
			Type:       consts.WebsitePipeline,
			Body: &clientpb.Pipeline_Web{Web: &clientpb.Website{
				Contents: map[string]*clientpb.WebContent{
					"/payload": {Path: "/payload", Comment: "old"},
				},
			}},
		},
	}}

	state.ReconcileEvent(&clientpb.Event{
		Type: consts.EventWebsite,
		Op:   consts.CtrlWebContentUpdate,
		Job: &clientpb.Job{
			Pipeline: &clientpb.Pipeline{
				Name:       "site",
				ListenerId: "listener-a",
				Type:       consts.WebsitePipeline,
				Body:       &clientpb.Pipeline_Web{Web: &clientpb.Website{}},
			},
			Contents: map[string]*clientpb.WebContent{
				"/payload": {Path: "/payload", Comment: "updated"},
			},
		},
	})

	if got := state.Pipelines["site"].GetWeb().GetContents()["/payload"].GetComment(); got != "updated" {
		t.Fatalf("cached comment = %q, want updated", got)
	}
}

func TestReconcileEventUpsertsManagementPipelineLifecycle(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		operation string
		pipeline  *clientpb.Pipeline
	}{
		{
			name:      "pipeline register",
			eventType: consts.EventJob,
			operation: consts.CtrlPipelineRegister,
			pipeline:  &clientpb.Pipeline{Name: "tcp-register", ListenerId: "listener-a", Type: consts.TCPPipeline},
		},
		{
			name:      "rem register",
			eventType: consts.EventJob,
			operation: consts.CtrlRemRegister,
			pipeline:  remPipeline("rem-register", "listener-a", "tcp://127.0.0.1:19966"),
		},
		{
			name:      "website register",
			eventType: consts.EventWebsite,
			operation: consts.CtrlWebsiteRegister,
			pipeline: &clientpb.Pipeline{
				Name:       "web-register",
				ListenerId: "listener-a",
				Type:       consts.WebsitePipeline,
				Body:       &clientpb.Pipeline_Web{Web: &clientpb.Website{Name: "web-register"}},
			},
		},
		{
			name:      "website update",
			eventType: consts.EventWebsite,
			operation: consts.CtrlWebsiteUpdate,
			pipeline: &clientpb.Pipeline{
				Name:       "web-update",
				ListenerId: "listener-a",
				Type:       consts.WebsitePipeline,
				Enable:     true,
				Body:       &clientpb.Pipeline_Web{Web: &clientpb.Website{Name: "web-update"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &ServerState{Pipelines: make(map[string]*clientpb.Pipeline)}
			state.ReconcileEvent(&clientpb.Event{
				Type: test.eventType,
				Op:   test.operation,
				Job:  &clientpb.Job{Pipeline: test.pipeline},
			})

			got, err := state.FindCachedPipeline(test.pipeline.Name, nil)
			if err != nil {
				t.Fatalf("FindCachedPipeline failed after %s: %v", test.operation, err)
			}
			if got.GetListenerId() != test.pipeline.GetListenerId() || got.GetType() != test.pipeline.GetType() {
				t.Fatalf("cached pipeline = %#v, want listener %q type %q", got, test.pipeline.GetListenerId(), test.pipeline.GetType())
			}
			if test.operation == consts.CtrlWebsiteUpdate && !got.GetEnable() {
				t.Fatal("website update did not replace the cached pipeline state")
			}
		})
	}
}

func TestReconcileEventRemovesManagementPipelineLifecycle(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		operation string
		pipeline  *clientpb.Pipeline
	}{
		{
			name:      "pipeline delete",
			eventType: consts.EventJob,
			operation: consts.CtrlPipelineDelete,
			pipeline:  &clientpb.Pipeline{Name: "tcp-delete", ListenerId: "listener-a", Type: consts.TCPPipeline},
		},
		{
			name:      "rem delete",
			eventType: consts.EventJob,
			operation: consts.CtrlRemDelete,
			pipeline:  remPipeline("rem-delete", "listener-a", "tcp://127.0.0.1:19966"),
		},
		{
			name:      "website delete",
			eventType: consts.EventWebsite,
			operation: consts.CtrlWebsiteDelete,
			pipeline: &clientpb.Pipeline{
				Name:       "web-delete",
				ListenerId: "listener-a",
				Type:       consts.WebsitePipeline,
				Body:       &clientpb.Pipeline_Web{Web: &clientpb.Website{Name: "web-delete"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &ServerState{Pipelines: map[string]*clientpb.Pipeline{
				test.pipeline.Name: test.pipeline,
			}}
			state.ReconcileEvent(&clientpb.Event{
				Type: test.eventType,
				Op:   test.operation,
				Job:  &clientpb.Job{Pipeline: test.pipeline},
			})

			for key, cached := range state.SnapshotPipelines() {
				if cached.GetName() == test.pipeline.GetName() && cached.GetListenerId() == test.pipeline.GetListenerId() {
					t.Fatalf("pipeline remained cached at %q after %s", key, test.operation)
				}
			}
		})
	}
}

func TestReconcileEventKeepsStoppedPipelineAsDisabled(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		operation string
		pipeline  *clientpb.Pipeline
	}{
		{
			name:      "pipeline stop",
			eventType: consts.EventJob,
			operation: consts.CtrlPipelineStop,
			pipeline:  &clientpb.Pipeline{Name: "tcp-stop", ListenerId: "listener-a", Type: consts.TCPPipeline},
		},
		{
			name:      "rem stop",
			eventType: consts.EventJob,
			operation: consts.CtrlRemStop,
			pipeline:  remPipeline("rem-stop", "listener-a", "tcp://127.0.0.1:19966"),
		},
		{
			name:      "website stop",
			eventType: consts.EventWebsite,
			operation: consts.CtrlWebsiteStop,
			pipeline: &clientpb.Pipeline{
				Name:       "web-stop",
				ListenerId: "listener-a",
				Type:       consts.WebsitePipeline,
				Body:       &clientpb.Pipeline_Web{Web: &clientpb.Website{Name: "web-stop"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			active := proto.Clone(test.pipeline).(*clientpb.Pipeline)
			active.Enable = true
			state := &ServerState{Pipelines: map[string]*clientpb.Pipeline{
				active.GetName(): active,
			}}
			state.ReconcileEvent(&clientpb.Event{
				Type: test.eventType,
				Op:   test.operation,
				Job:  &clientpb.Job{Pipeline: test.pipeline},
			})

			got, err := state.FindCachedPipeline(test.pipeline.GetName(), nil)
			if err != nil {
				t.Fatalf("FindCachedPipeline failed after %s: %v", test.operation, err)
			}
			if got.GetEnable() {
				t.Fatalf("cached pipeline remained enabled after %s", test.operation)
			}
		})
	}
}

func TestReconcileListenerStopDisablesOwnedPipelines(t *testing.T) {
	state := &ServerState{
		Listeners: map[string]*clientpb.Listener{
			"listener-a": {Id: "listener-a"},
			"listener-b": {Id: "listener-b"},
		},
		Pipelines: map[string]*clientpb.Pipeline{
			"tcp-a": {
				Name:       "tcp-a",
				ListenerId: "listener-a",
				Enable:     true,
				Type:       consts.TCPPipeline,
			},
			"listener-a:shared": {
				Name:       "shared",
				ListenerId: "listener-a",
				Enable:     true,
				Type:       consts.TCPPipeline,
			},
			"listener-b:shared": {
				Name:       "shared",
				ListenerId: "listener-b",
				Enable:     true,
				Type:       consts.TCPPipeline,
			},
		},
	}

	state.ReconcileEvent(&clientpb.Event{
		Type:     consts.EventListener,
		Op:       consts.CtrlListenerStop,
		Listener: &clientpb.Listener{Id: "listener-a"},
	})

	if _, ok := state.SnapshotListeners()["listener-a"]; ok {
		t.Fatal("stopped listener remained cached")
	}
	for key, pipeline := range state.SnapshotPipelines() {
		if pipeline.GetListenerId() == "listener-a" && pipeline.GetEnable() {
			t.Fatalf("pipeline %q remained enabled after its listener stopped", key)
		}
		if pipeline.GetListenerId() == "listener-b" && !pipeline.GetEnable() {
			t.Fatalf("pipeline %q from another listener was disabled", key)
		}
	}
}
