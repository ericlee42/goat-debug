package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/goatnetwork/goat/app"
)

func main() {
	var (
		NodeURL string
		Start   int64
	)

	flag.StringVar(&NodeURL, "node", "tcp://localhost:26657", "The node to connect to")
	flag.Int64Var(&Start, "start", 11044000, "the start height")
	flag.Parse()

	var (
		autoCliOpts        autocli.AppOptions
		moduleBasicManager module.BasicManager
		clientCtx          client.Context
	)

	basectx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := depinject.Inject(
		depinject.Configs(app.AppConfig(),
			depinject.Supply(log.NewNopLogger()),
			depinject.Provide(ProvideClientContext),
		),
		&autoCliOpts, &moduleBasicManager, &clientCtx,
	); err != nil {
		panic(err)
	}

	httpClient, err := client.NewClientFromNode(NodeURL)
	if err != nil {
		panic(err)
	}
	clientCtx = clientCtx.WithNodeURI(NodeURL).WithClient(httpClient)

	node, err := clientCtx.GetNode()
	if err != nil {
		panic(err)
	}

	statusRes, err := node.Status(basectx)
	if err != nil {
		panic(err)
	}

	fmt.Println("latest block:", statusRes.SyncInfo.LatestBlockHeight)

	var failedList []int64
	for height := Start; height <= statusRes.SyncInfo.LatestBlockHeight; {
		if height == 0 {
			continue
		}

		fmt.Println("Fetching block:", height)
		result, err := func() (res *coretypes.ResultBlockResults, err error) {
			ctx, cancel := context.WithTimeout(basectx, time.Second*5)
			defer cancel()
			return node.BlockResults(ctx, &height)
		}()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Println("context canceled, exiting...")
				return
			}
			fmt.Println("failed to fetch block results for height", height, "error:", err)
			fmt.Println("retrying...")
			continue
		}

		for i, txResult := range result.TxsResults {
			if i != 0 {
				break
			}

			if txResult.Code != 0 {
				failedList = append(failedList, height)
				fmt.Println("got bad block", height)
			}
		}
		height++
	}
	LogBadBlocks(failedList)
}

func ProvideClientContext(
	appCodec codec.Codec,
	interfaceRegistry codectypes.InterfaceRegistry,
	txConfigOpts tx.ConfigOptions,
	legacyAmino *codec.LegacyAmino,
) client.Context {
	clientCtx := client.Context{}.
		WithCodec(appCodec).
		WithInterfaceRegistry(interfaceRegistry).
		WithLegacyAmino(legacyAmino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{})

	txConfig, err := tx.NewTxConfigWithOptions(clientCtx.Codec, txConfigOpts)
	if err != nil {
		panic(err)
	}
	clientCtx = clientCtx.WithTxConfig(txConfig)

	return clientCtx
}

func LogBadBlocks(blocks []int64) {
	if len(blocks) == 0 {
		fmt.Println("no failed blocks to save")
		return
	}

	j := json.NewEncoder(os.Stdout)
	j.SetIndent("", "  ")
	if err := j.Encode(blocks); err != nil {
		panic(err)
	}
	fmt.Println(len(blocks), "bad blocks found")
}
