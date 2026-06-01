package client

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

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
