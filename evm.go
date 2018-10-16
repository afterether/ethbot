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
	"math/big"
	"fmt"
	"encoding/hex"
	"strconv"
	"bytes"
	"unicode/utf8"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
    "github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
)
const debug_token_discovery bool=false
func decode_token_input(input []byte) (int,string,int,[]interface{}) { // output: tokop_code,method_name,ERC standard code, input array
	var token_op int=TOKENOP_UNKNOWN
	if len(input)>4 {
		method_sig:=input[:4]
		input_data_bytes:=input[4:]
		var	method_name string
		if bytes.Equal(erc20_transfer_method_signature,method_sig) {	// ERC20::transfer
			method_name="transfer"
			token_op=TOKENOP_TRANSFER
		}
		if bytes.Equal(erc20_approve_method_signature,method_sig) {	// ERC20::transfer
			method_name="approve"
			token_op=TOKENOP_APPROVAL
		}
		if bytes.Equal(erc20_transfer_from_method_signature,method_sig) {	// ERC20::transfer
			method_name="transferFrom"
			token_op=TOKENOP_TRANSFER_FROM
		}
		if len(method_name)>0 {
			m,exists:=erc20_token_abi.Methods[method_name]
			if !exists {
				utils.Fatalf(fmt.Sprintf(`No '%v' method in ERC20 ABI`,method_name))
			}
			method_input,err:=m.Inputs.UnpackValues(input_data_bytes)
			if err==nil {
				return token_op,method_name,1,method_input
			}
		}
	}
	return token_op,"",0,nil
}
func record_simulation_token_amounts_before(chain *core.BlockChain,statedb *state.StateDB,block *types.Block,tx *types.Transaction,contract_addr, tx_sender *common.Address,ERC_iface int,method_name string,decoded_input []interface{}, sim_res *Simulation_result_t) {

	if ERC_iface==1 {
		if method_name=="transfer" {
			var to common.Address = decoded_input[0].(common.Address)
			var tok_value *big.Int = decoded_input[1].(*big.Int)
			bal_from:=get_ERC20_token_balance_from_EVM(chain,statedb,block,contract_addr,tx_sender)
			if bal_from!=nil {
				sim_res.Tok_from_old_balance=bal_from.String()
			}
			bal_to:=get_ERC20_token_balance_from_EVM(chain,statedb,block,contract_addr,&to)
			if bal_to!=nil {
				sim_res.Tok_to_old_balance=bal_to.String()
			}
			sim_res.Tok_value=tok_value.String()
			sim_res.Tok_from_address=hex.EncodeToString(tx_sender.Bytes())
			sim_res.Tok_to_address=hex.EncodeToString(to.Bytes())
		}
		if method_name=="approve" {
			var to common.Address = decoded_input[0].(common.Address)
			var tok_value *big.Int = decoded_input[1].(*big.Int)
			bal_from:=get_ERC20_token_balance_from_EVM(chain,statedb,block,contract_addr,tx_sender)
			if bal_from!=nil {
				sim_res.Tok_from_old_balance=bal_from.String()
				bal_from.Sub(bal_from,tok_value)
				sim_res.Tok_from_new_balance=bal_from.String()
			}
			bal_to:=get_ERC20_token_balance_from_EVM(chain,statedb,block,contract_addr,&to)
			if bal_to!=nil {
				sim_res.Tok_to_old_balance=bal_to.String()
				bal_to.Add(bal_to,tok_value)
				sim_res.Tok_to_new_balance=bal_to.String()
			}
			sim_res.Tok_value=tok_value.String()
			sim_res.Tok_from_address=hex.EncodeToString(tx_sender.Bytes())
			sim_res.Tok_to_address=hex.EncodeToString(to.Bytes())
		}
		if method_name=="transferFrom" {
			var from common.Address = decoded_input[0].(common.Address)
			var to common.Address = decoded_input[1].(common.Address)
			var tok_value *big.Int = decoded_input[2].(*big.Int)
			bal_from:=get_ERC20_token_balance_from_EVM(chain,statedb,block,contract_addr,&from)
			if bal_from!=nil {
				sim_res.Tok_from_old_balance=bal_from.String()
			}
			bal_to:=get_ERC20_token_balance_from_EVM(chain,statedb,block,contract_addr,&to)
			if bal_to!=nil {
				sim_res.Tok_to_old_balance=bal_to.String()
			}
			sim_res.Tok_value=tok_value.String()
			sim_res.Tok_from_address=hex.EncodeToString(from.Bytes())
			sim_res.Tok_to_address=hex.EncodeToString(to.Bytes())
		}
	}
}
func record_simulation_token_amounts_after(chain *core.BlockChain,statedb *state.StateDB,block *types.Block,tx *types.Transaction,contract_addr, tx_sender *common.Address,ERC_iface int,method_name string,decoded_input []interface{}, sim_res *Simulation_result_t) {

	if ERC_iface==1 {
		if method_name=="transfer" {
			var to common.Address = decoded_input[0].(common.Address)
			bal_from:=get_ERC20_token_balance_from_EVM(chain,statedb,block,contract_addr,tx_sender)
			if bal_from!=nil {
				sim_res.Tok_from_new_balance=bal_from.String()
			}
			bal_to:=get_ERC20_token_balance_from_EVM(chain,statedb,block,contract_addr,&to)
			if bal_to!=nil {
				sim_res.Tok_to_new_balance=bal_to.String()
			}
		}
		if method_name=="approve" {
			// all variables were set in before() call
		}
		if method_name=="transferFrom" {
			var from common.Address = decoded_input[0].(common.Address)
			var to common.Address = decoded_input[1].(common.Address)
			bal_from:=get_ERC20_token_balance_from_EVM(chain,statedb,block,contract_addr,&from)
			if bal_from!=nil {
				sim_res.Tok_from_new_balance=bal_from.String()
			}
			bal_to:=get_ERC20_token_balance_from_EVM(chain,statedb,block,contract_addr,&to)
			if bal_to!=nil {
				sim_res.Tok_to_new_balance=bal_to.String()
			}
		}
	}
}
func record_token_amount_differences(simres *Simulation_result_t) {
	old_from:=big.NewInt(0)
	old_from.SetString(simres.Tok_from_old_balance,10)
	new_from:=big.NewInt(0)
	new_from.SetString(simres.Tok_from_new_balance,10)
	from_diff:=big.NewInt(0)
	from_diff.Sub(old_from,new_from)
	from_diff.Abs(from_diff)
	simres.Tok_from_diff=from_diff.String()

	old_to:=big.NewInt(0)
	old_to.SetString(simres.Tok_to_old_balance,10)
	new_to:=big.NewInt(0)
	new_to.SetString(simres.Tok_to_new_balance,10)
	to_diff:=big.NewInt(0)
	to_diff.Sub(old_to,new_to)
	to_diff.Abs(to_diff)
	simres.Tok_to_diff=to_diff.String()
}
func record_output(ERC_iface int,method_name string,vm_ret []byte, sim_res *Simulation_result_t) {

	if ERC_iface==1 {
		output_bool:=new(bool)
		err:=erc20_token_abi.Unpack(&output_bool,method_name,vm_ret)
		if err!=nil {
			log.Info(fmt.Sprintf("EVM output: Unpack error for %v: %v",method_name,err))
		} else {
			sim_res.Tok_ret_val=*output_bool
		}
	}
}
func (bot *EthBot_t) Simulate_TX(signed_tx string,no_validation bool) Simulation_result_t {
	var output Simulation_result_t
	var tx *types.Transaction
	raw_tx,err:=hex.DecodeString(signed_tx)
	if err!=nil {
		output.Error=err.Error()
		log.Info(fmt.Sprintf("TX Simulation: decode raw string failed: %v",err))
		return output
	}
	unused1:=big.NewInt(0); unused2:=big.NewInt(0); unused3:=big.NewInt(0); unused4:=big.NewInt(0); unused5:=big.NewInt(0)
	err=rlp.DecodeBytes(raw_tx, &tx)
	if err!=nil {
		output.Error=err.Error()
		log.Info(fmt.Sprintf("TX Simulation: decode bytes failed: %v",err))
		return output
	}
	to:=tx.To()
	output.To_address=hex.EncodeToString(to.Bytes())
	value:=tx.Value();
	output.Value=value.String()
	chain_id:=tx.ChainId()
	signer:=types.NewEIP155Signer(chain_id)
	from,err:=signer.Sender(tx)
	if err!=nil {
		output.Error=err.Error()
		return output
	}
	txpool:=bot.ethereum.TxPool()
	output.Gas_price_threshold=txpool.GasPrice().String()
	if no_validation {
		// in this mode simlation does not verify GasPrice minimums, bump for resending, etc..
	} else {
		// txpool validation is applied
		err=txpool.ValidateTx(tx,false)
		if err!=nil {
			output.Error=err.Error()
			return output
		}
		err=txpool.SimAdd2Pool(tx,&from)
		if err!=nil {
			output.Error=err.Error()
			return output
		}
	}

	log.Info(fmt.Sprintf("TX from %v for %v has no errors: %v",hex.EncodeToString(from.Bytes()),tx.Value().String(),err))
	output.From_address=hex.EncodeToString(from.Bytes())
	blockchain := bot.ethereum.BlockChain()
	bc_cfg:=blockchain.Config()
	block:=blockchain.CurrentBlock()
	if block==nil {
		output.Error="Block is null"
		return output
	}
    if block.NumberU64()== uint64(0) {
		output.Error="Can't use genesis block as starting block, mine some blocks"
		return output
	}
	gp:=new(core.GasPool).AddGas(block.GasLimit())
	vm_cfg:=vm.Config{
        Debug: false,
	}
	hdr:=block.Header()
	null_addr:=common.Address{}
	hdr.Coinbase.SetBytes(null_addr.Bytes())// clear miner address, let the mining reward go to nowhere
	block=types.NewBlock(hdr,nil,nil,nil)
    statedb, err := blockchain.StateAt(block.Root())
	if err!=nil {
		output.Error=err.Error()
		return output
	}
	output.Signed_tx=signed_tx
	output.Gas_limit=strconv.FormatUint(tx.Gas(),10)
	output.Gas_price=tx.GasPrice().String()
	old_from_balance:=statedb.GetBalance(from)
	old_to_balance:=statedb.GetBalance(*to)
	output.From_old_balance=old_from_balance.String()
	output.To_old_balance=old_to_balance.String()

	var method_name string
	var ERC_code int
	var decoded_input []interface{}
	token_id:=lookup_token_by_addr(output.To_address)
	if (token_id>0) {	// it's a token transfer
		_,method_name,ERC_code,decoded_input=decode_token_input(tx.Data())
		log.Info(fmt.Sprintf("returning method name: %v",method_name))
		record_simulation_token_amounts_before(blockchain,statedb,block,tx,to,&from,ERC_code,method_name,decoded_input,&output)
	}
	var ret_val []byte
	vm_err4ethbot:=new(error)
	vm_VTs4ethbot:=make([]*vm.Ethbot_EVM_VT_t,0,64)
	msg,err:=tx.AsMessage(signer)
	if err!=nil {
		output.Error=err.Error()
		return output
	}
	evm_ctx:=core.NewEVMContext(msg,block.Header(),blockchain,nil)
	ethvirmac:=vm.NewEVM(evm_ctx,statedb,bc_cfg,vm_cfg,&vm_VTs4ethbot,vm_err4ethbot)
	ret_val,_,gas,failed,tx_err:=core.ApplyMessage(ethvirmac,msg,gp,unused1,unused2,unused3,unused4,unused5)
	_=failed
	new_from_balance:=statedb.GetBalance(from)
	new_to_balance:=statedb.GetBalance(*to)
	output.From_new_balance=new_from_balance.String()
	output.To_new_balance=new_to_balance.String()
	to_exist_balance:=statedb.GetBalanceIfExists(*to)
	minus_one:=big.NewInt(-1)
	if minus_one.Cmp(to_exist_balance)==0 {
		output.New_acct=true
	}
	from_diff:=big.NewInt(0)
	from_diff.Sub(old_from_balance,new_from_balance)
	from_diff.Abs(from_diff)
	to_diff:=big.NewInt(0)
	to_diff.Sub(new_to_balance,old_to_balance)
	to_diff.Abs(to_diff)
	output.From_diff=from_diff.String()
	output.To_diff=to_diff.String()
	output.Gas_used=strconv.FormatUint(gas,10)
	if (token_id>0) {	// it's a token transfer
		if *vm_err4ethbot==nil {
			record_simulation_token_amounts_after(blockchain,statedb,block,tx,to,&from,ERC_code,method_name,decoded_input,&output)
			record_token_amount_differences(&output)
			record_output(ERC_code,method_name,ret_val,&output)
		}
	}
	if tx_err!=nil {
		output.Error=tx_err.Error()
	}
	if *vm_err4ethbot!=nil {
		output.VMError=(*vm_err4ethbot).Error()
	}
	return output
}
func (this *Simulation_result_t) dump() {
	log.Info(fmt.Sprintf("Status: %v",this.Status))
	log.Info(fmt.Sprintf("From_address: %v",this.From_address))
	log.Info(fmt.Sprintf("To_address: %v",this.To_address))
	log.Info(fmt.Sprintf("From_old_balance: %v",this.From_old_balance))
	log.Info(fmt.Sprintf("To_old_balance: %v",this.To_old_balance))
	log.Info(fmt.Sprintf("From_new_balance: %v",this.From_new_balance))
	log.Info(fmt.Sprintf("To_new_balance: %v",this.To_new_balance))
	log.Info(fmt.Sprintf("From_diff: %v",this.From_diff))
	log.Info(fmt.Sprintf("To_diff: %v",this.To_diff))
	log.Info(fmt.Sprintf("Value: %v",this.Value))
	log.Info(fmt.Sprintf("Tok_from_address: %v",this.Tok_from_address))
	log.Info(fmt.Sprintf("Tok_to_address: %v",this.Tok_to_address))
	log.Info(fmt.Sprintf("Tok_from_old_balance: %v",this.Tok_from_old_balance))
	log.Info(fmt.Sprintf("Tok_from_new_balance: %v",this.Tok_from_new_balance))
	log.Info(fmt.Sprintf("Tok_from_diff: %v",this.Tok_from_diff))
	log.Info(fmt.Sprintf("Tok_to_old_balance: %v",this.Tok_to_old_balance))
	log.Info(fmt.Sprintf("Tok_to_new_balance: %v",this.Tok_to_new_balance))
	log.Info(fmt.Sprintf("Tok_to_diff: %v",this.Tok_to_diff))
	log.Info(fmt.Sprintf("Tok_value: %v",this.Tok_value))
	log.Info(fmt.Sprintf("Tok_ret_val: %v",this.Tok_ret_val))
	log.Info(fmt.Sprintf("Signed_tx: %v",this.Signed_tx))
	log.Info(fmt.Sprintf("Gas_used: %v",this.Gas_used))
	log.Info(fmt.Sprintf("Gas_limit: %v",this.Gas_limit))
	log.Info(fmt.Sprintf("Gas_price: %v",this.Gas_price))
	log.Info(fmt.Sprintf("Gas_price_threshold: %v",this.Gas_price_threshold))
	log.Info(fmt.Sprintf("Error: %v",this.Error))
	log.Info(fmt.Sprintf("Token_transfeR: %v",this.Token_transfer))
	log.Info(fmt.Sprintf("New_acct: %v",this.New_acct))
}
func get_ERC20_token_balance_from_EVM(chain *core.BlockChain,statedb *state.StateDB, block *types.Block,contract_address,querying_addr *common.Address) *big.Int {

	var (
		err		error
		input	[]byte
	)
	vm_cfg:=vm.Config{
        Debug: false,
	}
	bconf:=chain.Config()
	unused1:=big.NewInt(0); unused2:=big.NewInt(0); unused3:=big.NewInt(0); unused4:=big.NewInt(0); unused5:=big.NewInt(0)
	value:=big.NewInt(0)
	gas_limit:=uint64(50000000)
	gas_price:=big.NewInt(1)
	fake_src_addr:=common.HexToAddress("1999999999999999999999999999999999999991")
	fake_balance:=big.NewInt(0);
	fake_balance.SetString("9999999999999999999999999999",10)
	statedb.AddBalance(fake_src_addr,fake_balance)		// add a fake account with huge balance so we have money to pay for gas to execute instructions on the EVM

	// encode input for retrieving token balance
	input, err = erc20_token_abi.Pack("balanceOf",querying_addr)
	if err!=nil {
		utils.Fatalf("can't pack balanceOf input: %v",err)
	}
	msg:=types.NewMessage(fake_src_addr,contract_address,0,value,gas_limit,gas_price,input,false)
	evm_ctx:=core.NewEVMContext(msg,block.Header(),chain,nil)
	vm_err4ethbot:=new(error)
	vm_VTs4ethbot:=make([]*vm.Ethbot_EVM_VT_t,0,64)  // Ethbot variables are usless here, we are just complying with calling requirements
	ethvirmac:=vm.NewEVM(evm_ctx,statedb,bconf,vm_cfg,&vm_VTs4ethbot,vm_err4ethbot)
	gp:=new(core.GasPool).AddGas(math.MaxUint64)
	ret,gas,_,failed,err:=core.ApplyMessage(ethvirmac,msg,gp,unused1,unused2,unused3,unused4,unused5)
	if (failed) {
		log.Info(fmt.Sprintf("get_ERC20_token_balance: vm err for symbol: %v, failed=%v",err,failed))
	}
	if err!=nil {
		log.Info(fmt.Sprintf("get_ERC20_token_balance: getting 'balanceOf' caused error in vm: %v",err))
	}
	if len(ret)==0 {
		return nil
	}
	balance:=big.NewInt(0)
	if !((err!=nil) || (failed)) {
		err=erc20_token_abi.Unpack(&balance,"balanceOf",ret)
		if err!=nil {
			utils.Fatalf("Can't upack balanceOf output from the EVM: %v",err)
		}
	}
	_=gas
	return balance
}
func find_account_with_non_zero_balance(chain *core.BlockChain,block *types.Block,statedb *state.StateDB, events []*Event_t,contract_address,owner_address *common.Address) (bool,bool,*common.Address) {
	// searches for an account that holds some amount of tokens, this account will be used to make transfer operations during contract testing
	// otputs: 1) account with balance found, 2) initial transfer event emitted, 3) holder of tokens with non-zero balance
	transfer_event_found:=false
	for _,elog:=range events {
		trsf:=event2transfer_record(elog)
		if trsf!=nil {
			if trsf.Kind==TOKENOP_TRANSFER {
				transfer_event_found=true
			}
		}
	}
	for _,elog:=range events {
		trsf:=event2transfer_record(elog)
		if trsf!=nil {
			balance:=get_ERC20_token_balance_from_EVM(chain,statedb, block,contract_address,&trsf.To)
			if balance!=nil {
				if balance.Cmp(zero)>0 {
					addr:=&common.Address{}
					addr.SetBytes(trsf.To.Bytes())
					return true,transfer_event_found,addr
				}
			}
			balance=get_ERC20_token_balance_from_EVM(chain,statedb, block ,contract_address,&trsf.From)
			if balance!=nil {
				if balance.Cmp(zero)>0 {
					addr:=&common.Address{}
					addr.SetBytes(trsf.From.Bytes())
					return true,transfer_event_found,addr
				}
			}
		}
	}
	balance:=get_ERC20_token_balance_from_EVM(chain,statedb, block ,contract_address,contract_address)
	if balance!=nil {
		if balance.Cmp(zero)>0 {
			addr:=&common.Address{}
			addr.SetBytes(contract_address.Bytes())
			return true,transfer_event_found,addr
		}
	}
	balance=get_ERC20_token_balance_from_EVM(chain,statedb, block ,contract_address,owner_address)
	if balance!=nil {
		if balance.Cmp(zero)>0 {
			addr:=&common.Address{}
			addr.SetBytes(owner_address.Bytes())
			return true,transfer_event_found,addr
		}
	}
	return false,transfer_event_found,nil
}
func (bot *EthBot_t) extract_token_info(chain *core.BlockChain,block *types.Block,statedb *state.StateDB, bconf *params.ChainConfig, owner_address,contract_address *common.Address, event_logs []*Event_t, tok *Token_t,tok_info *Token_info_t) bool {
// By calling contract functions explores the contract for presence of different methods of different standards, returns true if the contract is a token, and false if it isn't
	if *contract_address==(common.Address{}) {
		return false
	}
	var (
		err		error
		input	[]byte
	)
	vm_cfg:=vm.Config{
        Debug: false,
	}
	unused1:=big.NewInt(0); unused2:=big.NewInt(0); unused3:=big.NewInt(0); unused4:=big.NewInt(0); unused5:=big.NewInt(0)
	value:=big.NewInt(0)
	gas_limit:=uint64(50000000)
	gas_price:=big.NewInt(1)
	fake_src_addr:=common.HexToAddress("9999999999999999999999999999999999999999")
	fake_balance:=big.NewInt(0);
	fake_balance.SetString("9999999999999999999999999999",10)
	statedb.AddBalance(fake_src_addr,fake_balance)		// add a fake account with huge balance so we have money to pay for gas to execute instructions on the EVM

	tok_info.fully_discovered=true				// for now we discover fully 
	///////////// Test for ERC 20 token
	// get token's total supply. If a contract has 'totalSupply' then it is a token, otherwise it isn't
	input, err = erc20_token_abi.Pack("totalSupply")
	if err!=nil {
		utils.Fatalf("can't pack totalSupply input: %v",err)
	}
	msg:=types.NewMessage(fake_src_addr,contract_address,0,value,gas_limit,gas_price,input,false)
	block_header:=block.Header()
	evm_ctx:=core.NewEVMContext(msg,block_header,chain,nil)
	vm_err4ethbot:=new(error)
	vm_VTs4ethbot:=make([]*vm.Ethbot_EVM_VT_t,0,2048)  // Ethbot variables are usless here, we are just complying with calling requirements
	ethvirmac:=vm.NewEVM(evm_ctx,statedb,bconf,vm_cfg,&vm_VTs4ethbot,vm_err4ethbot)
	gp:=new(core.GasPool).AddGas(math.MaxUint64)
	ret,gas,_,failed,err:=core.ApplyMessage(ethvirmac,msg,gp,unused1,unused2,unused3,unused4,unused5)
	if debug_token_discovery {
		log.Info(fmt.Sprintf("Checking for totalSupply() ret=%v (len=%v), failed=%v, err=%v",hex.EncodeToString(ret),len(ret),failed,err))
	}
	if ((err!=nil) || (failed)) {
		if (failed) {
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+`totalSupply() method fails at execution`
		}
		if err!=nil {
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+`totalSupply() method returns an error: `+err.Error()
		}
	} else {
		if len(ret)==0 {
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+`totalSupply() method doesn't exist, empty output returned`
		} else {
			tok.total_supply=big.NewInt(0)
			output_int:=big.NewInt(0)
			err=erc20_token_abi.Unpack(&output_int,"totalSupply",ret)
			if err!=nil {
				tok_info.nc_ERC20=true
				log.Info(fmt.Sprintf("Contract %v: can;t upack output from totalSupply: %v",hex.EncodeToString(contract_address.Bytes()),err))
				if len(tok_info.e_ERC20)>0 {
					tok_info.e_ERC20=tok_info.e_ERC20+", "
				}
				tok_info.e_ERC20=tok_info.e_ERC20+`unpacking output from totalSupply() returns an error: `+err.Error()
			} else {
				tok.total_supply.Set(output_int)
				tok_info.m_ERC20_total_supply=true
			}
		}
	}
	//Checking balanceOf function
	input, err = erc20_token_abi.Pack("balanceOf",fake_src_addr)
	if err!=nil {
		utils.Fatalf("packing input to check for balanceOf failed: %v",err)
	}
	msg=types.NewMessage(fake_src_addr,contract_address,0,value,gas_limit,gas_price,input,false)
	ret,gas,_,failed,err=core.ApplyMessage(ethvirmac,msg,gp,unused1,unused2,unused3,unused4,unused5)
	if debug_token_discovery {
		log.Info(fmt.Sprintf("Checking for balanceOf() ret=%v (len=%v), failed=%v, err=%v",hex.EncodeToString(ret),len(ret),failed,err))
	}
	if ((err!=nil) || (failed)) {
		if (failed) {
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+`balanceOf() method fails at execution`
		}
		if err!=nil {
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+`balanceOf() method returns an error: `+err.Error()
		}
	} else {
		if len(ret)==0 {
			log.Info(fmt.Sprintf("Contract %v, balanceOf function doesn't exist",hex.EncodeToString(contract_address.Bytes())))
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+`balanceOf() method doesn't exist null output returned`
		} else {
			output_int:=big.NewInt(0)
			err=erc20_token_abi.Unpack(&output_int,"balanceOf",ret)
			if err!=nil {
				tok_info.nc_ERC20=true
				log.Info(fmt.Sprintf("Contract %v: can;t upack output from balanceOf: %v",hex.EncodeToString(contract_address.Bytes()),err))
				if len(tok_info.e_ERC20)>0 {
					tok_info.e_ERC20=tok_info.e_ERC20+", "
				}
				tok_info.e_ERC20=tok_info.e_ERC20+`unpack of balanceOf() method output returns an error: `+err.Error()
				return false
			} else {
				if output_int.Cmp(zero)<0 {
					log.Info(fmt.Sprintf("Contract %v, negative balance in balaneOf output detected",hex.EncodeToString(contract_address.Bytes())))
					tok_info.nc_ERC20=true
					if len(tok_info.e_ERC20)>0 {
						tok_info.e_ERC20=tok_info.e_ERC20+", "
					}
					tok_info.e_ERC20=tok_info.e_ERC20+"balanceOf() method returns negative value"
					return false;		// negative balance not allowed
				} else {
					tok_info.m_ERC20_balance_of=true
				}
			}
		}
	}
	// Getting token Name
	input, err = erc20_token_abi.Pack("name")
	if err!=nil {
		utils.Fatalf("packing input for name failed: %v",err)
	}
	msg=types.NewMessage(fake_src_addr,contract_address,0,value,gas_limit,gas_price,input,false)
	ret,gas,_,failed,err=core.ApplyMessage(ethvirmac,msg,gp,unused1,unused2,unused3,unused4,unused5)
	if debug_token_discovery {
		log.Info(fmt.Sprintf("Checking for name() ret=%v (len=%v), failed=%v, err=%v",hex.EncodeToString(ret),len(ret),failed,err))
	}
	if ((err!=nil) || (failed)) {
		if (failed) {
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+`name() method fails at execution`
		}
		if err!=nil {
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+`name() method returns an error: `+err.Error()
		}
	} else {
		if len(ret)==0 {
			tok_info.nc_ERC20=true
			log.Info(fmt.Sprintf("Contract %v, name function returns nil",hex.EncodeToString(contract_address.Bytes())))
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+`name() method doesn't exist`
		} else {
			var output = new(string)
			err=erc20_token_abi.Unpack(output,"name",ret)
			if err!=nil {
				tok_info.nc_ERC20=true
				if len(tok_info.e_ERC20)>0 {
					tok_info.e_ERC20=tok_info.e_ERC20+", "
				}
				tok_info.e_ERC20=tok_info.e_ERC20+"name not found: "+err.Error()
			} else {
				if utf8.Valid([]byte(*output)) {
					tok.name=*output
					tok_info.name=*output
					log.Info(fmt.Sprintf("token name=%v",tok.name))
					tok_info.m_ERC20_name=true
				} else {
					tok_info.nc_ERC20=true
					if len(tok_info.e_ERC20)>0 {
						tok_info.e_ERC20=tok_info.e_ERC20+", "
					}
					tok_info.e_ERC20=tok_info.e_ERC20+"name contains invalid unicode characters"
				}
			}
		}
	}

	// Getting token Symbol
	input, err = erc20_token_abi.Pack("symbol")
	if err!=nil {
		utils.Fatalf("packing input for symbol failed: %v",err)
	}
	msg=types.NewMessage(fake_src_addr,contract_address,0,value,gas_limit,gas_price,input,false)
	ret,gas,_,failed,err=core.ApplyMessage(ethvirmac,msg,gp,unused1,unused2,unused3,unused4,unused5)
	if debug_token_discovery {
		log.Info(fmt.Sprintf("Checking for symbol() ret=%v (len=%v), failed=%v, err=%v",hex.EncodeToString(ret),len(ret),failed,err))
	}
	if (err!=nil) || failed {
		tok_info.nc_ERC20=true
		if len(tok_info.e_ERC20)>0 {
			tok_info.e_ERC20=tok_info.e_ERC20+", "
		}
		tok_info.e_ERC20=tok_info.e_ERC20+"symbol not found"
		if err!=nil {
			tok_info.e_ERC20=tok_info.e_ERC20+": "+err.Error()
		}
	} else {
		if len(ret)==0 {
			tok_info.nc_ERC20=true
			log.Info(fmt.Sprintf("Contract %v, symbol function returns nil",hex.EncodeToString(contract_address.Bytes())))
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+"symbol not found"
		} else {
			var output = new(string)
			if !((err!=nil) || (failed)) {
				err=erc20_token_abi.Unpack(output,"symbol",ret)
				if err!=nil {
					log.Info(fmt.Sprintf("Contract %v: can;t upack symbol: %v",hex.EncodeToString(contract_address.Bytes()),err))
					tok_info.nc_ERC20=true
					if len(tok_info.e_ERC20)>0 {
						tok_info.e_ERC20=tok_info.e_ERC20+", "
					}
					tok_info.e_ERC20=tok_info.e_ERC20+"symbol not found: "+err.Error()
				} else {
					log.Info(fmt.Sprintf("symbol=%v",*output))
					if utf8.Valid([]byte(*output)) {
						tok.symbol=*output
						tok_info.symbol=*output
						tok_info.m_ERC20_symbol=true
					} else {
						tok_info.nc_ERC20=true
						if len(tok_info.e_ERC20)>0 {
							tok_info.e_ERC20=tok_info.e_ERC20+", "
						}
						tok_info.e_ERC20=tok_info.e_ERC20+"symbol contains invalid unicode characters"
					}
				}
			}
		}
	}
	// Getting token Decimals 
	input, err = erc20_token_abi.Pack("decimals")
	if err!=nil {
		utils.Fatalf("packing input for decimals failed: %v",err)
	}
	msg=types.NewMessage(fake_src_addr,contract_address,0,value,gas_limit,gas_price,input,false)
	ret,gas,_,failed,err=core.ApplyMessage(ethvirmac,msg,gp,unused1,unused2,unused3,unused4,unused5)
	if debug_token_discovery {
		log.Info(fmt.Sprintf("Checking for decimals() ret=%v (len=%v), failed=%v, err=%v",hex.EncodeToString(ret),len(ret),failed,err))
	}
	if (err!=nil) || (failed) {
		if (failed) {
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+`decimals() method fails at execution`
		}
		if err!=nil {
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+`decimals() method returns an error: `+err.Error()
		}
	} else {
		if len(ret)==0 {
			tok_info.nc_ERC20=true
			log.Info(fmt.Sprintf("Contract %v, name function returns nil",hex.EncodeToString(contract_address.Bytes())))
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+`name() method doesn't exist`
		} else {
			int_output:=new(uint8)
			err=erc20_token_abi.Unpack(int_output,"decimals",ret)
			if err!=nil {
				tok_info.nc_ERC20=true
				if len(tok_info.e_ERC20)>0 {
					tok_info.e_ERC20=tok_info.e_ERC20+", "
				}
				tok_info.e_ERC20=tok_info.e_ERC20+"decimals not found: "+err.Error()
				log.Info(fmt.Sprintf("Contract %v: can;t upack decimals: %v",hex.EncodeToString(contract_address.Bytes()),err))
			} else {
				tok.decimals=int32(*int_output)
				tok_info.m_ERC20_decimals=true
			}
		}
	}

	if tok_info.m_ERC20_symbol && tok_info.m_ERC20_name && tok_info.m_ERC20_total_supply && tok_info.m_ERC20_balance_of {
		if (len(tok.symbol)>0) && (len(tok.name)>0) {
			tok_info.i_ERC20=true
		}
	}
	if !tok_info.i_ERC20 {
		return false
	}
	balance_positive,initial_transfer_event_found,test_src_addr:=find_account_with_non_zero_balance(chain,block,statedb, event_logs,contract_address,owner_address)
	if !initial_transfer_event_found {
		tok_info.nc_ERC20=true
		if len(tok_info.e_ERC20)>0 {
			tok_info.e_ERC20=tok_info.e_ERC20+", "
		}
		tok_info.e_ERC20=tok_info.e_ERC20+"initial Transfer() event wasn't found in event logs"
	}
	if !balance_positive {
		tok_info.nc_ERC20=true
		if len(tok_info.e_ERC20)>0 {
			tok_info.e_ERC20=tok_info.e_ERC20+", "
		}
		tok_info.e_ERC20=tok_info.e_ERC20+"can't find any account holding tokens"
		if len(tok_info.e_ERC20)>0 {
			tok_info.e_ERC20=tok_info.e_ERC20+", "
		}
		tok_info.e_ERC20=tok_info.e_ERC20+"no accounts to test transfer() functionality"
		if len(tok_info.e_ERC20)>0 {
			tok_info.e_ERC20=tok_info.e_ERC20+", "
		}
		tok_info.e_ERC20=tok_info.e_ERC20+"no accounts to test approve() functionality"
		if len(tok_info.e_ERC20)>0 {
			tok_info.e_ERC20=tok_info.e_ERC20+", "
		}
		tok_info.e_ERC20=tok_info.e_ERC20+"no accounts to test transferFrom() functionality"
		return tok_info.i_ERC20
	}

	// Testing transfer()
	tok_value:=big.NewInt(1)
	test_dst_addr:=common.HexToAddress("8888888888888888888888888888888888888888")
	statedb.AddBalance(*test_src_addr,fake_balance)
	statedb.AddBalance(test_dst_addr,fake_balance)
	input, err = erc20_token_abi.Pack("transfer",test_dst_addr,tok_value)
	if err!=nil {
		utils.Fatalf("packing input for transfer() failed: %v",err)
	}
	msg=types.NewMessage(*test_src_addr,contract_address,0,value,gas_limit,gas_price,input,false)
	ret,gas,_,failed,err=core.ApplyMessage(ethvirmac,msg,gp,unused1,unused2,unused3,unused4,unused5)
	if debug_token_discovery {
		log.Info(fmt.Sprintf("Checking for transfer() ret=%v (len=%v), failed=%v, err=%v",hex.EncodeToString(ret),len(ret),failed,err))
	}
	if ((err!=nil) || (failed)) {
		tok_info.nc_ERC20=true
		if len(tok_info.e_ERC20)>0 {
			tok_info.e_ERC20=tok_info.e_ERC20+", "
		}
		tok_info.e_ERC20=tok_info.e_ERC20+"transfer() method failed"
		if err!=nil {
			tok_info.e_ERC20=tok_info.e_ERC20+":"+err.Error()
		}
	} else {
		output_bool:=new(bool)
		err:=erc20_token_abi.Unpack(&output_bool,"transfer",ret)
		if err!=nil {
			log.Info(fmt.Sprintf("EVM output: Unpack error for transfer(): %v",err))
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+"Cant decode output from transfer():"+err.Error()
		} else {
			if *output_bool==false {
				tok_info.nc_ERC20=true
				if len(tok_info.e_ERC20)>0 {
					tok_info.e_ERC20=tok_info.e_ERC20+", "
				}
				tok_info.e_ERC20=tok_info.e_ERC20+"transfer() method returns false"
			} else {
				test_balance:=get_ERC20_token_balance_from_EVM(chain,statedb, block, contract_address,&test_dst_addr)
				test_failed:=true
				if test_balance!=nil {
					if test_balance.Cmp(tok_value)==0 {
						test_failed=false
					}
				}
				if test_failed {
					tok_info.nc_ERC20=true
					if len(tok_info.e_ERC20)>0 {
						tok_info.e_ERC20=tok_info.e_ERC20+", "
					}
					tok_info.e_ERC20=tok_info.e_ERC20+"transfer() does not work as expected, balance mismatch"
				} else {
					tok_info.m_ERC20_transfer=true
					tok_info.m_ERC20_balance_of=true
				}
			}
		}
	}

	// Testing approve()
	approve_dst_addr:=common.HexToAddress("8888888888888888888888888888888888888887")
	statedb.AddBalance(approve_dst_addr,fake_balance)
	input, err = erc20_token_abi.Pack("approve",approve_dst_addr,tok_value)
	if err!=nil {
		utils.Fatalf("packing input for approve() failed: %v",err)
	}
	msg=types.NewMessage(test_dst_addr,contract_address,0,value,gas_limit,gas_price,input,false)
	ret,gas,_,failed,err=core.ApplyMessage(ethvirmac,msg,gp,unused1,unused2,unused3,unused4,unused5)
	if ((err!=nil) || (failed)) {
		tok_info.nc_ERC20=true
		if len(tok_info.e_ERC20)>0 {
			tok_info.e_ERC20=tok_info.e_ERC20+", "
		}
		tok_info.e_ERC20=tok_info.e_ERC20+"approve() method failed"
		if err!=nil {
			tok_info.e_ERC20=tok_info.e_ERC20+":"+err.Error()
		}
	} else {
		output_bool:=new(bool)
		err:=erc20_token_abi.Unpack(&output_bool,"approve",ret)
		if err!=nil {
			log.Info(fmt.Sprintf("EVM output: Unpack error for approve(): %v",err))
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+"Cant decode output from approve():"+err.Error()
		} else {
			if *output_bool==false {
				tok_info.nc_ERC20=true
				if len(tok_info.e_ERC20)>0 {
					tok_info.e_ERC20=tok_info.e_ERC20+", "
				}
				tok_info.e_ERC20=tok_info.e_ERC20+"approve() method returns false"
			} else {
				tok_info.m_ERC20_approve=true
			}
		}
	}

	// Testing allowance()
	input, err = erc20_token_abi.Pack("allowance",test_dst_addr,approve_dst_addr)
	if err!=nil {
		utils.Fatalf("packing input for allowance() failed: %v",err)
	}
	msg=types.NewMessage(approve_dst_addr,contract_address,0,value,gas_limit,gas_price,input,false)
	ret,gas,_,failed,err=core.ApplyMessage(ethvirmac,msg,gp,unused1,unused2,unused3,unused4,unused5)
	if ((err!=nil) || (failed)) {
		tok_info.nc_ERC20=true
		if len(tok_info.e_ERC20)>0 {
			tok_info.e_ERC20=tok_info.e_ERC20+", "
		}
		tok_info.e_ERC20=tok_info.e_ERC20+"allowance() method failed"
		if err!=nil {
			tok_info.e_ERC20=tok_info.e_ERC20+":"+err.Error()
		}
	} else {
		output_int:=big.NewInt(0)
		err:=erc20_token_abi.Unpack(&output_int,"allowance",ret)
		if err!=nil {
			log.Info(fmt.Sprintf("EVM output: Unpack error for allowance(): %v",err))
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+"Cant decode output from allowance():"+err.Error()
		} else {
			if tok_value.Cmp(output_int)!=0 {
				tok_info.nc_ERC20=true
				if len(tok_info.e_ERC20)>0 {
					tok_info.e_ERC20=tok_info.e_ERC20+", "
				}
				tok_info.e_ERC20=tok_info.e_ERC20+"allowance() method ditn't return correct value"
			} else {
				tok_info.m_ERC20_allowance=true
			}
		}
	}

	// Testing transferFrom()
	input, err = erc20_token_abi.Pack("transferFrom",test_dst_addr,approve_dst_addr,tok_value)
	if err!=nil {
		utils.Fatalf("packing input for transferFrom() failed: %v",err)
	}
	msg=types.NewMessage(approve_dst_addr,contract_address,0,value,gas_limit,gas_price,input,false)
	ret,gas,_,failed,err=core.ApplyMessage(ethvirmac,msg,gp,unused1,unused2,unused3,unused4,unused5)
	if ((err!=nil) || (failed)) {
		tok_info.nc_ERC20=true
		if len(tok_info.e_ERC20)>0 {
			tok_info.e_ERC20=tok_info.e_ERC20+", "
		}
		tok_info.e_ERC20=tok_info.e_ERC20+"transferFrom() method failed"
		if err!=nil {
			tok_info.e_ERC20=tok_info.e_ERC20+":"+err.Error()
		}
	} else {
		output_bool:=new(bool)
		err:=erc20_token_abi.Unpack(&output_bool,"transferFrom",ret)
		if err!=nil {
			log.Info(fmt.Sprintf("EVM output: Unpack error for transferFrom(): %v",err))
			tok_info.nc_ERC20=true
			if len(tok_info.e_ERC20)>0 {
				tok_info.e_ERC20=tok_info.e_ERC20+", "
			}
			tok_info.e_ERC20=tok_info.e_ERC20+"Cant decode output from transferFrom():"+err.Error()
		} else {
			if *output_bool==false {
				tok_info.nc_ERC20=true
				if len(tok_info.e_ERC20)>0 {
					tok_info.e_ERC20=tok_info.e_ERC20+", "
				}
				tok_info.e_ERC20=tok_info.e_ERC20+"transferFrom() method returns false"
			} else {
				test_balance:=get_ERC20_token_balance_from_EVM(chain,statedb, block ,contract_address,&approve_dst_addr)
				test_failed:=true
				if test_balance!=nil {
					if test_balance.Cmp(tok_value)==0 {
						test_failed=false
					}
				}
				if test_failed {
					tok_info.nc_ERC20=true
					if len(tok_info.e_ERC20)>0 {
						tok_info.e_ERC20=tok_info.e_ERC20+", "
					}
					tok_info.e_ERC20=tok_info.e_ERC20+"transferFrom() does not work as expected, balance mismatch"
				} else {
					tok_info.m_ERC20_transfer_from=true
				}
			}
		}
	}

	if !tok_info.m_ERC20_transfer {
		tok_info.i_ERC20=false			// this clears the flag if it was previously set
	}
	//////////////  Test for ERC721 support
	// ToDo

	_=gas
	return tok_info.i_ERC20
}
func (bot *EthBot_t) find_contract_creation_receipt(bchain *core.BlockChain,contract_address common.Address) *types.Receipt {

	block_num:=lookup_account_block_created(hex.EncodeToString(contract_address.Bytes()))
	if (block_num==-1) {
		log.Error("EthBot: can't find block number of contract creation")
		return nil
	}
	block:=bchain.GetBlockByNumber(uint64(block_num))
	if block==nil {
		log.Error("EthBot: can't get block object of contract creation")
		return nil
	}
	receipts:=rawdb.ReadReceipts(bot.ethereum.ChainDb(), block.Hash(), block.NumberU64())
	for _,receipt:=range receipts {
		if receipt.ContractAddress==contract_address {
			return receipt
		}
	}
	log.Error("Receipt for the creation of contract %v wasn't found",hex.EncodeToString(contract_address.Bytes()))
	return nil
}
