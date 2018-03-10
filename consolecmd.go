// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"os"
	"os/signal"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"gopkg.in/urfave/cli.v1"
	"github.com/ethereum/go-ethereum/log"
)

var (
	console_obj *console.Console		// we made 'console' global var so verifier can access it
	consoleFlags = []cli.Flag{utils.JSpathFlag, utils.ExecFlag, utils.PreloadJSFlag}
	remote_EthBot	*rpc.Client
	consoleCommand = cli.Command{
		Action:   utils.MigrateFlags(localConsole),
		Name:     "console",
		Usage:    "Start an interactive JavaScript environment",
		Flags:    append(append(append(nodeFlags, rpcFlags...), consoleFlags...), whisperFlags...),
		Category: "CONSOLE COMMANDS",
		Description: `
The Geth console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://github.com/ethereum/go-ethereum/wiki/Javascipt-Console.`,
	}

	attachCommand = cli.Command{
		Action:    utils.MigrateFlags(remoteConsole),
		Name:      "attach",
		Usage:     "Start an interactive JavaScript environment (connect to node)",
		ArgsUsage: "[endpoint]",
		Flags:     append(consoleFlags, utils.DataDirFlag),
		Category:  "CONSOLE COMMANDS",
		Description: `
The Geth console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://github.com/ethereum/go-ethereum/wiki/Javascipt-Console.
This command allows to open a console on a running geth node.`,
	}

	javascriptCommand = cli.Command{
		Action:    utils.MigrateFlags(ephemeralConsole),
		Name:      "js",
		Usage:     "Execute the specified JavaScript files",
		ArgsUsage: "<jsfile> [jsfile...]",
		Flags:     append(nodeFlags, consoleFlags...),
		Category:  "CONSOLE COMMANDS",
		Description: `
The JavaScript VM exposes a node admin interface as well as the Ðapp
JavaScript API. See https://github.com/ethereum/go-ethereum/wiki/Javascipt-Console`,
	}
)

// localConsole starts a new geth node, attaching a JavaScript console to it at the
// sam time.
func localConsole(ctx *cli.Context) error {
	// Create and start the node based on the CLI flags
	node := makeFullNode(ctx)
	startNode(ctx, node)
	defer node.Stop()

	// Attach to the newly started node and start the JavaScript console
	client, err := node.Attach()
	if err != nil {
		utils.Fatalf("Failed to attach to the inproc geth: %v", err)
	}
	config := console.Config{
		DataDir: utils.MakeDataDir(ctx),
		DocRoot: ctx.GlobalString(utils.JSpathFlag.Name),
		Client:  client,
		Preload: utils.MakeConsolePreloads(ctx),
	}

	console_obj, err = console.New(config)
	if err != nil {
		utils.Fatalf("Failed to start the JavaScript console: %v", err)
	}
	defer console_obj.Stop(false)

	// If only a short execution was requested, evaluate and return
	if script := ctx.GlobalString(utils.ExecFlag.Name); script != "" {
		console_obj.Evaluate(script)
		return nil
	}

	jsre:=console_obj.JSRE()
	vm:=jsre.VM()
	object, err := vm.Object(`ethbot={}`)
	if (err!=nil) {
		utils.Fatalf("Failed to initialize ethbot JS API")
	}
	object.Set("verifySQLdata",js_local_verify_sql_data);
	object.Set("verificationStatus",js_local_verification_status);
	object.Set("stopVerification",js_local_stop_verification);
	object.Set("exportBlockRange",js_local_export_block_range);
	object.Set("verifyAccount",js_local_verify_account);
	object.Set("verifyAllAccounts",js_local_verify_all_accounts);
	object.Set("blockchainExportStart",js_local_blockchain_export_start);
	object.Set("blockchainExportStop",js_local_blockchain_export_stop);
	object.Set("blockchainExportStatus",js_local_blockchain_export_status);

	log.Info("Starting local console")
	// Otherwise print the welcome screen and enter interactive mode
	console_obj.Welcome()
	console_obj.Interactive()

	return nil
}

// remoteConsole will connect to a remote geth instance, attaching a JavaScript
// console to it.
func remoteConsole(ctx *cli.Context) error {
	// Attach to a remotely running geth instance and start the JavaScript console
	client, err := dialRPC(ctx.Args().First())
	if err != nil {
		utils.Fatalf("Unable to attach to remote geth: %v", err)
	}

	remote_EthBot=client
	modules,err:=remote_EthBot.SupportedModules()
	if (err!=nil) {
		log.Error("Error getting RPC modules, check RPC is enabledi or use IPC ","error",err)
		os.Exit(2)
	}
	_=modules
	config := console.Config{
		DataDir: utils.MakeDataDir(ctx),
		DocRoot: ctx.GlobalString(utils.JSpathFlag.Name),
		Client:  client,
		Preload: utils.MakeConsolePreloads(ctx),
	}
	console_obj, err = console.New(config)
	if err != nil {
		utils.Fatalf("Failed to start the JavaScript console: %v", err)
	}
	defer console_obj.Stop(false)

	if script := ctx.GlobalString(utils.ExecFlag.Name); script != "" {
		console_obj.Evaluate(script)
		return nil
	}

	jsre:=console_obj.JSRE()
	vm:=jsre.VM()
	object, err := vm.Object(`ethbot={}`)
	if (err!=nil) {
		utils.Fatalf("Failed to initialize ethbot JS API")
	}
	object.Set("verifySQLdata",js_remote_verify_sql_data);
	object.Set("verificationStatus",js_remote_verification_status);
	object.Set("stopVerification",js_remote_stop_verification);
	object.Set("exportBlockRange",js_remote_export_block_range);
	object.Set("verifyAccount",js_remote_verify_account);
	object.Set("verifyAllAccounts",js_remote_verify_all_accounts);
	object.Set("blockchainExportStart",js_remote_blockchain_export_start);
	object.Set("blockchainExportStop",js_remote_blockchain_export_stop);
	object.Set("blockchainExportStatus",js_remote_blockchain_export_status);
	// Otherwise print the welcome screen and enter interactive mode
	console_obj.Welcome()
	console_obj.Interactive()

	return nil
}

// dialRPC returns a RPC client which connects to the given endpoint.
// The check for empty endpoint implements the defaulting logic
// for "geth attach" and "geth monitor" with no argument.
func dialRPC(endpoint string) (*rpc.Client, error) {
	if endpoint == "" {
		endpoint = node.DefaultIPCEndpoint(clientIdentifier)
	} else if strings.HasPrefix(endpoint, "rpc:") || strings.HasPrefix(endpoint, "ipc:") {
		// Backwards compatibility with geth < 1.5 which required
		// these prefixes.
		endpoint = endpoint[4:]
	}
	return rpc.Dial(endpoint)
}

// ephemeralConsole starts a new geth node, attaches an ephemeral JavaScript
// console to it, executes each of the files specified as arguments and tears
// everything down.
func ephemeralConsole(ctx *cli.Context) error {
	// Create and start the node based on the CLI flags
	node := makeFullNode(ctx)
	startNode(ctx, node)
	defer node.Stop()

	// Attach to the newly started node and start the JavaScript console
	client, err := node.Attach()
	if err != nil {
		utils.Fatalf("Failed to attach to the inproc geth: %v", err)
	}
	config := console.Config{
		DataDir: utils.MakeDataDir(ctx),
		DocRoot: ctx.GlobalString(utils.JSpathFlag.Name),
		Client:  client,
		Preload: utils.MakeConsolePreloads(ctx),
	}

	console, err := console.New(config)
	if err != nil {
		utils.Fatalf("Failed to start the JavaScript console: %v", err)
	}
	defer console.Stop(false)

	// Evaluate each of the specified JavaScript files
	for _, file := range ctx.Args() {
		if err = console.Execute(file); err != nil {
			utils.Fatalf("Failed to execute %s: %v", file, err)
		}
	}
	// Wait for pending callbacks, but stop for Ctrl-C.
	abort := make(chan os.Signal, 1)
	signal.Notify(abort, os.Interrupt)

	go func() {
		<-abort
		os.Exit(0)
	}()
	console.Stop(true)

	return nil
}
