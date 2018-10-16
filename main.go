/*
	Copyright 2018 The AfterEther Team
	This file is part of the EthBot, Ethereum Blockchain -> SQL converter.
		
	EthBot is free software: you can redistribute it and/or modify
	it under the terms of the GNU Lesser General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
	
	EthBot is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
	GNU Lesser General Public License for more details.
	
	You should have received a copy of the GNU Lesser General Public License
	along with EthBot. If not, see <http://www.gnu.org/licenses/>.
*/
package main

import (
    "fmt"
    "os"
	"io"
	"os/signal"
    "runtime"
    "sort"
    "time"
	"strings"
	"encoding/hex"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/node"
    "github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/metrics"
    "github.com/ethereum/go-ethereum/console"
    "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/afterether/ethbot/internal/debug"

	"gopkg.in/urfave/cli.v1"
)
const (
    clientIdentifier = "geth" // Client identifier to advertise over the network
)

var (
    // Git SHA1 commit hash of the release (set via linker flags)
    gitCommit = ""
    // Ethereum address of the Geth release oracle.
    relOracle = common.HexToAddress("0xfa7b9770ca4cb04296cac84f37736d4041251cdf")
    // The app that holds all commands and flags.
    app = utils.NewApp(gitCommit, "the go-ethereum command line interface")
    // flags that configure the node
    nodeFlags = []cli.Flag{
		utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,
		utils.BootnodesFlag,
		utils.BootnodesV4Flag,
		utils.BootnodesV5Flag,
		utils.DataDirFlag,
		utils.KeyStoreDirFlag,
		utils.EthashCacheDirFlag,
		utils.EthashCachesInMemoryFlag,
		utils.EthashCachesOnDiskFlag,
		utils.EthashDatasetDirFlag,
		utils.EthashDatasetsInMemoryFlag,
		utils.EthashDatasetsOnDiskFlag,
		utils.TxPoolNoLocalsFlag,
		utils.TxPoolJournalFlag,
		utils.TxPoolRejournalFlag,
		utils.TxPoolPriceLimitFlag,
		utils.TxPoolPriceBumpFlag,
		utils.TxPoolAccountSlotsFlag,
		utils.TxPoolGlobalSlotsFlag,
		utils.TxPoolAccountQueueFlag,
		utils.TxPoolGlobalQueueFlag,
		utils.TxPoolLifetimeFlag,
		utils.FastSyncFlag,
		utils.LightModeFlag,
		utils.SyncModeFlag,
		utils.GCModeFlag,
		utils.CacheFlag,
		utils.CacheDatabaseFlag,
		utils.CacheGCFlag,
		utils.TrieCacheGenFlag,
		utils.ListenPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.EtherbaseFlag,
		utils.GasPriceFlag,
		utils.MinerThreadsFlag,
		utils.MiningEnabledFlag,
		utils.TargetGasLimitFlag,
		utils.NATFlag,
		utils.NoDiscoverFlag,
		utils.DiscoveryV5Flag,
		utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.VMEnableDebugFlag,
		utils.NetworkIdFlag,
		utils.RPCCORSDomainFlag,
		utils.RPCVirtualHostsFlag,
		utils.EthStatsURLFlag,
		utils.MetricsEnabledFlag,
		utils.FakePoWFlag,
		utils.NoCompactionFlag,
		utils.GpoBlocksFlag,
		utils.GpoPercentileFlag,
		utils.ExtraDataFlag,
		configFileFlag,
		utils.NoExportFlag,
		utils.PTXOutFlag,
    }
    rpcFlags = []cli.Flag{
        utils.RPCEnabledFlag,
        utils.RPCListenAddrFlag,
        utils.RPCPortFlag,
        utils.RPCApiFlag,
        utils.WSEnabledFlag,
        utils.WSListenAddrFlag,
        utils.WSPortFlag,
        utils.WSApiFlag,
        utils.WSAllowedOriginsFlag,
        utils.IPCDisabledFlag,
        utils.IPCPathFlag,
    }
    whisperFlags = []cli.Flag{
    }
)
func Fatalf(format string, args ...interface{}) {
    w := io.MultiWriter(os.Stdout, os.Stderr)
    if runtime.GOOS == "windows" {
        // The SameFile check below doesn't work on Windows.
        // stdout is unlikely to get redirected though, so just print there.
        w = os.Stdout
    } else {
        outf, _ := os.Stdout.Stat()
        errf, _ := os.Stderr.Stat()
        if outf != nil && errf != nil && os.SameFile(outf, errf) {
            w = os.Stderr
        }
    }
    fmt.Fprintf(w, "Fatal: "+format+"\n", args...)
    os.Exit(1)
}
func init() {
	var err error

	_, err = os.Stat(alarms_dir)
	if os.IsNotExist(err) {
		err=os.Mkdir(alarms_dir,0755)
		if err!=nil {
			utils.Fatalf(fmt.Sprintf("Can't create directory %v: %v",alarms_dir,err))
		}
	}
	erc20_token_abi, err = abi.JSON(strings.NewReader(erc20_token_abi_str))
	if err!=nil {
		utils.Fatalf("Invalid events ABI for ERC20 token standard: %v",err)
	}
	erc20_methods_abi, err = abi.JSON(strings.NewReader(erc20_methods_abi_json_str))
	if err!=nil {
		utils.Fatalf("Invalid methods ABI for ERC20 token standard: %v",err)
	}
	erc20_transfer_event_signature,_ = hex.DecodeString("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	erc20_approval_event_signature,_ = hex.DecodeString("8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")
	erc20_transfer_method_signature,_= hex.DecodeString("a9059cbb")
	erc20_approve_method_signature,_ = hex.DecodeString("095ea7b3")
	erc20_transfer_from_method_signature,_ = hex.DecodeString("23b872dd")

    // Initialize the CLI app and start Geth
    app.Action = geth
    app.HideVersion = true // we have a command to print the version
    app.Copyright = "Copyright 2013-2017 The go-ethereum Authors"
    app.Commands = []cli.Command{
        initCommand,
        importCommand,
        consoleCommand,
        attachCommand,
        javascriptCommand,
    }
    sort.Sort(cli.CommandsByName(app.Commands))

    app.Flags = append(app.Flags, nodeFlags...)
    app.Flags = append(app.Flags, rpcFlags...)
    app.Flags = append(app.Flags, consoleFlags...)
    app.Flags = append(app.Flags, debug.Flags...)
    app.Flags = append(app.Flags, whisperFlags...)
    app.Before = func(ctx *cli.Context) error {
        runtime.GOMAXPROCS(runtime.NumCPU())
        if err := debug.Setup(ctx); err != nil {
            return err
        }
	// Start system runtime metrics collection
        go metrics.CollectProcessMetrics(3 * time.Second)

        utils.SetupNetwork(ctx)
        return nil
    }

    app.After = func(ctx *cli.Context) error {
        debug.Exit()
        console.Stdin.Close() // Resets terminal mode.
        return nil
    }
}
// geth is the main entry point into the system if no special subcommand is ran.
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
func geth(ctx *cli.Context) error {
    node := makeFullNode(ctx)

    startNode(ctx, node)
    node.Wait()
    return nil
}
// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func startNode(ctx *cli.Context, stack *node.Node) {
	log.Info("EthBot: node starting...")

    if err := stack.Register(
			func(ctx *node.ServiceContext) (node.Service, error) {
				return NewEthBot(ctx)
			})
		err != nil {
        utils.Fatalf("Failed to register the etherquery service: %v", err)
    }

    if err := stack.Start(); err != nil {
        Fatalf("Error starting protocol stack: %v", err)
    }
    go func() {
        sigc := make(chan os.Signal, 1)
        signal.Notify(sigc, os.Interrupt)
        defer signal.Stop(sigc)
        <-sigc
		log.Info("EthBot: got interrupt, shutting down...")
        go stack.Stop()
        for i := 10; i > 0; i-- {
            <-sigc
            if i > 1 {
                log.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
            }
        }
        debug.LoudPanic("boom")
    }()

	method,_:=erc20_token_abi.Methods["approve"]
	ethbot_instance.check_export_on_startup(ctx)
	ethbot_instance.verify_genesis_block_hashes()
}

func main() {
    log.PrintOrigins(true)

    if err := app.Run(os.Args); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
