package store

import (
	"fmt"

	"github.com/BiJie/BinanceChain/common/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/wire"
	"github.com/pkg/errors"
)

type Mapper interface {
	NewToken(ctx sdk.Context, token types.Token) error
	Exists(ctx sdk.Context, symbol string) bool
	GetTokenList(ctx sdk.Context) []types.Token
	GetToken(ctx sdk.Context, symbol string) (types.Token, error)
	UpdateToken(ctx sdk.Context, token types.Token) error
}

type mapper struct {
	key   sdk.StoreKey
	cdc   *wire.Codec
}

func NewMapper(cdc *wire.Codec, key sdk.StoreKey) mapper {
	return mapper{
		key:   key,
		cdc:   cdc,
	}
}

func (m mapper) GetToken(ctx sdk.Context, symbol string) (types.Token, error) {
	store := ctx.KVStore(m.key)
	key := []byte(symbol)

	bz := store.Get(key)
	if bz != nil {
		return m.decodeToken(bz), nil
	}

	return types.Token{}, errors.New(fmt.Sprintf("token(%v) not found", symbol))
}

func (m mapper) GetTokenList(ctx sdk.Context) []types.Token {
	var res []types.Token

	store := ctx.KVStore(m.key)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		token := m.decodeToken(iter.Value())
		res = append(res, token)
	}
	return res
}

func (m mapper) Exists(ctx sdk.Context, symbol string) bool {
	store := ctx.KVStore(m.key)
	key := []byte(symbol)
	return store.Has(key)
}

func (m mapper) NewToken(ctx sdk.Context, token types.Token) error {
	symbol := token.Symbol
	if len(symbol) == 0 {
		return errors.New("symbol cannot be empty")
	}

	key := []byte(symbol)
	store := ctx.KVStore(m.key)
	value := m.encodeToken(token)
	store.Set(key, value)
	return nil
}

func (m mapper) UpdateToken(ctx sdk.Context, token types.Token) error {
	symbol := token.Symbol
	if len(symbol) == 0 {
		return errors.New("symbol cannot be empty")
	}

	key := []byte(symbol)
	store := ctx.KVStore(m.key)
	bz := store.Get(key)
	if bz == nil {
		return errors.New("token does not exist")
	}

	toBeUpdated := m.decodeToken(bz)
	// currently, only name and supply can be updated.
	if len(token.Name) != 0 {
		toBeUpdated.Name = token.Name
	}

	if len(token.Supply.Value) !=0 {
		toBeUpdated.Supply = token.Supply
	}

	store.Set(key, m.encodeToken(toBeUpdated))
	return nil
}

func (m mapper) encodeToken(token types.Token) []byte {
	bz, err := m.cdc.MarshalBinaryBare(token)
	if err != nil {
		panic(err)
	}

	return bz
}

func (m mapper) decodeToken(bz []byte) (token types.Token) {
	err := m.cdc.UnmarshalBinaryBare(bz, &token)
	if err != nil {
		panic(err)
	}

	return
}