package api

import (
	"encoding/hex"
	"net/url"
	"pandora-pay/blockchain"
	"pandora-pay/config"
	"strconv"
	"sync/atomic"
	"unsafe"
)

type API struct {
	GetMap map[string]func(values url.Values) interface{}

	chain      *blockchain.Blockchain
	localChain unsafe.Pointer
}

func (api *API) getBlockchain(values url.Values) interface{} {
	pointer := atomic.LoadPointer(&api.localChain)
	return (*APIBlockchain)(pointer)
}

func (api *API) getInfo(values url.Values) interface{} {
	return &struct {
		Name        string
		Version     string
		Network     uint64
		CPU_THREADS int
	}{
		Name:        config.NAME,
		Version:     config.VERSION,
		Network:     config.NETWORK_SELECTED,
		CPU_THREADS: config.CPU_THREADS,
	}
}

func (api *API) getPing(values url.Values) interface{} {
	return &struct {
		Ping string
	}{Ping: "Pong"}
}

func (api *API) getBlockComplete(values url.Values) interface{} {
	heightStr := values.Get("height")
	if heightStr != "" {
		height, err := strconv.Atoi(heightStr)
		if err != nil {
			panic("parameter 'height' is not a number")
		}
		return api.loadBlockCompleteFromHeight(uint64(height))
	}
	hashStr := values.Get("hash")
	if hashStr != "" {
		hash, err := hex.DecodeString(values.Get("hash"))
		if err != nil {
			panic("parameter 'hash' was is not a valid hex number")
		}
		return api.loadBlockCompleteFromHash(hash)
	}
	panic("parameter 'hash' or 'height' are missing")
}

func (api *API) getBlock(values url.Values) interface{} {
	heightStr := values.Get("height")
	if heightStr != "" {
		height, err := strconv.Atoi(heightStr)
		if err != nil {
			panic("parameter 'height' is not a number")
		}
		return api.loadBlockWithTXsFromHeight(uint64(height))
	}
	hashStr := values.Get("hash")
	if hashStr != "" {
		hash, err := hex.DecodeString(values.Get("hash"))
		if err != nil {
			panic("parameter 'hash' was is not a valid hex number")
		}
		return api.loadBlockWithTXsFromHash(hash)
	}
	panic("parameter 'hash' or 'height' are missing")
}

func (api *API) getTx(values url.Values) interface{} {
	hashStr := values.Get("hash")
	if hashStr != "" {
		hash, err := hex.DecodeString(values.Get("hash"))
		if err != nil {
			panic("parameter 'hash' was is not a valid hex number")
		}
		return api.loadTxFromHash(hash)
	}
	panic("parameter 'hash' or ")
}

//make sure it is safe to read
func (api *API) readLocalBlockchain(newChain *blockchain.Blockchain) {
	newLocalChain := APIBlockchain{
		Height:          newChain.Height,
		Hash:            hex.EncodeToString(newChain.Hash),
		PrevHash:        hex.EncodeToString(newChain.PrevHash),
		KernelHash:      hex.EncodeToString(newChain.KernelHash),
		PrevKernelHash:  hex.EncodeToString(newChain.PrevKernelHash),
		Timestamp:       newChain.Timestamp,
		Transactions:    newChain.Transactions,
		Target:          newChain.Target.String(),
		TotalDifficulty: newChain.BigTotalDifficulty.String(),
	}
	atomic.StorePointer(&api.localChain, unsafe.Pointer(&newLocalChain))
}

func CreateAPI(chain *blockchain.Blockchain) *API {

	api := API{
		chain: chain,
	}

	api.GetMap = map[string]func(values url.Values) interface{}{
		"/":               api.getInfo,
		"/chain":          api.getBlockchain,
		"/ping":           api.getPing,
		"/block-complete": api.getBlockComplete,
		"/block":          api.getBlock,
		"/tx":             api.getTx,
	}

	go func() {
		for {
			newChain := <-api.chain.UpdateNewChainChannel
			//it is safe to read
			api.readLocalBlockchain(newChain)
		}
	}()

	chain.RLock()
	api.readLocalBlockchain(chain)
	chain.RUnlock()

	return &api
}
