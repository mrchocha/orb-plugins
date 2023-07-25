package etcd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	hash "github.com/mitchellh/hashstructure"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// This is here to make sure RegistryEtcd implements registry.Registry.
var _ registry.Registry = (*RegistryEtcd)(nil)

var (
	prefix = "/micro/registry/"
)

// RegistryEtcd implements the registry interface. It runs a etcd service registry.
type RegistryEtcd struct {
	client *clientv3.Client
	config Config
	logger log.Logger

	sync.RWMutex
	register map[string]uint64
	leases   map[string]clientv3.LeaseID
}

func ProvideRegistryEtcd(
	name types.ServiceName,
	data types.ConfigData,
	logger log.Logger,
	opts ...registry.Option,
) (registry.Type, error) {
	cfg, err := NewConfig(name, data, opts...)
	if err != nil {
		return registry.Type{}, fmt.Errorf("create etcd registry config: %w", err)
	}

	logger, err = logger.WithComponent(registry.ComponentType, Name, "", nil)
	if err != nil {
		return registry.Type{}, err
	}

	cfg.Logger = logger

	// Return the new registry.
	reg := New(cfg, logger)

	return registry.Type{Registry: reg}, nil
}

func New(cfg Config, log log.Logger) *RegistryEtcd {
	if cfg.Timeout == 0 {
		cfg.Timeout = registry.DefaultTimeout
	}

	username, password := os.Getenv("ETCD_USERNAME"), os.Getenv("ETCD_PASSWORD")
	if len(username) > 0 && len(password) > 0 {
		WithAuth(username, password) // TODO: check if this is correct
	}
	address := os.Getenv("MICRO_REGISTRY_ADDRESS")
	if len(address) > 0 {
		WithAddress(address)
	}

	clientV3Config := clientv3.Config{
		Endpoints: []string{"127.0.0.1:2379"},
	}

	clientV3Config.DialTimeout = time.Duration(cfg.Timeout)

	defaultTLSConfig := tls.Config{
		InsecureSkipVerify: true,
	}

	if cfg.Secure || cfg.TLSConfig != nil {
		tlsConfig := cfg.TLSConfig
		if tlsConfig == nil {
			tlsConfig = &defaultTLSConfig
		}
		clientV3Config.TLS = tlsConfig
	}

	if cfg.Auth != nil {
		clientV3Config.Username = cfg.Auth.Username
		clientV3Config.Password = cfg.Auth.Password
	}

	if cfg.LogConfig != nil {
		clientV3Config.LogConfig = cfg.LogConfig
	}

	var cAddrs []string
	for _, address := range cfg.Addresses {
		if len(address) == 0 {
			continue
		}
		addr, port, err := net.SplitHostPort(address)
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			port = "2379"
			addr = address
			cAddrs = append(cAddrs, net.JoinHostPort(addr, port))
		} else if err == nil {
			cAddrs = append(cAddrs, net.JoinHostPort(addr, port))
		}
	}

	// if we got addrs then we'll update
	if len(cAddrs) > 0 {
		clientV3Config.Endpoints = cAddrs
	}

	cli, _ := clientv3.New(clientV3Config)
	// TODO: ASK should we return error?
	// if err != nil {
	// 	return err
	// }

	return &RegistryEtcd{
		config:   cfg,
		logger:   log,
		client:   cli,
		register: make(map[string]uint64),
		leases:   make(map[string]clientv3.LeaseID),
	}
}

func encode(s *registry.Service) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func decode(ds []byte) *registry.Service {
	var s *registry.Service
	json.Unmarshal(ds, &s)
	return s
}

func nodePath(s, id string) string {
	service := strings.Replace(s, "/", "-", -1)
	node := strings.Replace(id, "/", "-", -1)
	return path.Join(prefix, service, node)
}

func servicePath(s string) string {
	return path.Join(prefix, strings.Replace(s, "/", "-", -1))
}

func (e *RegistryEtcd) Start() error {
	// TODO: ASK what logic should go here?
	return nil
}

func (e *RegistryEtcd) Stop(_ context.Context) error {
	e.client.Close()
	return nil
}

func (e *RegistryEtcd) registerNode(s *registry.Service, node *registry.Node, opts ...registry.RegisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

	// check existing lease cache
	e.RLock()
	leaseID, ok := e.leases[s.Name+node.ID]
	e.RUnlock()

	log := e.logger

	if !ok {
		// missing lease, check if the key exists
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.config.Timeout))
		defer cancel()

		// look for the existing key
		rsp, err := e.client.Get(ctx, nodePath(s.Name, node.ID), clientv3.WithSerializable())
		if err != nil {
			return err
		}

		// get the existing lease
		for _, kv := range rsp.Kvs {
			if kv.Lease > 0 {
				leaseID = clientv3.LeaseID(kv.Lease)

				// decode the existing node
				srv := decode(kv.Value)
				if srv == nil || len(srv.Nodes) == 0 {
					continue
				}

				// create hash of service; uint64
				h, err := hash.Hash(srv.Nodes[0], nil)
				if err != nil {
					continue
				}

				// save the info
				e.Lock()
				e.leases[s.Name+node.ID] = leaseID
				e.register[s.Name+node.ID] = h
				e.Unlock()

				break
			}
		}
	}

	var leaseNotFound bool

	// renew the lease if it exists
	if leaseID > 0 {
		// TODO: ASK if this is correct ? what about logger.TraceLevel similar in other places
		log.Error("Renewing existing lease for %s %d", s.Name, leaseID)
		if _, err := e.client.KeepAliveOnce(context.TODO(), leaseID); err != nil {
			if err != rpctypes.ErrLeaseNotFound {
				return err
			}

			log.Error("Lease not found for %s %d", s.Name, leaseID)
			// lease not found do register
			leaseNotFound = true
		}
	}

	// create hash of service; uint64
	h, err := hash.Hash(node, nil)
	if err != nil {
		return err
	}

	// get existing hash for the service node
	e.Lock()
	v, ok := e.register[s.Name+node.ID]
	e.Unlock()

	// the service is unchanged, skip registering
	if ok && v == h && !leaseNotFound {
		log.Error("Service %s node %s unchanged skipping registration", s.Name, node.ID)
		return nil
	}

	service := &registry.Service{
		Name:      s.Name,
		Version:   s.Version,
		Metadata:  s.Metadata,
		Endpoints: s.Endpoints,
		Nodes:     []*registry.Node{node},
	}

	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.config.Timeout))
	defer cancel()

	var lgr *clientv3.LeaseGrantResponse
	if options.TTL.Seconds() > 0 {
		// get a lease used to expire keys since we have a ttl
		lgr, err = e.client.Grant(ctx, int64(options.TTL.Seconds()))
		if err != nil {
			return err
		}
	}

	log.Error("Registering %s id %s with lease %v and leaseID %v and ttl %v", service.Name, node.ID, lgr, lgr.ID, options.TTL)
	// create an entry for the node
	if lgr != nil {
		_, err = e.client.Put(ctx, nodePath(service.Name, node.ID), encode(service), clientv3.WithLease(lgr.ID))
	} else {
		_, err = e.client.Put(ctx, nodePath(service.Name, node.ID), encode(service))
	}
	if err != nil {
		return err
	}

	e.Lock()
	// save our hash of the service
	e.register[s.Name+node.ID] = h
	// save our leaseID of the service
	if lgr != nil {
		e.leases[s.Name+node.ID] = lgr.ID
	}
	e.Unlock()

	return nil
}

func (e *RegistryEtcd) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

	var gerr error

	// register each node individually
	for _, node := range s.Nodes {
		err := e.registerNode(s, node, opts...)
		if err != nil {
			gerr = err
		}
	}

	return gerr
}

func (e *RegistryEtcd) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

	for _, node := range s.Nodes {
		e.Lock()
		// delete our hash of the service
		delete(e.register, s.Name+node.ID)
		// delete our lease of the service
		delete(e.leases, s.Name+node.ID)
		e.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.config.Timeout))
		defer cancel()

		e.config.Logger.Error("Deregistering %s id %s", s.Name, node.ID)
		_, err := e.client.Delete(ctx, nodePath(s.Name, node.ID))
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *RegistryEtcd) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.config.Timeout))
	defer cancel()

	rsp, err := e.client.Get(ctx, servicePath(name)+"/", clientv3.WithPrefix(), clientv3.WithSerializable())
	if err != nil {
		return nil, err
	}

	if len(rsp.Kvs) == 0 {
		return nil, registry.ErrNotFound
	}

	serviceMap := map[string]*registry.Service{}

	for _, n := range rsp.Kvs {
		if sn := decode(n.Value); sn != nil {
			s, ok := serviceMap[sn.Version]
			if !ok {
				s = &registry.Service{
					Name:      sn.Name,
					Version:   sn.Version,
					Metadata:  sn.Metadata,
					Endpoints: sn.Endpoints,
				}
				serviceMap[s.Version] = s
			}

			s.Nodes = append(s.Nodes, sn.Nodes...)
		}
	}

	services := make([]*registry.Service, 0, len(serviceMap))
	for _, service := range serviceMap {
		services = append(services, service)
	}

	return services, nil
}

func (e *RegistryEtcd) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	versions := make(map[string]*registry.Service)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(e.config.Timeout))
	defer cancel()

	rsp, err := e.client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithSerializable())
	if err != nil {
		return nil, err
	}

	if len(rsp.Kvs) == 0 {
		return []*registry.Service{}, nil
	}

	for _, n := range rsp.Kvs {
		sn := decode(n.Value)
		if sn == nil {
			continue
		}
		v, ok := versions[sn.Name+sn.Version]
		if !ok {
			versions[sn.Name+sn.Version] = sn
			continue
		}
		// append to service:version nodes
		v.Nodes = append(v.Nodes, sn.Nodes...)
	}

	services := make([]*registry.Service, 0, len(versions))
	for _, service := range versions {
		services = append(services, service)
	}

	// sort the services
	sort.Slice(services, func(i, j int) bool { return services[i].Name < services[j].Name })

	return services, nil
}

func (e *RegistryEtcd) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	// TODO: add this method
	// return newEtcdWatcher(e, e..Timeout, opts...)
	return nil, nil
}

func (e *RegistryEtcd) String() string {
	return "etcd"
}

func (e *RegistryEtcd) Type() string {
	return registry.ComponentType
}
