package pubsub

import (
	"cmp"
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
	"github.com/yydsqu/tools/log"
	"os"
	"path/filepath"
	"time"
)

const (
	ProtocolPrefix = "/private_net/pub_sub/1.0.0"
	NS             = "pub-sub-service"
)

var (
	ListenAddr = []string{
		"/ip4/0.0.0.0/tcp/10001",
		"/ip4/0.0.0.0/udp/10001/quic-v1",
	}
)

type Config struct {
	Identity  string   `json:"identity" toml:"identity" yaml:"identity"`
	Listen    []string `json:"listen" toml:"listen" yaml:"listen"`
	Bootstrap []string `json:"bootstrap" toml:"bootstrap" yaml:"bootstrap"`
}

func (conf *Config) ListenAddr() []string {
	if len(conf.Listen) == 0 {
		return ListenAddr
	}
	return conf.Listen
}

func (conf *Config) BootstrapNode() ([]peer.AddrInfo, error) {
	var (
		addrInfo []peer.AddrInfo
		err      error
	)
	for _, addr := range conf.Bootstrap {
		var boot multiaddr.Multiaddr
		if boot, err = multiaddr.NewMultiaddr(addr); err != nil {
			return addrInfo, err
		}
		var peerInfo *peer.AddrInfo
		if peerInfo, err = peer.AddrInfoFromP2pAddr(boot); err != nil {
			return addrInfo, err
		}
		addrInfo = append(addrInfo, *peerInfo)
	}
	return addrInfo, nil
}

func LoadOrGenerateKey(path string) (crypto.PrivKey, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		priv, _, _ := crypto.GenerateKeyPair(crypto.Ed25519, -1)
		raw, _ := crypto.MarshalPrivateKey(priv)
		os.MkdirAll(filepath.Dir(path), 0700)
		os.WriteFile(path, raw, 0600)
		return priv, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}
	return crypto.UnmarshalPrivateKey(raw)
}

type PubSub struct {
	ctx       context.Context
	conf      *Config
	logger    log.Logger
	identity  crypto.PrivKey
	bootstrap []peer.AddrInfo
	discovery *routing.RoutingDiscovery
	host      host.Host
	pubSub    *pubsub.PubSub
	dht       *dht.IpfsDHT
}

func (pubSub *PubSub) Listen(n network.Network, multiaddr multiaddr.Multiaddr) {
	pubSub.logger.Debug("listening for incoming connections", "addr", multiaddr.String())
}

func (pubSub *PubSub) ListenClose(n network.Network, multiaddr multiaddr.Multiaddr) {
	pubSub.logger.Debug("closing incoming connections", "addr", multiaddr.String())
}

func (pubSub *PubSub) Connected(n network.Network, conn network.Conn) {
	pubSub.host.Peerstore().AddAddrs(conn.RemotePeer(), []multiaddr.Multiaddr{conn.RemoteMultiaddr()}, peerstore.PermanentAddrTTL)
	pubSub.logger.Debug("connected", "peer", conn.RemotePeer(), "count", len(n.Peers()))
}

func (pubSub *PubSub) Disconnected(n network.Network, conn network.Conn) {
	pubSub.logger.Debug("disconnected", "peer", conn.RemotePeer(), "count", len(n.Peers()))
}

func (pubSub *PubSub) Host() host.Host {
	return pubSub.host
}

func (pubSub *PubSub) PubSub() *pubsub.PubSub {
	return pubSub.pubSub
}

func (pubSub *PubSub) Self() peer.ID {
	return pubSub.host.ID()
}

func (pubSub *PubSub) Addr() []string {
	var addrs []string
	for _, addr := range pubSub.host.Addrs() {
		addrs = append(addrs, fmt.Sprintf(`%s/p2p/%s`, addr, pubSub.host.ID()))
	}
	return addrs
}

func (pubSub *PubSub) Topic(topic string, opts ...pubsub.TopicOpt) (*pubsub.Topic, error) {
	return pubSub.pubSub.Join(topic, opts...)
}

func (pubSub *PubSub) Connect(ctx context.Context, pi peer.AddrInfo) error {
	if pi.ID == pubSub.host.ID() {
		return nil
	}
	return pubSub.host.Connect(ctx, pi)
}

func (pubSub *PubSub) ConnectBootstrap() {
	if len(pubSub.bootstrap) == 0 {
		return
	}
	for _, addr := range pubSub.bootstrap {
		if err := pubSub.Connect(pubSub.ctx, addr); err != nil {
			pubSub.logger.Trace("connect peer failure", "id", addr.ID.String())
		}
		pubSub.host.ConnManager().Protect(addr.ID, "bootstrap")
	}
}

func (pubSub *PubSub) discoverPeers() {
	ctx, cancel := context.WithTimeout(pubSub.ctx, time.Second*30)
	defer cancel()
	pubSub.logger.Trace("starting discovery")
	if len(pubSub.host.Network().Peers()) == 0 {
		pubSub.ConnectBootstrap()
	}
	util.Advertise(ctx, pubSub.discovery, NS)
	// 寻找节点
	peerChan, err := pubSub.discovery.FindPeers(ctx, NS)
	if err != nil {
		pubSub.logger.Warn("find peers failure", "err", err)
		return
	}

	for info := range peerChan {
		// 排除自己或空地址
		if info.ID == pubSub.host.ID() || len(info.Addrs) == 0 {
			continue
		}
		if pubSub.host.Network().Connectedness(info.ID) == network.Connected {
			continue
		}
		if err = pubSub.host.Connect(ctx, info); err != nil {
			pubSub.logger.Trace("failed to connect discovered peer", "id", info.ID)
			continue
		}
		pubSub.logger.Trace("discovered and connected to peer", "id", info.ID)
	}
}

func (pubSub *PubSub) Start() error {
	var (
		ticker = time.NewTicker(time.Second * 30)
		err    error
	)
	defer ticker.Stop()
	pubSub.ConnectBootstrap()
	if err = pubSub.dht.Bootstrap(pubSub.ctx); err != nil {
		return err
	}
	for {
		select {
		case <-pubSub.ctx.Done():
			return nil
		case <-ticker.C:
			go pubSub.discoverPeers()
		}
	}
}

func NewPubSub(ctx context.Context, logger log.Logger, conf *Config) (*PubSub, error) {
	var (
		pub = &PubSub{
			ctx:    ctx,
			conf:   conf,
			logger: logger,
		}
		err error
	)
	if pub.identity, err = LoadOrGenerateKey(cmp.Or(conf.Identity, "identity")); err != nil {
		return nil, fmt.Errorf("load key failure: %w", err)
	}
	if pub.bootstrap, err = conf.BootstrapNode(); err != nil {
		return nil, fmt.Errorf("bootstrap node failure: %w", err)
	}
	if pub.host, err = libp2p.New(libp2p.Identity(pub.identity), libp2p.ListenAddrStrings(conf.ListenAddr()...), libp2p.DefaultTransports); err != nil {
		return nil, fmt.Errorf("create p2p host failure: %w", err)
	}
	if pub.dht, err = dht.New(ctx, pub.host, dht.Mode(dht.ModeServer), dht.ProtocolPrefix(ProtocolPrefix)); err != nil {
		return nil, fmt.Errorf("create dht failure: %w", err)
	}
	pub.discovery = routing.NewRoutingDiscovery(pub.dht)
	if pub.pubSub, err = pubsub.NewFloodSub(ctx, pub.host, pubsub.WithDiscovery(pub.discovery)); err != nil {
		return nil, fmt.Errorf("create pubsub failure: %w", err)
	}
	pub.host.Network().Notify(pub)
	return pub, err
}
