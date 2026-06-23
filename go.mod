module github.com/chainreactors/IoM-go

go 1.24.0

toolchain go1.24.13

require (
	github.com/chainreactors/logs v0.0.0-20250312104344-9f30fa69d3c9
	github.com/mitchellh/mapstructure v1.5.0
	gopkg.in/yaml.v3 v3.0.1
)

// compatibility
require (
	golang.org/x/exp v0.0.0-20230817173708-d852ddb80c63
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/chainreactors/files v0.0.0-20231102192550-a652458cee26 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
)
