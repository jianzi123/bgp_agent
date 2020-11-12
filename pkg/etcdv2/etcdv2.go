package etcdv2

import (
	"github.com/sirupsen/logrus"
	etcd_client "go.etcd.io/etcd/client"
	"golang.org/x/net/context"
	"github.com/coreos/etcd/pkg/transport"
	"net/http"
	"net"
	"time"
)

//const (
//    etcdKeyNotFound = " 100: Key not found"
//    etcdKeyalReadyExists = " 105: Key already exists"
//)

type EtcdV2 struct {
	keysAPI etcd_client.KeysAPI
}

func newHTTPSTransport(certFile, keyFile, caFile string) (*http.Transport, error) {
	info := transport.TLSInfo{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}
	cfg, err := info.ClientConfig()
	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     cfg,
	}

	return tr, nil
}

func (etcdcli *EtcdV2)InitEtcd(etcdServerList []string, etcdCertfile, etcdKeyFile, etcdCafile string) error {

	if etcdCertfile != "" && etcdKeyFile != "" && etcdCafile != "" {
		t_sport, err := newHTTPSTransport(etcdCertfile, etcdKeyFile, etcdCafile)
		if err != nil {
			logrus.Warningln("New https transport failed: " + err.Error())
			return err
		}

		cfg := etcd_client.Config{
			Endpoints: etcdServerList,
			Transport: t_sport,
		}
		etcdClient, err := etcd_client.New(cfg)
		if err != nil {
			return err
		}
		etcdcli.keysAPI = etcd_client.NewKeysAPI(etcdClient)
	} else {
		cfg := etcd_client.Config{
			Endpoints: etcdServerList,
		}
		etcdClient, err := etcd_client.New(cfg)
		if err != nil {
			return err
		}
		etcdcli.keysAPI = etcd_client.NewKeysAPI(etcdClient)
	}

	return nil
}

func (etcdcli *EtcdV2)Get(key string, sort bool, recursive bool) (*etcd_client.Response, error) {
	getOptions := &etcd_client.GetOptions{
		Recursive: recursive,
		Sort:      sort,
	}
	resp, err := etcdcli.keysAPI.Get(context.TODO(), key, getOptions)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (etcdcli *EtcdV2)Set(key string, val string) error {
	setOptions := &etcd_client.SetOptions{
		PrevExist:etcd_client.PrevNoExist,
	}
	_, err := etcdcli.keysAPI.Set(context.TODO(), key, val, setOptions)

	if err != nil{
		return err
	}
	return nil
}

func (etcdcli *EtcdV2)Update(key string, val string, preVal string, check bool) error {
	if check {
		setOptions := &etcd_client.SetOptions{
			PrevValue : preVal,
			PrevExist:etcd_client.PrevExist,
		}
		_, err := etcdcli.keysAPI.Set(context.TODO(), key, val, setOptions)
		return err
	} else {
		_, err := etcdcli.keysAPI.Set(context.TODO(), key, val, nil)
		return err
	}
}

func (etcdcli *EtcdV2)DoDelete(key string, val string) error {
	delOptions := &etcd_client.DeleteOptions{
		PrevValue : val,
	}
	_, err := etcdcli.keysAPI.Delete(context.TODO(), key, delOptions)

	if err != nil{
		return err
	}
	return nil
}

func (etcdcli *EtcdV2)Delete(res *etcd_client.Response) (err error) {
	if res.Node.Dir {
		for _, n := range (res.Node.Nodes) {
			err = etcdcli.DoDelete(n.Key, n.Value)
		}
		err = etcdcli.DeleteRaw(res.Node.Key)
	} else {
		err = etcdcli.DoDelete(res.Node.Key, res.Node.Value)
	}
	return err
}

func (etcdcli *EtcdV2)DeleteRaw(key string) error {
	delOptions := &etcd_client.DeleteOptions{
		Recursive : true,
	}

	_, err := etcdcli.keysAPI.Delete(context.TODO(), key, delOptions)

	return err
}

func (etcdcli *EtcdV2) Watch(prefix string, waitIndex uint64, recursive bool, receiver chan *etcd_client.Response) {
	//Watcher(key string, opts *WatcherOptions) Watcher
	go func() {
		for {
			if response, err := etcdcli.keysAPI.Watcher(prefix, &etcd_client.WatcherOptions{AfterIndex: waitIndex, Recursive: recursive}).Next(context.TODO()); err == nil {
				receiver <- response
			} else {
				logrus.Errorf("Get call-back response error when watch ", prefix, ", due to ", err.Error())
				time.Sleep(time.Second * 10)
			}
		}
	}()
}

