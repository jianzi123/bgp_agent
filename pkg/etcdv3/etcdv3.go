package etcdv3

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
	"golang.org/x/net/context"
	"io/ioutil"
	"time"
)

type EtcdV3 struct {
	client *clientv3.Client
	ctx    context.Context
}

func (etcdcli *EtcdV3) ttlOpts(ctx context.Context, ttl int64) ([]clientv3.OpOption, error) {
	if ttl == 0 {
		return nil, nil
	}
	// put keys within into same lease. We shall benchmark this and optimize the performance.
	lcr, err := etcdcli.client.Lease.Grant(ctx, ttl)
	if err != nil {
		return nil, err
	}
	return []clientv3.OpOption{clientv3.WithLease(clientv3.LeaseID(lcr.ID))}, nil
}
func notFound(key string) clientv3.Cmp {
	return clientv3.Compare(clientv3.ModRevision(key), "=", 0)
}

func (etcdcli *EtcdV3)InitEtcd(etcdServerList []string, etcdCertfile, etcdKeyFile, etcdCafile string) error {

	cfg := clientv3.Config{
		Endpoints: etcdServerList,
		DialTimeout: time.Duration(5) * time.Millisecond,
	}

	tlsEnabled := false
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}

	if etcdCafile != "" {
		certBytes, err := ioutil.ReadFile(etcdCafile)
		if err != nil {
			return err
		}

		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(certBytes)

		if ok {
			tlsConfig.RootCAs = caCertPool
		}
		tlsEnabled = true
	}

	if etcdCertfile != "" && etcdKeyFile != "" {
		tlsCert, err := tls.LoadX509KeyPair(etcdCertfile, etcdKeyFile)
		if err != nil {
			return err
		}
		tlsConfig.Certificates = []tls.Certificate{tlsCert}
		tlsEnabled = true
	}

	if tlsEnabled {
		cfg.TLS = tlsConfig
	}

	etcdClient, err := clientv3.New(cfg)
	if err != nil {
		return err
	}
	etcdcli.client = etcdClient
	etcdcli.ctx = context.Background()
	return nil
}


// Get implements storage.Interface.Get.
func (etcdcli *EtcdV3) Get(key string, recursive bool) (*clientv3.GetResponse, error) {

	var getResp *clientv3.GetResponse
	var err error

	if recursive {
		getResp, err = etcdcli.client.KV.Get(etcdcli.ctx, key, clientv3.WithPrefix())
	} else {
		getResp, err = etcdcli.client.KV.Get(etcdcli.ctx, key)
	}

	if err != nil {
		return nil, err
	}
	if len(getResp.Kvs) == 0 {
		return nil, errors.New("key not found")
	}

	return getResp, nil
}

// Create implements storage.Interface.Creat
func (etcdcli *EtcdV3)Set(key string, val string) error {

	opts, err := etcdcli.ttlOpts(etcdcli.ctx, int64(0))
	if err != nil {
		return err
	}

	txnResp, err := etcdcli.client.KV.Txn(etcdcli.ctx).If(notFound(key),).Then(clientv3.OpPut(key, val, opts...),).Commit()
	if err != nil {
		return err
	}
	if !txnResp.Succeeded {
		return errors.New("key exists")
	}
	return nil
}

func (s *EtcdV3) Update(key string, val string) error {

	getResp, err := s.client.KV.Get(s.ctx, key)
	if err != nil {
		return err
	}
	for {
		opts, err := s.ttlOpts(s.ctx, int64(0))
		if err != nil {
			return err
		}

		txnResp, err := s.client.KV.Txn(s.ctx).If(
			clientv3.Compare(clientv3.ModRevision(key), "=", getResp.Kvs[0].ModRevision),
		).Then(
			clientv3.OpPut(key, val, opts...),
		).Else(
			clientv3.OpGet(key),
		).Commit()
		if err != nil {
			return err
		}
		if !txnResp.Succeeded {
			getResp = (*clientv3.GetResponse)(txnResp.Responses[0].GetResponseRange())
			logrus.Infof("GuaranteedUpdate of %s failed because of a conflict, going to retry", key)
			continue
		}
		return nil
	}
}

func (s *EtcdV3) DoDelete(key string) error {
	// We need to do get and delete in single transaction in order to
	// know the value and revision before deleting it.
	txnResp, err := s.client.KV.Txn(s.ctx).If().Then(
		clientv3.OpGet(key),
		clientv3.OpDelete(key),
	).Commit()
	if err != nil {
		return err
	}
	getResp := txnResp.Responses[0].GetResponseRange()
	if len(getResp.Kvs) == 0 {
		return errors.New("key not found")
	}
	return nil
}

func (s *EtcdV3) Delete(res *clientv3.GetResponse) error {

	for _, item := range res.Kvs {
		err := s.DoDelete(string(item.Key))
		if err != nil {
			logrus.Infof("%s\n", err.Error())
		}
		time.Sleep(20 * time.Microsecond)
	}
	return nil
}

func (s *EtcdV3)Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	return s.client.Watch(ctx, key, opts...)
}


func (e *EtcdV3) WatchPrefix(key string, keyChan, valueChan, typeChan chan string) {
	wch := e.Watch(context.Background(), key, clientv3.WithPrefix())
	for item := range wch {

		for _, ev := range item.Events{
			keyChan <- string(ev.Kv.Key)
			valueChan <- string(ev.Kv.Value)
			etype := fmt.Sprintf("%s",ev.Type)
			typeChan <- etype
			//fmt.Printf("%s %q:%q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
		}
	}
}
//// Watch for changes on a key
//func (s *EtcdV3) Watchv3(key string, stopCh <-chan struct{}) (<-chan *store.KVPair, error) {
//	watchCh := make(chan *store.KVPair)
//
//	go func() {
//		defer close(watchCh)
//
//		pair, err := s.Get(key)
//		if err != nil {
//			return
//		}
//		watchCh <- pair
//
//		rch := s.client.Watch(context.Background(), key)
//		for {
//			select {
//			case <-s.done:
//				return
//			case wresp := <-rch:
//				for _, event := range wresp.Events {
//					watchCh <- &store.KVPair{
//						Key:       string(event.Kv.Key),
//						Value:     event.Kv.Value,
//						LastIndex: uint64(event.Kv.Version),
//					}
//				}
//			}
//		}
//	}()
//
//	return watchCh, nil
//}
