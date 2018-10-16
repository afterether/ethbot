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
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/robertkrimen/otto"

)
func NewEthBot(ctx *node.ServiceContext) (node.Service, error) {
    var ethereum *eth.Ethereum
    if err := ctx.Service(&ethereum); err != nil {
        return nil, err
    }
    ethbot_instance=&EthBot_t {
        ethereum:				ethereum,
		head_ch:				nil,
        head_sub:				nil,
		blocks:					make(chan *types.Block),
		verification:			Verification_t{},
		export:					Export_t{},
    }
	ethbot_instance.token_export.blocks=make(chan *types.Block)
	return ethbot_instance, nil
}
func (bot *EthBot_t) Start(server *p2p.Server) error {
    log.Info("EthBot: Starting EthBot.")
    bot.server = server
	init_postgres()
    return nil
}
func (bot *EthBot_t) Stop() error {
    log.Info("EthBot: Stopping EthBot.")
	if (bot.head_sub!=nil) {
	    bot.head_sub.Unsubscribe()
	}
    return nil
}
func (bot *EthBot_t) APIs() []rpc.API {
    return []rpc.API{{
			Namespace: "ethbot",
			Version:	"1.0",
			Service:	&EthBotAPI {
				bot: bot,
			},
		},{
			Namespace: "sim",
			Version:	"1.0",
			Public: true,
			Service:	&SimAPI {
				bot: bot,
			},
		},
	}
}
func (bot *EthBot_t) Protocols() ([]p2p.Protocol) {
    return []p2p.Protocol{}
}
func (bot *EthBot_t) empty_object() *otto.Object {
	jsre:=console_obj.JSRE()
	vm:=jsre.VM()
	object, err := vm.Object("({})")
	if (err!=nil) {
		utils.Fatalf("Failed to create object in Javascript VM for blockchain export status: %v",err)
	}
	return object
}
