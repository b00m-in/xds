module b00m.in/xds/server

require (
	b00m.in/crypto/util v0.0.0-00010101000000-000000000000
	b00m.in/xds/resource v0.0.0-00010101000000-000000000000
	github.com/envoyproxy/go-control-plane v0.10.3-0.20221219165740-8b998257ff09
	google.golang.org/grpc v1.51.0
)

require (
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cncf/xds/go v0.0.0-20220314180256-7f1daf1720fc // indirect
	github.com/envoyproxy/protoc-gen-validate v0.9.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.2.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	google.golang.org/genproto v0.0.0-20220822174746-9e6da59bd2fc // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)

replace b00m.in/xds/resource => ../resource

replace b00m.in/crypto/util => ../../crypto/util

go 1.19
