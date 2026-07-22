package types

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

func TestBuildSpiteTunnelMessages(t *testing.T) {
	spite, err := BuildSpite(&implantpb.Spite{}, &implantpb.TunnelOpen{ConnId: 1, Host: "a.com", Port: 80})
	if err != nil {
		t.Fatal(err)
	}
	if spite.Name != consts.ModuleTcpRelay {
		t.Fatalf("name=%s", spite.Name)
	}
	if spite.GetTunnelOpen() == nil || spite.GetTunnelOpen().Host != "a.com" {
		t.Fatalf("open body=%v", spite.GetTunnelOpen())
	}

	spite, err = BuildSpite(&implantpb.Spite{}, &implantpb.TunnelCtrl{Action: implantpb.TunnelCtrl_START})
	if err != nil {
		t.Fatal(err)
	}
	if spite.GetTunnelCtrl() == nil {
		t.Fatal("ctrl nil")
	}

	spite, err = BuildSpite(&implantpb.Spite{}, &implantpb.TunnelData{ConnId: 2, Data: []byte("x")})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(spite.GetTunnelData().GetData()); got != "x" {
		t.Fatalf("data=%q", got)
	}

	// MessageType mapping
	if MessageType(spite) != MsgTunnelData {
		t.Fatalf("msg type=%s", MessageType(spite))
	}
}
