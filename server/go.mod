module main

go 1.16

replace github.com/polarbroadband/cfs/pkg/cfsprotobuf => ../pkg/cfsprotobuf

require (
	github.com/bramvdbogaerde/go-scp v1.1.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/polarbroadband/cfs/pkg/cfsprotobuf v0.0.0-00010101000000-000000000000
	github.com/polarbroadband/goto v0.2.37
	github.com/sirupsen/logrus v1.8.1
	github.com/tmc/scp v0.0.0-20170824174625-f7b48647feef
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	google.golang.org/genproto v0.0.0-20210816143620-e15ff196659d // indirect
	google.golang.org/grpc v1.40.0
)
