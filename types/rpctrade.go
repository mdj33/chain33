package types

import "encoding/json"

var RpcTradeTypeTransList = []RpcTypeInfo{
	{
		"GetTokenSellOrderByStatus",
		TradeQueryTokenSellOrder{},
	},
	{
		"GetOnesSellOrderWithStatus",
		TradeQueryOnesSellOrder{},
	},
	{
		"GetOnesSellOrder",
		TradeQueryOnesSellOrder{},
	},

	{
		"GetTokenBuyOrderByStatus",
		TradeQueryTokenBuyOrder{},
	},
	{
		"GetOnesBuyOrderWithStatus",
		TradeQueryOnesBuyOrder{},
	},
	{
		"GetOnesBuyOrder",
		TradeQueryOnesBuyOrder{},
	},
}

// rpc query trade sell order part

type RpcReplySellOrder struct {
	TokenSymbol       string `json:"tokenSymbol"`
	Owner             string `json:"owner"`
	AmountPerBoardlot int64  `json:"amountPerBoardlot"`
	MinBoardlot       int64  `json:"minBoardlot"`
	PricePerBoardlot  int64  `json:"pricePerBoardlot"`
	TotalBoardlot     int64  `json:"totalBoardlot"`
	SoldBoardlot      int64  `json:"soldBoardlot"`
	BuyID             string `json:"buyID"`
	Status            int32  `json:"status"`
	SellID            string `json:"sellID"`
	TxHash            string `json:"txHash"`
	Height            int64  `json:"height"`
	Key               string `json:"key"`
}
type RpcReplySellOrders struct {
	SellOrders []*RpcReplySellOrder `json:"sellOrders"`
}

type TradeQueryTokenSellOrder struct {
}

func (t *TradeQueryTokenSellOrder) Input(message json.RawMessage) ([]byte, error) {
	var req ReqTokenSellOrder
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return Encode(&req), nil
}

func (t *TradeQueryTokenSellOrder) Output(reply *ReplySellOrders) (*RpcReplySellOrders, error) {
	str, err := json.Marshal(*reply)
	if err != nil {
		return nil, err
	}
	var rpcReply RpcReplySellOrders
	json.Unmarshal(str, &rpcReply)
	return &rpcReply, nil
}

type TradeQueryOnesSellOrder struct {
}

func (t *TradeQueryOnesSellOrder) Input(message json.RawMessage) ([]byte, error) {
	var req ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return Encode(&req), nil
}

func (t *TradeQueryOnesSellOrder) Output(reply *ReplySellOrders) (*RpcReplySellOrders, error) {
	str, err := json.Marshal(*reply)
	if err != nil {
		return nil, err
	}
	var rpcReply RpcReplySellOrders
	json.Unmarshal(str, &rpcReply)
	return &rpcReply, nil
}

// rpc query trade buy order

type RpcReplyBuyOrder struct {
	TokenSymbol       string `json:"tokenSymbol"`
	Owner             string `json:"owner"`
	AmountPerBoardlot int64  `json:"amountPerBoardlot"`
	MinBoardlot       int64  `json:"minBoardlot"`
	PricePerBoardlot  int64  `json:"pricePerBoardlot"`
	TotalBoardlot     int64  `json:"totalBoardlot"`
	BoughtBoardlot    int64  `json:"boughtBoardlot"`
	BuyID             string `json:"buyID"`
	Status            int32  `json:"status"`
	SellID            string `json:"sellID"`
	TxHash            string `json:"txHash"`
	Height            int64  `json:"height"`
	Key               string `json:"key"`
}

type RpcReplyBuyOrders struct {
	BuyOrders []*RpcReplyBuyOrder `json:"buyOrders"`
}

type TradeQueryTokenBuyOrder struct {
}

func (t *TradeQueryTokenBuyOrder) Input(message json.RawMessage) ([]byte, error) {
	var req ReqTokenBuyOrder
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return Encode(&req), nil
}

func (t *TradeQueryTokenBuyOrder) Output(reply *ReplyBuyOrders) (*RpcReplyBuyOrders, error) {
	str, err := json.Marshal(*reply)
	if err != nil {
		return nil, err
	}
	var rpcReply RpcReplyBuyOrders
	json.Unmarshal(str, &rpcReply)
	return &rpcReply, nil
}

type TradeQueryOnesBuyOrder struct {
}

func (t *TradeQueryOnesBuyOrder) Input(message json.RawMessage) ([]byte, error) {
	var req ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return Encode(&req), nil
}

func (t *TradeQueryOnesBuyOrder) Output(reply *ReplyBuyOrders) (*RpcReplyBuyOrders, error) {
	str, err := json.Marshal(*reply)
	if err != nil {
		return nil, err
	}
	var rpcReply RpcReplyBuyOrders
	json.Unmarshal(str, &rpcReply)
	return &rpcReply, nil
}

// trade order
type ReplyTradeOrder struct {
	TokenSymbol       string `protobuf:"bytes,1,opt,name=tokenSymbol" json:"tokenSymbol,omitempty"`
	Owner             string `protobuf:"bytes,2,opt,name=owner" json:"owner,omitempty"`
	AmountPerBoardlot int64  `protobuf:"varint,3,opt,name=amountPerBoardlot" json:"amountPerBoardlot,omitempty"`
	MinBoardlot       int64  `protobuf:"varint,4,opt,name=minBoardlot" json:"minBoardlot,omitempty"`
	PricePerBoardlot  int64  `protobuf:"varint,5,opt,name=pricePerBoardlot" json:"pricePerBoardlot,omitempty"`
	TotalBoardlot     int64  `protobuf:"varint,6,opt,name=totalBoardlot" json:"totalBoardlot,omitempty"`
	TradedBoardlot    int64  `protobuf:"varint,7,opt,name=tradedBoardlot" json:"tradedBoardlot,omitempty"`
	BuyID             string `protobuf:"bytes,8,opt,name=buyID" json:"buyID,omitempty"`
	Status            int32  `protobuf:"varint,9,opt,name=status" json:"status,omitempty"`
	SellID            string `protobuf:"bytes,10,opt,name=sellID" json:"sellID,omitempty"`
	TxHash            string `protobuf:"bytes,11,opt,name=txHash" json:"txHash,omitempty"`
	Height            int64  `protobuf:"varint,12,opt,name=height" json:"height,omitempty"`
	Key               string `protobuf:"bytes,13,opt,name=key" json:"key,omitempty"`
	BlockTime         int64  `protobuf:"varint,14,opt,name=blockTime" json:"blockTime,omitempty"`
	IsSellOrder       bool   `protobuf:"varint,15,opt,name=isSellOrder" json:"isSellOrder,omitempty"`
}

type ReplyTradeOrders struct {
	Orders []*ReplyTradeOrder `protobuf:"bytes,1,rep,name=orders" json:"orders,omitempty"`
}
