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
	"os"
	"fmt"
	"math/big"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/robertkrimen/otto"
)
func js_local_blockchain_export_status() *otto.Object {
	return ethbot_instance.blockchain_export_status(&ethbot_instance.export)
}
func js_remote_blockchain_export_status() *otto.Object {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return ethbot_instance.empty_object()
	}
	var result Export_t
	err:=remote_EthBot.Call(&result,"ethbot_blockchainexportstatus");
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_blockchainexportstatus","error",err)
		return ethbot_instance.empty_object()
	} else {
		return ethbot_instance.blockchain_export_status(&result)
	}
}
func js_local_token_export_status() *otto.Object {
	return ethbot_instance.token_export_status(&ethbot_instance.token_export)
}
func js_remote_token_export_status() *otto.Object {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return ethbot_instance.empty_object()
	}
	var result TokenExport_t
	err:=remote_EthBot.Call(&result,"ethbot_tokenexportstatus");
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_tokenexportstatus","error",err)
		return ethbot_instance.empty_object()
	} else {
		return ethbot_instance.token_export_status(&result)
	}
}
func js_local_blockchain_export_stop() otto.Value {
	if ethbot_instance.blockchain_export_stop() {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_blockchain_export_stop() otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err:=remote_EthBot.Call(&result,"ethbot_blockchainexportstop");
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_blockchainexportstop","error",err)
		return otto.FalseValue()
	} else {
		if result==true {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_blockchain_export_start(arg_starting_block otto.Value,arg_ending_block otto.Value) otto.Value {

	p_starting_block,p_ending_block,err:=check_input_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	err=ethbot_instance.blockchain_export_start(p_starting_block,p_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	return otto.TrueValue();
}
func js_remote_blockchain_export_start(arg_starting_block otto.Value,arg_ending_block otto.Value) otto.Value {

	p_starting_block,p_ending_block,err:=check_input_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result error
	err=remote_EthBot.Call(&result,"ethbot_blockchainexportstart",p_starting_block,p_ending_block);
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_blockchainexportstart","error",err)
		return otto.FalseValue()
	} else {
		if result==nil {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_export_block_range(arg_starting_block otto.Value,arg_ending_block otto.Value) otto.Value {

	starting_block,ending_block,err:=check_input_export_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	ret:=ethbot_instance.export_block_range(starting_block,ending_block)
	if ret {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_export_block_range(arg_starting_block otto.Value,arg_ending_block otto.Value) otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	starting_block,ending_block,err:=check_input_export_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	var result bool
	err=remote_EthBot.Call(&result,"ethbot_exportblockrange",starting_block,ending_block);
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_exportblockrange","error",err)
		return otto.FalseValue()
	} else {
		if result {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_update_main_stats(p_block_num otto.Value) otto.Value {
	var err error
	block_num,err:=p_block_num.ToInteger()
	if (err!=nil) {
		err_text:="EthBot: invalid input value for `block_num` parameter: positive integer values or -1 are allowed"
		log.Error(err_text)
		return otto.FalseValue()
	}
	if ethbot_instance.update_main_stats(int(block_num)) {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_update_main_stats(p_block_num otto.Value) otto.Value {
	var err error
	block_num,err:=p_block_num.ToInteger()
	if (err!=nil) {
		err_text:="EthBot: invalid input value for `block_num` parameter: positive integer values or -1 are allowed"
		log.Error(err_text)
		return otto.FalseValue()
	}
	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}

	var result bool
	err=remote_EthBot.Call(&result,"ethbot_updatemainstats",block_num);
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method updatemainstats","error",err)
		return otto.FalseValue()
	} else {
		if result {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_fix_last_balances() otto.Value {
	if ethbot_instance.fix_last_balances() {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_fix_last_balances() otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err:=remote_EthBot.Call(&result,"ethbot_fixlastbalances");
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_blockchainexportstop","error",err)
		return otto.FalseValue()
	} else {
		if result==true {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_verify_last_balances() otto.Value {
	if ethbot_instance.verify_last_balances() {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_verify_last_balances() otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err:=remote_EthBot.Call(&result,"ethbot_verifylastbalances");
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_verifylastbalances","error",err)
		return otto.FalseValue()
	} else {
		if result==true {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_get_balance_if_exists(acct_addr string,p_block_num otto.Value) otto.Value {

	block_num,err:=p_block_num.ToInteger()
	if (err!=nil) {
		err_text:="EthBot: invalid input value for `block_num` parameter: positive integer values are allowed"
		log.Error(err_text)
		return otto.FalseValue()
	}
	addr:=common.HexToAddress(acct_addr)
	chain:=ethbot_instance.ethereum.BlockChain()
	block:=chain.GetBlockByNumber(uint64(block_num))
	statedb, err := chain.StateAt(block.Root())
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: can't get state: %v",err))
		return otto.FalseValue()
	}
	balance:=statedb.GetBalanceIfExists(addr)
	log.Info(fmt.Sprintf("EthBot: account %v block=%v",acct_addr,block_num),"balance",balance)
	return otto.TrueValue()
}
func js_local_writeout_state(filename string,p_block_num otto.Value) otto.Value {

	block_num,err:=p_block_num.ToInteger()
	if (err!=nil) {
		err_text:="EthBot: invalid input value for `block_num` parameter: positive integer values are allowed"
		log.Error(err_text)
		return otto.FalseValue()
	}
	chain:=ethbot_instance.ethereum.BlockChain()
	block:=chain.GetBlockByNumber(uint64(block_num))
	statedb, err := chain.StateAt(block.Root())
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: can't get state: %v",err))
		return otto.FalseValue()
	}
	f, err := os.Create(filename)
	if err!=nil {
		log.Info(fmt.Sprintf("Ethbot: error creating dump file '%v'",err.Error()))
		return otto.FalseValue()
	}
	defer f.Close()
	iterator:=statedb.GetNewIterator()
	log.Info(fmt.Sprintf("Exporting accounts to %v",filename))
	var data string
	var addr common.Address
	var balance *big.Int=big.NewInt(0)
	for statedb.GetNextAccount(iterator,&addr,balance) {
		data=fmt.Sprintf("%v\t%v\n",hex.EncodeToString(addr.Bytes()),balance.String())
		f.Write([]byte(data))
	}
	log.Info(fmt.Sprintf("Ethbot: export finished"))
	return otto.TrueValue()
}
func js_local_token_export_stop() otto.Value {
	if ethbot_instance.token_export_stop() {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_token_export_stop() otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err:=remote_EthBot.Call(&result,"ethbot_tokenexportstop");
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_tokenexportstop","error",err)
		return otto.FalseValue()
	} else {
		if result==true {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_token_export_start(arg_starting_block otto.Value,arg_ending_block otto.Value) otto.Value {

	p_starting_block,p_ending_block,err:=check_input_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	err=ethbot_instance.token_export_start(p_starting_block,p_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	return otto.TrueValue();
}
func js_remote_token_export_start(arg_starting_block otto.Value,arg_ending_block otto.Value) otto.Value {

	p_starting_block,p_ending_block,err:=check_input_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result error
	err=remote_EthBot.Call(&result,"ethbot_tokenexportstart",p_starting_block,p_ending_block);
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_tokenexportstart","error",err)
		return otto.FalseValue()
	} else {
		if result==nil {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_fix_deleted_flag(arg_block_num otto.Value) otto.Value {
	block_num,err:=verif_check_input_block_num(arg_block_num)
	if err!=nil {
		return otto.FalseValue()
	}
	sql_update_deleted_attribute4all_accounts(block_num)
	return otto.TrueValue()
}
func js_local_verify_sql_data_1(arg_block_num otto.Value) otto.Value {

	block_num,err:=verif_check_input_block_num(arg_block_num)
	if (err!=nil) {
		return otto.FalseValue()
	}
	result:=verify_SQL_data(0,block_num,block_num)
	if (result) {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_local_verify_sql_data_2(arg_block_num otto.Value) otto.Value {

	block_num,err:=verif_check_input_block_num(arg_block_num)
	if (err!=nil) {
		return otto.FalseValue()
	}
	result:=verify_SQL_data(VERIFICATION_SQL,block_num,block_num)
	if (result) {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_verify_sql_data_1(arg_block_num otto.Value) otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}

	block_num,err:=verif_check_input_block_num(arg_block_num)
	if (err!=nil) {
		return otto.FalseValue()
	}
	var result bool
	err=remote_EthBot.Call(&result,"ethbot_verifysqldata1",block_num);
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_verifysqldata1","error",err)
		return otto.FalseValue()
	} else {
		if result {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_remote_verify_sql_data_2(arg_block_num otto.Value) otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}

	block_num,err:=verif_check_input_block_num(arg_block_num)
	if (err!=nil) {
		return otto.FalseValue()
	}
	var result bool
	err=remote_EthBot.Call(&result,"ethbot_verifysqldata2",block_num);
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_verifysqldata2","error",err)
		return otto.FalseValue()
	} else {
		if result {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_stop_verification() otto.Value {
	stop_verification()
	stop_verification_completed()
	return otto.TrueValue();
}
func js_remote_stop_verification() otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err:=remote_EthBot.Call(&result,"ethbot_stopverification");
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_stopverification","error",err)
		return otto.FalseValue()
	} else {
		return otto.TrueValue()
	}
}
func js_local_verification_status() *otto.Object {
	return verification_status(&ethbot_instance.verification)
}
func js_remote_verification_status() *otto.Object {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return ethbot_instance.empty_object()
	}
	var result Verification_t
	err:=remote_EthBot.Call(&result,"ethbot_verificationstatus");
	if (err!=nil) {
		log.Error("EthBot: error calling ethbot_verificationstatus","error",err)
		return ethbot_instance.empty_object()
	} else {
		return verification_status(&result)
	}
}
func js_local_verify_account(arg_account_address otto.Value,arg_block_num otto.Value) otto.Value {
	// returns true if all balances and values match the state DB

	account_addr_str,block_num,err:=check_input_verify_account(arg_account_address,arg_block_num)
	if (err!=nil) {
		return otto.FalseValue()
	}
	res:=verify_single_account(account_addr_str,block_num)
	if res {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_verify_account(arg_account_address otto.Value,arg_block_num otto.Value) otto.Value {

	account_addr_str,block_num,err:=check_input_verify_account(arg_account_address,arg_block_num)
	if (err!=nil) {
		return otto.FalseValue()
	}
	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return  otto.FalseValue()
	}
	var result bool
	err=remote_EthBot.Call(&result,"ethbot_verifyaccount",account_addr_str,block_num);
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_verify_account","error",err)
		return otto.FalseValue()
	} else {
		if (result) {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_verify_all_accounts(arg_block_num otto.Value) otto.Value {
	block_num,err:=check_input_verify_all_accounts(arg_block_num)
	if (err!=nil) {
		return otto.FalseValue()
	}
	res:=verify_all_accounts(block_num)
	if res {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_verify_all_accounts(arg_block_num otto.Value) otto.Value {
	block_num,err:=check_input_verify_all_accounts(arg_block_num)
	if (err!=nil) {
		return otto.FalseValue()
	}
	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err=remote_EthBot.Call(&result,"ethbot_verifyallaccounts",block_num);
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_verifyallaccounts","error",err)
		return otto.FalseValue()
	} else {
		if result {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_verify_all_tokens(arg_block_num otto.Value) otto.Value {
	block_num,err:=check_input_verify_all_accounts(arg_block_num)
	if (err!=nil) {
		return otto.FalseValue()
	}
	res:=verify_all_tokens(block_num)
	if res {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_verify_all_tokens(arg_block_num otto.Value) otto.Value {
	block_num,err:=check_input_verify_all_accounts(arg_block_num)
	if (err!=nil) {
		return otto.FalseValue()
	}
	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err=remote_EthBot.Call(&result,"ethbot_verifyalltokens",block_num);
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_verifyalltokens","error",err)
		return otto.FalseValue()
	} else {
		if result {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_verify_token(arg_contract_address otto.Value,arg_block_num otto.Value) otto.Value {
	contract_addr_str,block_num,err:=check_input_verify_account(arg_contract_address,arg_block_num)
	if (err!=nil) {
		return otto.FalseValue()
	}
	res:=verify_token(contract_addr_str,block_num)
	if res {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_verify_token(arg_contract_address otto.Value,arg_block_num otto.Value) otto.Value {
	contract_addr_str,block_num,err:=check_input_verify_account(arg_contract_address,arg_block_num)
	if (err!=nil) {
		return otto.FalseValue()
	}
	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err=remote_EthBot.Call(&result,"ethbot_verifytoken",contract_addr_str,block_num);
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_verifytoken","error",err)
		return otto.FalseValue()
	} else {
		if result {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_fix_vt_balances(arg_account_address otto.Value) otto.Value {

	var err error
	account_addr_str,err:=arg_account_address.ToString()
	if (err!=nil) {
		err_text:="Invalid input value for `account_address` parameter: does not look like a string"
		log.Error(err_text)
		otto.FalseValue()
	}
	if len(account_addr_str)!=(common.AddressLength*2) {
		if account_addr_str!="0" {
			err_text:="Invalid input value for `account_address` parameter: 40 character HEX string is required, without 0x prepended"
			log.Error(err_text)
			return otto.FalseValue()
		} else {
			// single zero is a special case for NONEXISTENT account
		}
	}
	res:=fix_account_vt_balances(account_addr_str)
	if res {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_local_alarms_on() otto.Value {
	if ethbot_instance.alarms_on() {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_alarms_on() otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err:=remote_EthBot.Call(&result,"ethbot_alarmson");
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_alarmson","error",err)
		return otto.FalseValue()
	} else {
		if result==true {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_alarms_off() otto.Value {
	if ethbot_instance.alarms_off() {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_alarms_off() otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err:=remote_EthBot.Call(&result,"ethbot_alarmsoff");
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_alarmsoff","error",err)
		return otto.FalseValue()
	} else {
		if result==true {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
