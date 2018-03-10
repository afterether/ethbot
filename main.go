package main

import (
    "fmt"
    "os"
	"io"
	"os/signal"
    "runtime"
    "sort"
    "time"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/node"
    "github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/metrics"
    "github.com/ethereum/go-ethereum/console"
    "github.com/ethereum/go-ethereum/log"

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
        utils.NoUSBFlag,
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
        utils.LightServFlag,
        utils.LightPeersFlag,
        utils.LightKDFFlag,
        utils.CacheFlag,
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
        utils.DevModeFlag,
        utils.TestnetFlag,
        utils.RinkebyFlag,
        utils.VMEnableDebugFlag,
        utils.NetworkIdFlag,
        utils.RPCCORSDomainFlag,
        utils.EthStatsURLFlag,
        utils.MetricsEnabledFlag,
        utils.FakePoWFlag,
        utils.NoCompactionFlag,
        utils.GpoBlocksFlag,
        utils.GpoPercentileFlag,
        utils.ExtraDataFlag,
        utils.NoExportFlag,
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
        utils.WhisperEnabledFlag,
        utils.WhisperMaxMessageSizeFlag,
        utils.WhisperMinPOWFlag,
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
    // Initialize the CLI app and start Geth
    app.Action = geth
    app.HideVersion = true // we have a command to print the version
    app.Copyright = "Copyright 2013-2017 The go-ethereum Authors"
    app.Commands = []cli.Command{
        // See chaincmd.go:
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
	log.Info("EthBot node starting...")

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
		log.Info("Got interrupt, shutting down...")
        go stack.Stop()
        for i := 10; i > 0; i-- {
            <-sigc
            if i > 1 {
                log.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
            }
        }
        debug.LoudPanic("boom")
    }()

	ethbot_instance.check_export_on_startup(ctx)
}

func main() {
    log.PrintOrigins(true)

    if err := app.Run(os.Args); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
