module main

go 1.16

replace github.com/polarbroadband/cfs/pkg/cfsprotobuf => ../../pkg/cfsprotobuf

require (
	github.com/polarbroadband/cfs/pkg/cfsprotobuf v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.8.1
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/sys v0.0.0-20210816183151-1e6c022a8912 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20210816143620-e15ff196659d // indirect
	google.golang.org/grpc v1.40.0
)
