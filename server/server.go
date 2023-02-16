// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package server

import (
	"context"
	"fmt"
	"log"
        "math/rand"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	runtimeservice "github.com/envoyproxy/go-control-plane/envoy/service/runtime/v3"
	secretservice "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/v3"
        "b00m.in/crypto/util"
        "b00m.in/xds/resource"
)

const (
	grpcKeepaliveTime        = 30 * time.Second
	grpcKeepaliveTimeout     = 5 * time.Second
	grpcKeepaliveMinTime     = 30 * time.Second
	grpcMaxConcurrentStreams = 1000000
)

var (
        versions = []string{"1.2", "1.3", "1.4", "1.5", "1.6", "1.7", "1.8", "1.9", "2.0", "2.1", "2.2", "2.3", "2.4", "2.5", "3.0"}
)

type Server struct {
        Sc cache.SnapshotCache
        Port uint
        Domains []string
        NodeId string
        Version string
        CertDir string
        KeyDir string
        l Logger
        Debug bool
}

func NewServer(cfg util.SDS) *Server {
        s := &Server{NodeId: cfg.NodeId, Port: uint(cfg.SdsPort), Debug: cfg.Debug}
        if len(cfg.Domains) > 0 {
                s.Domains = cfg.Domains
        }
        s.CertDir = cfg.CertDir
        s.KeyDir = cfg.KeyDir
        if cfg.Version == "" {
                s.Version = "0.01"
        } else {
                s.Version = cfg.Version
        }
	s.l = Logger{Debug: s.Debug}
        return s
}

func (s *Server) NewSDSWithCache() (server.Server, error) {
	// Create a cache
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, s.l)
        s.Sc = cache
	// Create the snapshot that we'll serve to Envoy
	//snapshot := GenerateSnapshot()
	snapshot := resource.GenerateSnapshots(s.Version, s.CertDir, s.KeyDir, s.Domains)
	if err := snapshot.Consistent(); err != nil {
		s.l.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
		return nil, err //os.Exit(1)
	}
	//l.Debugf("will serve snapshot %+v", snapshot)

	// Add the snapshot to the cache
	if err := cache.SetSnapshot(context.Background(), s.NodeId, snapshot); err != nil {
		s.l.Errorf("snapshot error %q for %+v", err, snapshot)
		return nil, err //os.Exit(1)
	}

	// Run the xDS server
	ctx := context.Background()
	cb := &test.Callbacks{Debug: s.l.Debug}
	srv := server.NewServer(ctx, cache, cb)
        return srv, nil
}

func (s *Server) NewSDSWithoutCache() (server.Server, error) {
	// Create a cache
	cache := cache.NewSnapshotCache(false, cache.IDHash{}, s.l)
        s.Sc = cache
	// Run the xDS server
	ctx := context.Background()
	cb := &test.Callbacks{Debug: s.l.Debug}
	srv := server.NewServer(ctx, cache, cb)
        return srv, nil
}

func (s *Server) SetCacheSnapshot(snapshot *cache.Snapshot) error {
        if err := snapshot.Consistent(); err != nil {
                s.l.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
                return err //os.Exit(1)
        }

        // Add the snapshot to the cache
        if err := s.Sc.SetSnapshot(context.Background(), s.NodeId, snapshot); err != nil {
                s.l.Errorf("snapshot error %q for %+v", err, snapshot)
                return err //os.Exit(1)
        }
        return nil
}

//Refreshsc listens on a channel and if anything is sent on the channel simply regenerates the snapshot with a new version number and sets the cache with this snapshot
func (s *Server) RefreshSc(rc <-chan int) error {
        for {
                select {
                case <-rc:
                        vv := versions[rand.Intn(13)]
                        fmt.Printf("%v: re-generating snapshot version: %s \n", time.Now(), vv)
                        // Create the snapshot that we'll serve to Envoy
                        //snapshot := GenerateSnapshot()
                        snapshot := resource.GenerateSnapshots(vv, s.CertDir, s.KeyDir, s.Domains)
                        if err := snapshot.Consistent(); err != nil {
                                s.l.Errorf("snapshot inconsistency: %+v\n%+v", snapshot, err)
                                return err //os.Exit(1)
                        }
                        //l.Debugf("will serve snapshot %+v", snapshot)

                        // Add the snapshot to the cache
                        if err := s.Sc.SetSnapshot(context.Background(), s.NodeId, snapshot); err != nil {
                                s.l.Errorf("snapshot error %q for %+v", err, snapshot)
                                return err //os.Exit(1)
                        }
                }
        }
        return nil
}

func registerServer(grpcServer *grpc.Server, server server.Server) {
	// register services
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	secretservice.RegisterSecretDiscoveryServiceServer(grpcServer, server)
	runtimeservice.RegisterRuntimeDiscoveryServiceServer(grpcServer, server)
}

// RunServer starts an xDS server at the given port. ctx is never used!
func RunServer(ctx context.Context, srv server.Server, port uint) {
	// gRPC golang library sets a very small upper bound for the number gRPC/h2
	// streams over a single TCP connection. If a proxy multiplexes requests over
	// a single connection to the management server, then it might lead to
	// availability problems. Keepalive timeouts based on connection_keepalive parameter https://www.envoyproxy.io/docs/envoy/latest/configuration/overview/examples#dynamic
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions,
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    grpcKeepaliveTime,
			Timeout: grpcKeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcKeepaliveMinTime,
			PermitWithoutStream: true,
		}),
	)
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}

	registerServer(grpcServer, srv)

	log.Printf("management server listening on %d\n", port)
	if err = grpcServer.Serve(lis); err != nil {
		log.Println(err)
	}
}

// RunServer starts an xDS server at the given port. ctx is never used!
func RunServerGo(errch chan<-error, srv server.Server, port uint) {
	// gRPC golang library sets a very small upper bound for the number gRPC/h2
	// streams over a single TCP connection. If a proxy multiplexes requests over
	// a single connection to the management server, then it might lead to
	// availability problems. Keepalive timeouts based on connection_keepalive parameter https://www.envoyproxy.io/docs/envoy/latest/configuration/overview/examples#dynamic
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions,
		grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    grpcKeepaliveTime,
			Timeout: grpcKeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcKeepaliveMinTime,
			PermitWithoutStream: true,
		}),
	)
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
                errch<-err
	}

	registerServer(grpcServer, srv)

	log.Printf("management server listening on %d\n", port)
	if err = grpcServer.Serve(lis); err != nil {
		log.Println(err)
                errch<-err
	}
}
