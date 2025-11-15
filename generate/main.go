package main

//go:generate protoc -I proto/ proto/client/clientpb/client.proto --go_out=paths=source_relative:../proto
//go:generate protoc -I proto/ proto/client/rootpb/root.proto --go_out=paths=source_relative:../proto
//go:generate protoc -I proto/ proto/implant/implantpb/implant.proto --go_out=paths=source_relative:../proto
//go:generate protoc -I proto/ proto/implant/implantpb/module.proto --go_out=paths=source_relative:../proto
//go:generate protoc -I proto/ proto/services/clientrpc/service.proto --go_out=paths=source_relative:../proto --go-grpc_out=paths=source_relative:../proto
//go:generate protoc -I proto/ proto/services/listenerrpc/service.proto --go_out=paths=source_relative:../proto --go-grpc_out=paths=source_relative:../proto

func main() {

}
