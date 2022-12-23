package main

import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"phqb.com/gethplayground/erc20"
)

// const RPC_ENDPOINT = "https://ethereum.kyberengineering.io"
const RPC_ENDPOINT = "https://eth-mainnet.alchemyapi.io/v2/hqcYQ0PDgAtqFtVAzg64Mc40YjRhX9sd"

func fetchERC20TransferEvents() {
	var (
		client, _           = ethclient.Dial(RPC_ENDPOINT)
		erc20ABI, _         = erc20.Erc20MetaData.GetAbi()
		erc20ABIInstance, _ = erc20.NewErc20(common.Address{}, client)
		blockHash           = common.HexToHash("0x7008451b87e4f126e3b5428d4ea2c6f23167ddbb8a1c1fa1d4e1d9ca70faaca8")
	)
	erc20TransferLogs, err := client.FilterLogs(context.TODO(), ethereum.FilterQuery{
		Topics:    [][]common.Hash{{erc20ABI.Events["Transfer"].ID}},
		BlockHash: &blockHash,
	})
	if err != nil {
		panic(err)
	}
	for _, log := range erc20TransferLogs {
		transferEvent, err := erc20ABIInstance.ParseTransfer(log)
		if err != nil {
			fmt.Printf("could not parse transfer event, error: %s\nLog %+v\n", err, log)
			continue
		}
		fmt.Printf("ERC20 Transfer from=%s to=%s token=%s amount=%s\n", transferEvent.From, transferEvent.To, transferEvent.Raw.Address, transferEvent.Value)
		// fmt.Printf("Log %+v\n", log)
	}
	fmt.Printf("erc20TransferLogs len %d", len(erc20TransferLogs))
}

// type address = [20]byte
type address string

type txCallOpcodeJSON struct {
	// CallOps []struct {
	// 	From address `json:"from"`
	// 	Addr string  `json:"addr"`
	// 	Val  string  `json:"val"`
	// } `json:"callOps"`
	CtxType   string  `json:"type"`
	CtxFrom   address `json:"from"`
	CtxTo     address `json:"to"`
	CtxValue  string  `json:"value"`
	CtxGas    string  `json:"gas"`
	CtxUsed   string  `json:"gasUsed"`
	CtxInput  string  `json:"input"`
	CtxOutput string  `json:"output"`
}

type traceBlockByHashResultJSON struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  []struct {
		Result txCallOpcodeJSON `json:"result"`
	} `json:"result"`

	// []txCallOpcodeJSON `json:"result"`
	// Error *string `json:"error"`
}

type rpcCallJSON struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type tracerParamJSON struct {
	Tracer string `json:"tracer"`
}

func bigIntToAddress(s string) common.Address {
	bn, _ := new(big.Int).SetString(s, 10)
	return common.BytesToAddress(bn.Bytes())
}

func fetchNativeTransfers() {
	var (
		blockHash = common.HexToHash("0x7008451b87e4f126e3b5428d4ea2c6f23167ddbb8a1c1fa1d4e1d9ca70faaca8") // blockNumber 16139416
		// blockHash  = common.HexToHash("0x8643ce4273b568a15bce44a528d2532c834d5f5c0a54bb2beaa888a484cc10bf") // blockNumber 16139417
		httpClient = &http.Client{}
		tracer     = "callTracer"
		// tracer     = `{
		// 	retVal: {
		// 		callOps: []
		// 	},
		// 	step: function (log, db) {
		// 		if (log.op.toNumber() == 0xF1)
		// 			this.retVal.callOps.push({
		// 				from: log.contract.getAddress(),
		// 				addr: log.stack.peek(1),
		// 				val: log.stack.peek(2)
		// 			});
		// 	},
		// 	fault: function(log, db) {},
		// 	result: function(ctx, db) {
		// 		this.retVal.from = ctx.from;
		// 		this.retVal.to = ctx.to;
		// 		this.retVal.value = ctx.value;
		// 		return this.retVal;
		// 	}
		// }`
	)
	payload, _ := json.Marshal(rpcCallJSON{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "debug_traceBlockByHash",
		Params: []interface{}{
			blockHash.String(),
			tracerParamJSON{
				Tracer: tracer,
			},
		},
	})
	request, err := http.NewRequest("POST", RPC_ENDPOINT, strings.NewReader(string(payload)))
	if err != nil {
		fmt.Printf("Error in init request, %s\n", err)
		panic(err)
	}
	request.Header.Add("Content-Type", "application/json")
	response, err := httpClient.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Error calling request, %s\n", err)
		panic(err)
	}

	// fmt.Printf("response.Body %+v", response.Body)
	var result traceBlockByHashResultJSON
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Printf("Error in parsing result, %s\n", err)
		// fmt.Printf("Response.body %+v\n", body)
		panic(err)
	}
	// fmt.Printf("result %+v", result)
	for _, item := range result.Result {
		t := item.Result
		//fmt.Printf("native transfer from=%s to=%s amount=%s\n", common.BytesToAddress(t.CtxFrom[:]), common.BytesToAddress(t.CtxTo[:]), t.CtxValue)
		hexAmount := strings.Replace(t.CtxValue, "0x", "", -1)
		amountDecimal, err := strconv.ParseInt(hexAmount, 16, 64)
		if err == nil {
			fmt.Printf("native transfer from=%s to=%s amount=%d\n", t.CtxFrom, t.CtxTo, amountDecimal)
		} else {
			fmt.Printf("Error %s", err)
		}
	}
}

type BlockNumber int64

const BLOCK_SUBGRAPH_URL = "https://api.thegraph.com/subgraphs/name/kybernetwork/ethereum-blocks"

func blockByTimestamp(timestamp int64, flag string) BlockNumber {
	var function, ordering string
	if flag == "before" {
		function = "timestamp_lte"
		ordering = "desc"
	} else {
		function = "timestamp_gte"
		ordering = "asc"
	}
	query := fmt.Sprintf(`{{
		blocks(
			where: {{ %s: %d }}
		orderBy: timestamp
		orderDirection: %s
		first: 1
		) {{
			number
		}}
	}}`, function, timestamp, ordering)

	fmt.Printf("query %s", query)
	type GraphqlBody struct {
		Query string `json:"query"`
	}
	payload, _ := json.Marshal(GraphqlBody{
		Query: query,
	})
	httpClient := &http.Client{}

	// New request
	request, err := http.NewRequest("POST", BLOCK_SUBGRAPH_URL, strings.NewReader(string(payload)))
	if err != nil {
		fmt.Printf("Error in init request, %s\n", err)
		panic(err)
	}
	request.Header.Add("Content-Type", "application/json")

	// Call request
	response, err := httpClient.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	// Parse response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Error calling request, %s\n", err)
		panic(err)
	}
	var result struct {
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Printf("Error in parsing result, %s\n", err)
		// fmt.Printf("Response.body %+v\n", body)
		panic(err)
	}
	return 0
}

func main() {
	// blockByTimestamp(1670495543, "closest")
	fetchERC20TransferEvents()
	fetchNativeTransfers()
}
