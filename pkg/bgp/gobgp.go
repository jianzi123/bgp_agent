package bgp

import (
	"context"

	"github.com/sirupsen/logrus"
	"io"
	"net"

	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	google_protobuf "github.com/golang/protobuf/ptypes/any"
	api "github.com/osrg/gobgp/api"
	"google.golang.org/grpc"
)

// read by
// https://github.com/osrg/gobgp/blob/master/docs/sources/lib.md
const (
	defaultGoBGPTarget = "127.0.0.1:50051"
	defaultNextHop     = "0.0.0.0"
)

//var families []api.Family = []api.Family{api.Family{
//	Afi:  api.Family_AFI_IP,
//	Safi: api.Family_SAFI_UNICAST,
//}, api.Family{
//	Afi:  api.Family_AFI_IP6,
//	Safi: api.Family_SAFI_UNICAST,
//},
//}

type GoBGPClient struct {
	client api.GobgpApiClient
	cancel context.CancelFunc
	c      *grpc.ClientConn
}

// start bgp server
func (c *GoBGPClient) AddLocalPeer(routerId string, localAS int) error {
	if _, err := c.client.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: &api.Global{
			As:       uint32(localAS),
			RouterId: routerId,
		},
	}); err != nil {
		return err
	}
	return nil
}

func (c *GoBGPClient) AddNetwork(advIP string) error {
	family := &api.Family{
		Afi:  api.Family_AFI_IP,
		Safi: api.Family_SAFI_UNICAST,
	}

	nlri1, _ := ptypes.MarshalAny(&api.IPAddressPrefix{
		Prefix:    advIP,
		PrefixLen: 32,
	})
	a1, _ := ptypes.MarshalAny(&api.OriginAttribute{
		Origin: 0,
	})
	a2, _ := ptypes.MarshalAny(&api.NextHopAttribute{
		NextHop: defaultNextHop,
	})
	attrs := []*google_protobuf.Any{a1, a2}

	_, err := c.client.AddPath(context.Background(), &api.AddPathRequest{
		TableType: api.TableType_GLOBAL,
		Path: &api.Path{
			//SourceAsn:  2002,
			Family: family,
			Nlri:   nlri1,
			Pattrs: attrs,
		},
	})

	return err
}

func fromAPIPath(path *api.Path) (net.IP, error) {
	for _, attr := range path.Pattrs {
		var value ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(attr, &value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal route distinguisher: %s", err)
		}

		switch a := value.Message.(type) {
		case *api.NextHopAttribute:
			nexthop := net.ParseIP(a.NextHop).To4()
			if nexthop == nil {
				if nexthop = net.ParseIP(a.NextHop).To16(); nexthop == nil {
					return nil, fmt.Errorf("invalid nexthop address: %s", a.NextHop)
				}
			}
			return nexthop, nil
		}
	}

	return nil, fmt.Errorf("cannot find nexthop")
}

//func GetPrefixfromAPIPath(path *api.Path) (net.IP, error) {
//	for _, attr := range path.Pattrs {
//		var value ptypes.DynamicAny
//		if err := ptypes.UnmarshalAny(attr, &value); err != nil {
//			return nil, fmt.Errorf("failed to unmarshal route distinguisher: %s", err)
//		}
//		logrus.Infof("GetPrefixfromAPIPath: %s \n", value.String())
//		logrus.Infof("-------value.type: %v", value.Message)
//		switch v := value.Message.(type) {
//		case *api.IPAddressPrefix:
//			Prefix := net.ParseIP(v.Prefix).To4()
//			nlri := bgp.NewIPAddrPrefix(uint8(v.PrefixLen), v.Prefix)
//			logrus.Infof("++++++++++1 NewIPAddrPrefix %s prefix: %s", nlri.String(), Prefix.String())
//		case *api.LabeledIPAddressPrefix:
//			nlri := bgp.NewLabeledIPAddrPrefix(uint8(v.PrefixLen), v.Prefix, *bgp.NewMPLSLabelStack(v.Labels...))
//			logrus.Infof("++++++++++2 NewIPAddrPrefix %s", nlri.String())
//		case *api.EncapsulationNLRI:
//			nlri := bgp.NewEncapNLRI(v.Address)
//			logrus.Infof("++++++++++3 NewIPAddrPrefix %s", nlri.String())
//		case *api.EVPNEthernetAutoDiscoveryRoute:
//		case *api.EVPNMACIPAdvertisementRoute:
//		case *api.EVPNInclusiveMulticastEthernetTagRoute:
//		case *api.EVPNEthernetSegmentRoute:
//		case *api.EVPNIPPrefixRoute:
//		case *api.LabeledVPNIPAddressPrefix:
//		case *api.RouteTargetMembershipNLRI:
//			//rt, err := UnmarshalRT(v.Rt)
//			//if err != nil {
//			//	return nil, err
//			//}
//			//nlri = bgp.NewRouteTargetMembershipNLRI(v.As, rt)
//		case *api.FlowSpecNLRI:
//			//rules, err := UnmarshalFlowSpecRules(v.Rules)
//			//if err != nil {
//			//	return nil, err
//			//}
//			//switch rf {
//			//case bgp.RF_FS_IPv4_UC:
//			//	nlri = bgp.NewFlowSpecIPv4Unicast(rules)
//			//case bgp.RF_FS_IPv6_UC:
//			//	nlri = bgp.NewFlowSpecIPv6Unicast(rules)
//			//}
//		case *api.VPNFlowSpecNLRI:
//		}
//	}
//
//	return nil, fmt.Errorf("cannot find prefix")
//}

func (c *GoBGPClient) ListPath() ([]string, error) {
	logrus.Infof("ListPath by gobgp1")
	result := []string{}
	stream, err := c.client.ListPath(context.Background(), &api.ListPathRequest{
		TableType: api.TableType_GLOBAL,
		Family: &api.Family{
			Afi:  api.Family_AFI_IP,
			Safi: api.Family_SAFI_UNICAST,
		},
		SortType: api.ListPathRequest_PREFIX,
	})
	if err != nil {
		return result, err
	}

	rib := make([]*api.Destination, 0)
	for {
		r, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return result, err
		}
		rib = append(rib, r.Destination)
		result = append(result, r.GetDestination().GetPrefix())
	}
	logrus.Infof("ListPath by gobgp2: %v", result)
	return result, nil
}

func (c *GoBGPClient) DeleteNetwork(advIP string) error {
	logrus.Infof("DeleteNetwork by gobgp %s", advIP)
	family := &api.Family{
		Afi:  api.Family_AFI_IP,
		Safi: api.Family_SAFI_UNICAST,
	}

	nlri, _ := ptypes.MarshalAny(&api.IPAddressPrefix{
		Prefix:    advIP,
		PrefixLen: 32,
	})
	a1, _ := ptypes.MarshalAny(&api.OriginAttribute{
		Origin: 0,
	})
	a2, _ := ptypes.MarshalAny(&api.NextHopAttribute{
		NextHop: defaultNextHop,
	})
	attrs := []*google_protobuf.Any{a1, a2}
	_, err := c.client.DeletePath(context.Background(), &api.DeletePathRequest{
		TableType: api.TableType_GLOBAL,
		Path: &api.Path{
			Family: family,
			Nlri:   nlri,
			Pattrs: attrs,
		},
	})
	return err
}

func (c *GoBGPClient) Close() {
	c.c.Close()
}

func NewGoBGPClient() (*GoBGPClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	grpcOpts := []grpc.DialOption{grpc.WithBlock(), grpc.WithInsecure()}
	conn, err := grpc.DialContext(ctx, defaultGoBGPTarget, grpcOpts...)
	if err != nil {
		return nil, fmt.Errorf("gobgp client connect bgpd: %s", err)
	}
	client := api.NewGobgpApiClient(conn)
	return &GoBGPClient{
		client: client,
		cancel: cancel,
		c:      conn,
	}, nil
}
