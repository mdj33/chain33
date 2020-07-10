package p2pstore

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (p *Protocol) getChunk(req *types.ChunkInfoMsg, queryRemote bool) (*types.BlockBodys, error) {
	if req == nil {
		return nil, types2.ErrInvalidParam
	}

	//优先获取本地p2pStore数据
	bodys, _ := p.getChunkBlock(req)
	if bodys != nil {
		return bodys, nil
	}

	if !queryRemote {
		return nil, types2.ErrNotFound
	}

	//本地数据不存在或已过期，则向临近节点查询
	return p.mustFetchChunk(req)
}

func (p *Protocol) getHeaders(param *types.ReqBlocks) *types.Headers {
	for _, pid := range p.P2PEnv.RoutingTable.RoutingTable().ListPeers() {
		headers, err := p.getHeadersFromPeer(param, pid)
		if err != nil {
			log.Error("getHeaders", "peer", pid, "error", err)
			continue
		}
		return headers
	}

	log.Error("getHeaders", "error", types2.ErrNotFound)
	return &types.Headers{}
}

func (p *Protocol) getHeadersFromPeer(param *types.ReqBlocks, pid peer.ID) (*types.Headers, error) {
	childCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stream, err := p.Host.NewStream(childCtx, pid, protocol.GetHeader)
	if err != nil {
		return nil, err
	}
	defer protocol.CloseStream(stream)
	msg := types.P2PRequest{
		Request: &types.P2PRequest_ReqBlocks{
			ReqBlocks: param,
		},
	}
	err = protocol.SignAndWriteStream(&msg, stream)
	if err != nil {
		log.Error("getHeadersFromPeer", "SignAndWriteStream error", err)
		return nil, err
	}
	var res types.P2PResponse
	err = protocol.ReadStreamAndAuthenticate(&res, stream)
	if err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	return res.Response.(*types.P2PResponse_BlockHeaders).BlockHeaders, nil
}

func (p *Protocol) getChunkRecords(param *types.ReqChunkRecords) *types.ChunkRecords {
	for _, prettyID := range param.Pid {
		pid, err := peer.Decode(prettyID)
		if err != nil {
			log.Error("getChunkRecords", "decode pid error", err)
			continue
		}
		records, err := p.getChunkRecordsFromPeer(param, pid)
		if err != nil {
			log.Error("getChunkRecords", "param peer", pid, "error", err, "start", param.Start, "end", param.End)
			continue
		}
		return records
	}

	for _, pid := range p.P2PEnv.RoutingTable.RoutingTable().ListPeers() {
		records, err := p.getChunkRecordsFromPeer(param, pid)
		if err != nil {
			log.Error("getChunkRecords", "peer", pid, "error", err, "start", param.Start, "end", param.End)
			continue
		}
		return records
	}

	log.Error("getChunkRecords", "error", types2.ErrNotFound)
	return nil
}

func (p *Protocol) getChunkRecordsFromPeer(param *types.ReqChunkRecords, pid peer.ID) (*types.ChunkRecords, error) {
	childCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stream, err := p.Host.NewStream(childCtx, pid, protocol.GetChunkRecord)
	if err != nil {
		return nil, err
	}
	defer protocol.CloseStream(stream)
	msg := types.P2PRequest{
		Request: &types.P2PRequest_ReqChunkRecords{
			ReqChunkRecords: param,
		},
	}
	err = protocol.SignAndWriteStream(&msg, stream)
	if err != nil {
		log.Error("getChunkRecordsFromPeer", "SignAndWriteStream error", err)
		return nil, err
	}

	var res types.P2PResponse
	err = protocol.ReadStreamAndAuthenticate(&res, stream)
	if err != nil {
		return nil, err
	}
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	return res.Response.(*types.P2PResponse_ChunkRecords).ChunkRecords, nil
}

//若网络中有节点保存了该chunk，该方法可以保证查询到
func (p *Protocol) mustFetchChunk(req *types.ChunkInfoMsg) (*types.BlockBodys, error) {
	//递归查询时间上限一小时
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	var retryCount int
Retry:
	//保存查询过的节点，防止重复查询
	searchedPeers := make(map[peer.ID]struct{})
	searchedPeers[p.Host.ID()] = struct{}{}
	peers := p.healthyRoutingTable.NearestPeers(genDHTID(req.ChunkHash), AlphaValue)
	if len(peers) == 0 {
		log.Error("mustFetchChunk", "error", "no healthy peers")
	}
	log.Info("into mustFetchChunk", "healthy peers len", p.healthyRoutingTable.Size())
	for len(peers) != 0 {
		var newPeers []peer.ID
		for _, pid := range peers {
			searchedPeers[pid] = struct{}{}
			start := time.Now()
			bodys, nearerPeers, err := p.fetchChunkOrNearerPeers(ctx, req, pid)
			log.Info("mustFetchChunk", "fetchChunkOrNearerPeers cost", time.Since(start))
			if err != nil {
				log.Error("mustFetchChunk", "fetchChunkOrNearerPeers error", err, "pid", pid, "chunk hash", hex.EncodeToString(req.ChunkHash), "maddrs", p.Host.Peerstore().Addrs(pid))
				continue
			}
			if bodys != nil {
				log.Info("mustFetchChunk found", "pid", pid, "maddrs", p.Host.Peerstore().Addrs(pid))
				return bodys, nil
			}
			newPeers = append(newPeers, nearerPeers...)
		}

		peers = nil
		for _, newPeer := range newPeers {
			//已经查询过的节点就不再查询
			if _, ok := searchedPeers[newPeer]; !ok {
				peers = append(peers, newPeer)
			}
		}
	}
	retryCount++
	//找不到数据重试5次，防止因为网络问题导致数据找不到
	//测试网和全节点不重试
	if !p.ChainCfg.IsTestNet() && !p.SubConfig.IsFullNode && retryCount < 5 {
		log.Error("mustFetchChunk", "retry count", retryCount)
		time.Sleep(p.retryInterval)
		goto Retry
	}
	log.Error("mustFetchChunk", "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "error", types2.ErrNotFound)
	//如果是分片节点没有在分片网络中找到数据，最后到全节点去请求数据
	if !p.SubConfig.IsFullNode {
		for _, pid := range p.fullNodes {
			bodys, _, err := p.fetchChunkOrNearerPeers(ctx, req, pid)
			if bodys == nil {
				log.Error("fetchChunkOrNearerPeers from full node failed", "pid", pid, "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start, "error", err)
				continue
			}
			log.Info("fetchChunkOrNearerPeers from full node succeed", "pid", pid, "chunk hash", hex.EncodeToString(req.ChunkHash), "start", req.Start)
			return bodys, nil
		}
	}
	return nil, types2.ErrNotFound
}

func (p *Protocol) fetchChunkOrNearerPeers(ctx context.Context, params *types.ChunkInfoMsg, pid peer.ID) (*types.BlockBodys, []peer.ID, error) {
	log.Info("into fetchChunkOrNearerPeers", "pid", pid)
	childCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	stream, err := p.Host.NewStream(childCtx, pid, protocol.FetchChunk)
	if err != nil {
		log.Error("fetchChunkOrNearerPeers", "error", err)
		return nil, nil, err
	}
	_ = stream.SetDeadline(time.Now().Add(time.Minute * 30))
	defer protocol.CloseStream(stream)
	msg := types.P2PRequest{
		Request: &types.P2PRequest_ChunkInfoMsg{
			ChunkInfoMsg: params,
		},
	}
	err = protocol.SignAndWriteStream(&msg, stream)
	if err != nil {
		log.Error("fetchChunkOrNearerPeers", "SignAndWriteStream error", err)
		return nil, nil, err
	}
	var res types.P2PResponse
	var result []byte
	buf := make([]byte, 1024*1024)
	t := time.Now()
	for {
		n, err := stream.Read(buf)
		result = append(result, buf[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}
	}
	log.Info("fetchChunkOrNearerPeers", "read data time cost", time.Since(t), "size", len(result))
	err = types.Decode(result, &res)
	if err != nil {
		return nil, nil, err
	}

	switch v := res.Response.(type) {
	case *types.P2PResponse_BlockBodys:
		log.Info("fetchChunkFromPeer", "remote", pid.Pretty(), "chunk hash", hex.EncodeToString(params.ChunkHash))
		return v.BlockBodys, nil, nil
	case *types.P2PResponse_AddrInfo:
		var addrInfos []peer.AddrInfo
		err = json.Unmarshal(v.AddrInfo, &addrInfos)
		if err != nil {
			log.Error("fetchChunkOrNearerPeers", "addrInfo error", err)
		}
		var peerList []peer.ID
		//如果对端节点返回了addrInfo，把节点信息加入到PeerStore，并返回节点id
		for _, addrInfo := range addrInfos {
			p.Host.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, time.Hour)
			peerList = append(peerList, addrInfo.ID)
		}
		return nil, peerList, nil
	}

	return nil, nil, errors.New(res.Error)
}

// 检查网络中是否能查到前一个chunk，最多往前查10个chunk，返回未保存的chunkInfo
func (p *Protocol) checkHistoryChunk(in *types.ChunkInfoMsg, queryRemote bool) []*types.ChunkInfoMsg {
	chunkLen := in.End - in.Start + 1
	req := &types.ReqChunkRecords{
		Start: in.Start/chunkLen - 10,
		End:   in.Start/chunkLen - 1,
	}
	if req.End < 0 {
		return nil
	}
	if req.Start < 0 {
		req.Start = 0
	}
	records, err := p.getChunkRecordFromBlockchain(req)
	if err != nil || records == nil {
		log.Error("checkHistoryChunk", "getChunkRecordFromBlockchain error", err, "start", req.Start, "end", req.End, "records", records)
		return nil
	}

	var res []*types.ChunkInfoMsg
	for i := len(records.Infos) - 1; i >= 0; i-- {
		//只检查chunk是否存在，因此为减少网络带宽消耗，只请求一个区块即可
		info := &types.ChunkInfoMsg{
			ChunkHash: records.Infos[i].ChunkHash,
			Start:     records.Infos[i].Start,
			End:       records.Infos[i].Start,
		}
		bodys, err := p.getChunk(info, queryRemote)
		if err == nil && bodys != nil {
			break
		}
		//网络中找不到上一个chunk,先把上一个chunk保存到本地p2pstore
		log.Debug("checkHistoryChunk", "chunk num", info.Start, "chunk hash", hex.EncodeToString(info.ChunkHash))
		info.End = records.Infos[i].End
		res = append(res, info)
	}
	return res
}

func (p *Protocol) storeChunk(req *types.ChunkInfoMsg) error {

	//如果p2pStore已保存数据，只更新时间即可
	if err := p.updateChunk(req); err == nil {
		return nil
	}
	//blockchain通知p2pStore保存数据，则blockchain应该有数据
	bodys, err := p.getChunkFromBlockchain(req)
	if err != nil {
		return err
	}
	return p.addChunkBlock(req, bodys)
}

func (p *Protocol) checkAndStoreChunk(req *types.ChunkInfoMsg, queryRemote bool) error {
	//先检查之前的chunk是否可以在网络中查到
	infos := p.checkHistoryChunk(req, queryRemote)
	infos = append(infos, req)
	var err error
	for _, info := range infos {
		if err = p.storeChunk(info); err != nil {
			log.Error("checkAndStoreChunk", "store chunk error", err, "chunkhash", hex.EncodeToString(info.ChunkHash), "start", info.Start)
			continue
		}
		//本地存储之后立即到其他节点做一次备份
		if queryRemote {
			p.notifyStoreChunk(req)
		}
	}

	return err
}

func (p *Protocol) getChunkFromBlockchain(param *types.ChunkInfoMsg) (*types.BlockBodys, error) {
	if param == nil {
		return nil, types2.ErrInvalidParam
	}
	msg := p.QueueClient.NewMessage("blockchain", types.EventGetChunkBlockBody, param)
	err := p.QueueClient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := p.QueueClient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if bodys, ok := resp.GetData().(*types.BlockBodys); ok {
		return bodys, nil
	}
	return nil, types2.ErrNotFound
}

func (p *Protocol) getChunkRecordFromBlockchain(req *types.ReqChunkRecords) (*types.ChunkRecords, error) {
	if req == nil {
		return nil, types2.ErrInvalidParam
	}
	msg := p.QueueClient.NewMessage("blockchain", types.EventGetChunkRecord, req)
	err := p.QueueClient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := p.QueueClient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if records, ok := resp.GetData().(*types.ChunkRecords); ok {
		return records, nil
	}

	return nil, types2.ErrNotFound
}

func (p *Protocol) getLastHeaderFromBlockChain() (*types.Header, error) {
	msg := p.QueueClient.NewMessage("blockchain", types.EventGetLastHeader, nil)
	err := p.QueueClient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	reply, err := p.QueueClient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if header, ok := reply.Data.(*types.Header); ok {
		return header, nil
	}
	return nil, types2.ErrNotFound
}
