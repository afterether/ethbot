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
	"strconv"
	"os"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/state"
    "github.com/ethereum/go-ethereum/cmd/utils"
)
func dump_vtransfers(items []*vm.Ethbot_EVM_VT_t) {
	for i,item := range items {
		log.Info(fmt.Sprintf("%v : value=%v, from %v -> to %v ; refund=%v",i,item.Value.String(),hex.EncodeToString(item.From.Bytes()),hex.EncodeToString(item.To.Bytes()),item.Gas_refund.String()))
		log.Info(fmt.Sprintf("\tDepth %v Kind %v. FromBal: %v ToBal: %v",item.Depth,item.Kind,item.From_balance.String(),item.To_balance.String()))
	}
}
func vt_dump(VT *vm.Ethbot_EVM_VT_t) {
	var err_str string
	if VT.Err!=nil {
		err_str=fmt.Sprintf(", err=%v",VT.Err.Error())
	}
	log.Info(fmt.Sprintf("from %v -> to %v, value=%v, refund=%v%v",hex.EncodeToString(VT.From.Bytes()),hex.EncodeToString(VT.To.Bytes()),VT.Value.String(),VT.Gas_refund.String(),err_str))
	log.Info(fmt.Sprintf("\tDepth %v Kind %v. FromBal: %v ToBal: %v",VT.Depth,VT.Kind,VT.From_balance.String(),VT.To_balance.String()))
}
func dump_State_balances(state *state.StateDB,VT *vm.Ethbot_EVM_VT_t) {
	from_balance:=state.GetBalanceIfExists(VT.From)
	to_balance:=state.GetBalanceIfExists(VT.To)
	log.Info(fmt.Sprintf("BAL (from) %v=%v , (to) %v=%v",hex.EncodeToString(VT.From.Bytes()),from_balance.String(),hex.EncodeToString(VT.To.Bytes()),to_balance))
}
func debug1() {
	log.Info("Nothing to do. Insert your golang code here")
}
func debug2() {
	log.Info("Nothing to do. Insert your golang code here")
}
func dump_VTs_to_file(block_num Block_num_t,tx_index int, transfers *[]*vm.Ethbot_EVM_VT_t) {

	block_num_str:=strconv.Itoa(int(block_num))
	filename:=dump_VTs_dir+"/"+block_num_str+"-"+fmt.Sprintf("%04d",tx_index)+".vt"
	os.Remove(filename)
	f, err := os.Create(filename)
	if err!=nil {
		utils.Fatalf("Cant create VT dump file: %v",err)
	}
	defer f.Close()
	for _,transf:=range *transfers {
		str:=fmt.Sprintf("%v --> %v for %v, kind=%v, depth=%v (%v)\n",hex.EncodeToString(transf.From.Bytes()),hex.EncodeToString(transf.To.Bytes()),transf.Value.String(),transf.Kind,transf.Depth,transf.Err)
		bytes,err:=f.Write([]byte(str))
		if err!=nil {
			utils.Fatalf("Couldn't write %v bytes to VTs dump file: %v",bytes,err)
		}
	}
}
