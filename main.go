package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/goatnetwork/goat/app"
	"github.com/goatnetwork/goat/cmd/goatd/cmd"
)

func main() {
	var (
		NodeURL string
	)

	flag.StringVar(&NodeURL, "node", "tcp://localhost:26657", "The node to connect to")

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
			depinject.Provide(cmd.ProvideClientContext),
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

	res, err := node.Status(basectx)
	if err != nil {
		panic(err)
	}

	var list []int64
	for height := range res.SyncInfo.LatestBlockHeight {
		if height == 0 {
			continue
		}

		fmt.Println("Fetching block:", height)
		result, err := node.BlockResults(basectx, &height)
		if err != nil {
			panic(err)
		}
		for i, txResult := range result.TxsResults {
			if i != 0 {
				break
			}

			if txResult.Code != 0 {
				list = append(list, height)
			}
		}
	}
	fmt.Println("Failed blocks:", list)
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
