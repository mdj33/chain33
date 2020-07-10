package healthy

import (
	"sync/atomic"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/network"
)

const (
	MaxQuery = 50
)

var log = log15.New("module", "protocol.healthy")

type Protocol struct {
	*protocol.P2PEnv //协议共享接口变量

	fallBehind int64 //落后多少高度，同步完成时该值应该为0
}

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

// InitProtocol initials healthy protocol.
func InitProtocol(env *protocol.P2PEnv) {
	p := Protocol{
		P2PEnv:     env,
		fallBehind: 1<<63 - 1,
	}
	p.Host.SetStreamHandler(protocol.IsSync, protocol.HandlerWithRW(p.HandleStreamIsSync))
	p.Host.SetStreamHandler(protocol.IsHealthy, protocol.HandlerWithRW(p.HandleStreamIsHealthy))
	p.Host.SetStreamHandler(protocol.GetLastHeader, protocol.HandlerWithRW(p.HandleStreamLastHeader))

	//保存一个全局变量备查，避免频繁到网络中请求。
	//全节点不参与分布式存储，因此不需要更新
	if !p.SubConfig.IsFullNode {
		go p.startUpdateFallBehind()
	}

}

// HandleStreamIsSync 实时查询是否已同步完成
func (p *Protocol) HandleStreamIsSync(_ *types.P2PRequest, res *types.P2PResponse, _ network.Stream) error {
	peers := p.Host.Network().Peers()
	shuffle(peers)

	maxHeight := int64(-1)
	var count int
	for _, pid := range peers {
		header, err := p.getLastHeaderFromPeer(pid)
		if err != nil {
			log.Error("HandleStreamIsSync", "getLastHeaderFromPeer error", err, "pid", pid)
			continue
		}
		if header.Height > maxHeight {
			maxHeight = header.Height
		}
		//最多访问50个节点，不包含请求失败的
		count++
		if count > MaxQuery {
			break
		}
	}

	if maxHeight == -1 {
		return types2.ErrUnknown
	}
	header, err := p.getLastHeaderFromBlockChain()
	if err != nil {
		log.Error("HandleStreamIsSync", "getLastHeaderFromBlockchain error", err)
		return types2.ErrUnknown
	}

	var isSync bool
	//本节点高度不小于临近节点则视为同步完成
	if header.Height >= maxHeight {
		isSync = true
	}
	res.Response = &types.P2PResponse_Reply{
		Reply: &types.Reply{
			IsOk: isSync,
		},
	}

	atomic.StoreInt64(&p.fallBehind, maxHeight-header.Height)
	return nil
}

// HandleStreamIsHealthy 非实时查询，定期更新
func (p *Protocol) HandleStreamIsHealthy(req *types.P2PRequest, res *types.P2PResponse, _ network.Stream) error {
	maxFallBehind := req.Request.(*types.P2PRequest_HealthyHeight).HealthyHeight

	var isHealthy bool
	if atomic.LoadInt64(&p.fallBehind) <= maxFallBehind {
		isHealthy = true
	}
	res.Response = &types.P2PResponse_Reply{
		Reply: &types.Reply{
			IsOk: isHealthy,
		},
	}
	log.Info("HandleStreamIsHealthy", "isHealthy", isHealthy)
	return nil
}

// HandleStreamLastHeader 获取节点最新高度
func (p *Protocol) HandleStreamLastHeader(_ *types.P2PRequest, res *types.P2PResponse, _ network.Stream) error {
	header, err := p.getLastHeaderFromBlockChain()
	if err != nil {
		return err
	}
	res.Response = &types.P2PResponse_LastHeader{
		LastHeader: header,
	}
	return nil
}
