package pub

import (
	"fmt"

	"github.com/linkedin/goavro"

	orderPkg "github.com/BiJie/BinanceChain/plugins/dex/order"
)

var (
	booksCodec            *goavro.Codec
	accountCodec          *goavro.Codec
	tradeseAndOrdersCodec *goavro.Codec
)

type msgType int8

const (
	accountsTpe = iota
	booksTpe
	tradesAndOrdersTpe
)

// the strings should be keep consistence with top level record name in schemas.go
func (this msgType) String() string {
	switch this {
	case accountsTpe:
		return "Accounts"
	case booksTpe:
		return "Books"
	case tradesAndOrdersTpe:
		return "TradesAndOrders"
	default:
		return "Unknown"
	}
}

type AvroMsg interface {
	ToNativeMap() map[string]interface{}
	String() string
}

func marshal(msg AvroMsg, tpe msgType) ([]byte, error) {
	native := msg.ToNativeMap()
	var codec *goavro.Codec
	switch tpe {
	case accountsTpe:
		codec = accountCodec
	case booksTpe:
		codec = booksCodec
	case tradesAndOrdersTpe:
		codec = tradeseAndOrdersCodec
	default:
		return nil, fmt.Errorf("doesn't support marshal kafka msg tpe: %s", tpe.String())
	}
	bb, err := codec.BinaryFromNative(nil, native)
	if err != nil {
		Logger.Error("failed to serialize message", "msg", msg, "err", err)
	}
	return bb, err
}

type tradesAndOrders struct {
	height    int64
	timestamp int64 // milli seconds since Epoch
	numOfMsgs int   // number of individual messages we published, consumer can verify messages they received against this field to make sure they does not miss messages
	trades    trades
	orders    orders
}

func (msg *tradesAndOrders) String() string {
	return fmt.Sprintf("TradesAndOrders at height: %d, numOfMsgs: %d", msg.height, msg.numOfMsgs)
}

func (msg *tradesAndOrders) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["height"] = msg.height
	native["timestamp"] = msg.timestamp
	native["numOfMsgs"] = msg.numOfMsgs
	if msg.trades.numOfMsgs > 0 {
		native["trades"] = map[string]interface{}{"org.binance.dex.model.avro.Trades": msg.trades.ToNativeMap()}
	}
	if msg.orders.numOfMsgs > 0 {
		native["orders"] = map[string]interface{}{"org.binance.dex.model.avro.Orders": msg.orders.ToNativeMap()}
	}
	return native
}

type trades struct {
	numOfMsgs int
	trades    []*Trade
}

func (msg *trades) String() string {
	return fmt.Sprintf("Trades numOfMsgs: %d", msg.numOfMsgs)
}

func (msg *trades) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["numOfMsgs"] = msg.numOfMsgs
	ts := make([]map[string]interface{}, len(msg.trades), len(msg.trades))
	for idx, trade := range msg.trades {
		ts[idx] = trade.toNativeMap()
	}
	native["trades"] = ts
	return native
}

type Trade struct {
	Id     string
	Symbol string
	Price  int64
	Qty    int64
	Sid    string
	Bid    string
	Sfee   string
	Bfee   string
}

func (msg *Trade) String() string {
	return fmt.Sprintf("Trade: %v", msg.toNativeMap())
}

func (msg *Trade) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["id"] = msg.Id
	native["symbol"] = msg.Symbol
	native["price"] = msg.Price
	native["qty"] = msg.Qty
	native["sid"] = msg.Sid
	native["bid"] = msg.Bid
	native["sfee"] = msg.Sfee
	native["bfee"] = msg.Bfee
	return native
}

type orders struct {
	numOfMsgs int
	orders    []order
}

func (msg *orders) String() string {
	return fmt.Sprintf("Orders numOfMsgs: %d", msg.numOfMsgs)
}

func (msg *orders) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["numOfMsgs"] = msg.numOfMsgs
	os := make([]map[string]interface{}, len(msg.orders), len(msg.orders))
	for idx, o := range msg.orders {
		os[idx] = o.toNativeMap()
	}
	native["orders"] = os
	return native
}

type order struct {
	symbol               string
	status               orderPkg.ChangeType
	orderId              string
	tradeId              string
	owner                string
	side                 int8
	orderType            int8
	price                int64
	qty                  int64
	lastExecutedPrice    int64
	lastExecutedQty      int64
	cumQty               int64
	fee                  string
	orderCreationTime    int64
	transactionTime      int64
	timeInForce          int8
	currentExecutionType orderPkg.ExecutionType
	txHash               string
}

func (msg *order) String() string {
	return fmt.Sprintf("Order: %v", msg.toNativeMap())
}

func (msg *order) effectQtyToOrderBook() int64 {
	switch msg.status {
	case orderPkg.Ack:
		return msg.qty
	case orderPkg.FullyFill, orderPkg.PartialFill:
		return -msg.lastExecutedQty
	case orderPkg.Expired, orderPkg.IocNoFill, orderPkg.Canceled:
		return msg.cumQty - msg.qty // deliberated be negative value
	default:
		Logger.Error("does not supported order status", "order", msg.String())
		return 0
	}
}

func (msg *order) toNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["symbol"] = msg.symbol
	native["status"] = msg.status.String() //TODO(#66): confirm with all teams to make this uint8 enum
	native["orderId"] = msg.orderId
	native["tradeId"] = msg.tradeId
	native["owner"] = msg.owner
	native["side"] = orderPkg.IToSide(msg.side)                //TODO(#66): confirm with all teams to make this uint8 enum
	native["orderType"] = orderPkg.IToOrderType(msg.orderType) //TODO(#66): confirm with all teams to make this uint8 enum
	native["price"] = msg.price
	native["qty"] = msg.qty
	native["lastExecutedPrice"] = msg.lastExecutedPrice
	native["lastExecutedQty"] = msg.lastExecutedQty
	native["cumQty"] = msg.cumQty
	native["fee"] = msg.fee
	native["orderCreationTime"] = msg.orderCreationTime
	native["transactionTime"] = msg.transactionTime
	native["timeInForce"] = orderPkg.IToTimeInForce(msg.timeInForce)   //TODO(#66): confirm with all teams to make this uint8 enum
	native["currentExecutionType"] = msg.currentExecutionType.String() //TODO(#66): confirm with all teams to make this uint8 enum
	native["txHash"] = msg.txHash
	return native
}

type PriceLevel struct {
	Price   int64
	LastQty int64
}

func (msg *PriceLevel) String() string {
	return fmt.Sprintf("priceLevel: %s", msg.ToNativeMap())
}

func (msg *PriceLevel) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["price"] = msg.Price
	native["lastQty"] = msg.LastQty
	return native
}

type OrderBookDelta struct {
	Symbol string
	Buys   []PriceLevel
	Sells  []PriceLevel
}

func (msg *OrderBookDelta) String() string {
	return fmt.Sprintf("orderBookDelta for: %s, num of buys prices: %d, num of sell prices: %d", msg.Symbol, len(msg.Buys), len(msg.Sells))
}

func (msg *OrderBookDelta) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["symbol"] = msg.Symbol
	bs := make([]map[string]interface{}, len(msg.Buys), len(msg.Buys))
	for idx, buy := range msg.Buys {
		bs[idx] = buy.ToNativeMap()
	}
	native["buys"] = bs
	ss := make([]map[string]interface{}, len(msg.Sells), len(msg.Sells))
	for idx, sell := range msg.Sells {
		ss[idx] = sell.ToNativeMap()
	}
	native["sells"] = ss
	return native
}

type Books struct {
	Height    int64
	Timestamp int64
	NumOfMsgs int
	Books     []OrderBookDelta
}

func (msg *Books) String() string {
	return fmt.Sprintf("Books at height: %d, numOfMsgs: %d", msg.Height, msg.NumOfMsgs)
}

func (msg *Books) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["height"] = msg.Height
	native["timestamp"] = msg.Timestamp
	native["numOfMsgs"] = msg.NumOfMsgs
	if msg.NumOfMsgs > 0 {
		bs := make([]map[string]interface{}, len(msg.Books), len(msg.Books))
		for idx, book := range msg.Books {
			bs[idx] = book.ToNativeMap()
		}
		native["books"] = bs
	}
	return native
}

type AssetBalance struct {
	Asset  string
	Free   int64
	Frozen int64
	Locked int64
}

func (msg *AssetBalance) String() string {
	return fmt.Sprintf("AssetBalance: %s", msg.ToNativeMap())
}

func (msg *AssetBalance) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["asset"] = msg.Asset
	native["free"] = msg.Free
	native["frozen"] = msg.Frozen
	native["locked"] = msg.Locked
	return native
}

type Account struct {
	Owner    string
	Balances []*AssetBalance
}

func (msg *Account) String() string {
	return fmt.Sprintf("Account of: %s, total kind of balance changes: %d", msg.Owner, len(msg.Balances))
}

func (msg *Account) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["owner"] = msg.Owner
	bs := make([]map[string]interface{}, len(msg.Balances), len(msg.Balances))
	for idx, b := range msg.Balances {
		bs[idx] = b.ToNativeMap()
	}
	native["balances"] = bs
	return native
}

type accounts struct {
	height    int64
	numOfMsgs int
	Accounts  []Account
}

func (msg *accounts) String() string {
	return fmt.Sprintf("Accounts at height: %d, numOfMsgs: %d", msg.height, msg.numOfMsgs)
}

func (msg *accounts) ToNativeMap() map[string]interface{} {
	var native = make(map[string]interface{})
	native["height"] = msg.height
	native["numOfMsgs"] = msg.numOfMsgs
	if msg.numOfMsgs > 0 {
		as := make([]map[string]interface{}, len(msg.Accounts), len(msg.Accounts))
		for idx, a := range msg.Accounts {
			as[idx] = a.ToNativeMap()
		}
		native["accounts"] = as
	}
	return native
}

func initAvroCodecs() (res error) {
	if tradeseAndOrdersCodec, res = goavro.NewCodec(tradesAndOrdersSchema); res != nil {
		return res
	} else if booksCodec, res = goavro.NewCodec(booksSchema); res != nil {
		return res
	} else if accountCodec, res = goavro.NewCodec(accountSchema); res != nil {
		return res
	}
	return nil
}