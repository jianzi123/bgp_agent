package app

import (
	"bgp_agent/pkg/bgp"
	"bgp_agent/pkg/config"
	"bgp_agent/pkg/etcdv3"
	mnet "bgp_agent/utils/net"
	"context"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
	"net"
	"time"
	"strings"
	"sync"
)

const (
	IpamPath               = "/calico/ipam/v2/host/"
	ETCDOPDELETE           = "delete"
	ETCDOPCOMPAREADNDELETE = "compareAndDelete"
	ETCDOPCREATE           = "create"
)

type App struct {
	EctdClient *etcdv3.EtcdV3
	BgpClient  *bgp.GoBGPClient
	BConfig    *config.BgpConfig
	Mutx       *sync.RWMutex
	HostName   string
}

func New(hostname, bconfigFile, configfile string) (*App, error) {
	client, err := bgp.NewGoBGPClient()
	if err != nil {
		return nil, err
	}
	config, err := config.LoadConfigFIle(bconfigFile, config.ConfigFileType, configfile)
	if err != nil {
		return nil, err
	}
	app := &App{
		EctdClient: &etcdv3.EtcdV3{},
		BgpClient:  client,
		BConfig:    config,
		HostName:   hostname,
	}
	app.SetDatastore()

	return app, nil
}

func (obj *App) SetDatastore() error {
	endpoints := strings.Split(obj.BConfig.Config.EtcdEndpoints, ",")
	logrus.Infof("SetDatastore endpoints: %v. \n", endpoints)
	err := obj.EctdClient.InitEtcd(endpoints, obj.BConfig.Config.EtcdCertFile, obj.BConfig.Config.EtcdKeyFile, obj.BConfig.Config.EtcdCaFile)
	if err != nil {
		logrus.Errorf("SetDatastore failed: %v. \n", err)
		return err
	}
	return nil
}

func (obj *App) Runs() {
	// delta
	go obj.Watch()
	// full-check
	go obj.Sync()
}

func (obj *App)Sync()  {
	for{

		logrus.Infof("full-check: ")
		addrsLocal , err := obj._get_running_network("")
		if err != nil{
			logrus.Errorf("_get_running_network failed: %v", err)
			continue
		}
		logrus.Infof("_get_running_network %v", addrsLocal)


		addrsEtcd, err := obj.GetFullIPsFromEtcd()
		if err != nil{
			logrus.Errorf("GetFullIPsFromEtcd failed: %v", err)
			continue
		}
		logrus.Infof("GetFullIPsFromEtcd %v", addrsEtcd)
		for _, value := range addrsLocal{
			if _, ok := addrsEtcd[value]; !ok{
				// delete
				err = obj.BgpClient.DeleteNetwork(value)
				if err != nil{
					logrus.Errorf("DeleteNetwork failed in loop check: %v", err)
					continue
				}
			}
		}

		for _, value := range addrsEtcd{
			if _, ok := addrsLocal[value]; !ok{
				// add
				err = obj.BgpClient.AddNetwork(value)
				if err != nil{
					logrus.Errorf("AddNetwork failed in loop check: %v", err)
					continue
				}
			}
		}

		time.Sleep(time.Second * 5)
	}

}

func (obj *App)GetFullIPsFromEtcd() (map[string]string, error) {
	fullNet := map[string]string{}
	HostName := obj.HostName
	network_path := IpamPath + HostName + "/ipv4/block/"
	logrus.Infof("%s", network_path)
	resp, err := obj.EctdClient.Get(network_path, true)
	if err != nil{
		logrus.Errorf("EctdClient.Get failed: %v", err)
		return fullNet, err
	}
	for _, value := range resp.Kvs{
		ip_address := GetIPFromEtcd(string(value.Key), network_path)
		fullNet[ip_address] = ip_address
	}
	return fullNet, nil
}

func (obj *App)_get_running_network(local_as string) (map[string]string, error) {
	running_network := make(map[string]string)

	list, err := obj.BgpClient.ListPath()
	if err != nil{
		return running_network, err
	}

	for _, value := range list{
		aip, anet, err := net.ParseCIDR(value)
		if err != nil{
			logrus.Errorf("parse prefix failed: %v", err)
			continue
		}
		ones, bits := anet.Mask.Size()
		if ones != 32 || bits != 32{
			continue
		}
		running_network[aip.String()] = aip.String()
	}

	return running_network, nil
}

const (
	PUT    int32 = 0
	DELETE int32 = 1
)

func GetIPFromEtcd(key, path string) (string) {
	ip_part := strings.Replace(key, path, "", -1)
	_, ip_address := mnet.AnalyIP(ip_part)
	return ip_address
}

func (obj *App) Watch() {

	//receiver := make(chan *client.Response)
	HostName := obj.HostName
	network_path := IpamPath + HostName + "/ipv4/block/"
	logrus.Infof("%s", network_path)

	wch := obj.EctdClient.Watch(context.Background(), network_path, clientv3.WithPrefix())
	for{
		select {
		case recerver := <- wch:
			for _, value := range recerver.Events{
				if int32(value.Type) == DELETE{
					ip_address := GetIPFromEtcd(string(value.Kv.Key), network_path)
					logrus.Infof("DELETE key %s ip_address %s", string(value.Kv.Key), ip_address)
					// broadcast bgp
					err := obj.BgpClient.DeleteNetwork(ip_address)
					if err != nil {
						logrus.Errorf("DeleteNetwork by gobgp failed when watching event from etcd: %v", err)
						continue
					}
				}else if int32(value.Type) == PUT{
					ip_address := GetIPFromEtcd(string(value.Kv.Key), network_path)
					logrus.Infof("PUT key %s ip_address %s", string(value.Kv.Key), ip_address)
					// broadcast bgp
					err := obj.BgpClient.AddNetwork(ip_address)
					if err != nil {
						logrus.Errorf("AddNetwork by gobgp failed when watching event from etcd: %v", err)
						continue
					}
				}
			}
		}
	}
}
