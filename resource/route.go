package resource

import (
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
        "github.com/gin-gonic/gin"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

type RoutesResource struct {
        ri gin.RoutesInfo
        routeName string
        clusterName string
}

func NewRR(routename, clustername string, gri gin.RoutesInfo) *RoutesResource {
        return &RoutesResource{gri, routename, clustername}
}

//func MakeRoutes(r gin.RoutesInfo, routeName, clusterName, version string) *route.RouteConfiguration 
func (rr *RoutesResource) MakeRoutes(version string) *route.RouteConfiguration {
        routes := make([]*route.Route, 0)
        for _,x := range rr.ri {
                rte := &route.Route{
                        Match: &route.RouteMatch{
                                PathSpecifier: &route.RouteMatch_Prefix{
                                        Prefix: x.Path, //"/",
                                },
                        },
                        Action: &route.Route_Route{
                                Route: &route.RouteAction{
                                        ClusterSpecifier: &route.RouteAction_Cluster{
                                                Cluster: rr.clusterName,
                                        },
                                        HostRewriteSpecifier: &route.RouteAction_HostRewriteLiteral{
                                                HostRewriteLiteral: UpstreamHost,
                                        },
                                },
                        },
                }
                routes = append(routes, rte)
        }
	return &route.RouteConfiguration{
		Name: rr.routeName,
		VirtualHosts: []*route.VirtualHost{{
			Name:    rr.routeName + version,
			Domains: []string{"*"},
			Routes: routes,
			}},
	}
}

//func GenerateRouteSnapshot(version string) *cache.Snapshot {
func (rr *RoutesResource) GenerateRoutesSnapshot(version string) *cache.Snapshot {
	snap, _ := cache.NewSnapshot(version,
		map[resource.Type][]types.Resource{
			resource.ClusterType:  {makeCluster(rr.clusterName)},
			resource.RouteType:    {rr.MakeRoutes(version)},
			resource.ListenerType: {makeHTTPListener(ListenerName, rr.routeName)},
		},
	)
	return snap
}
