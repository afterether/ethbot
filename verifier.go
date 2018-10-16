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
	"database/sql"
	"github.com/robertkrimen/otto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/common"
	"encoding/hex"
	"math/big"
	"strconv"
	"fmt"
	"errors"
	"time"
)
func set_verif_error(valtr_id int64,err_text string) {
	ethbot_instance.verification.Failed=true
	ethbot_instance.verification.Failing_valtr_id=valtr_id;
	ethbot_instance.verification.Error_str=err_text
	log.Error("Verification failed","valtr_id",ethbot_instance.verification.Failing_valtr_id,"error",ethbot_instance.verification.Error_str);
}
func verif_check_input_block_num(arg_block_num otto.Value) (Block_num_t,error) {
	p_block_num,err:=arg_block_num.ToInteger()
	if (err!=nil) {
		err_text:="Invalid input value for `block_num` parameter: positive integer values are allowed"
		log.Error(err_text)
		return 0,errors.New(err_text)
	}
	return Block_num_t(p_block_num),nil
}
func check_input_block_range(arg_starting_block otto.Value,arg_ending_block otto.Value) (Block_num_t,Block_num_t,error) {

	p_starting_block,err:=arg_starting_block.ToInteger()
	if (err!=nil) {
		err_text:="Invalid input value for `starting_block` parameter: positive integer values or -1 are allowed"
		log.Error(err_text)
		return 0,0,errors.New(err_text)
	}
	if (p_starting_block < -1) {
		err_text:="Invalid input value for `starting_block` parameter: positive integer values or -1 are allowed"
		log.Error(err_text)
		return 0,0,errors.New(err_text)
	}
	p_ending_block,err:=arg_ending_block.ToInteger()
	if (err!=nil) {
		err_text:="Invalid input value for `ending_block` parameter: positive integer values or -1 are allowed"
		log.Error(err_text)
		return 0,0,errors.New(err_text)
	}
	if (p_ending_block < -1) {
		err_text:="Invalid input value for `ending_block` parameter: positive integer values or -1 are allowed"
		log.Error(err_text)
		return 0,0,errors.New(err_text)
	}
	return Block_num_t(p_starting_block),Block_num_t(p_ending_block),nil
}
func init_verification_status(mode int,starting_block Block_num_t,ending_block Block_num_t) {
	ethbot_instance.verification.User_cancelled=false;
	ethbot_instance.verification.Mode=int(mode);
	ethbot_instance.verification.Failed=false;
	ethbot_instance.verification.Finished_threads_counter=0;
	ethbot_instance.verification.Threads_counter=0;
	ethbot_instance.verification.In_progress=true;
	ethbot_instance.verification.Error_str="";
	ethbot_instance.verification.Num_accounts=0;
	ethbot_instance.verification.Num_processed=0;
	ethbot_instance.verification.Current_block_num=Block_num_t(starting_block);
	ethbot_instance.verification.Starting_block=Block_num_t(starting_block);
	ethbot_instance.verification.Ending_block=Block_num_t(ending_block);
}
func verify_SQL_data(mode int, starting_block Block_num_t,ending_block Block_num_t) bool {
	// stage of variable initialization, per verification process
	if ethbot_instance.verification.In_progress {
		set_verif_error(0,"Verification process is already running at this `geth` instance, start another process for parallel validations");
		return false
	}
	init_verification_status(mode,starting_block,ending_block)
	if (ethbot_instance.verification.Starting_block<ethbot_instance.verification.Ending_block) {
		ethbot_instance.verification.Direction=1;
	} else {
		if (ethbot_instance.verification.Starting_block>ethbot_instance.verification.Ending_block) {
			ethbot_instance.verification.Direction=-1
		} else {
			ethbot_instance.verification.Direction=0;
		}
	}
	// end of: stage of variable initialization, per verification process
	// pick the correct function, depending on the mode
	switch(mode) {
		case VERIFICATION_LEVELDB: {
			verify_SQL_data_1();
		}
		case VERIFICATION_SQL: {
			verify_SQL_data_2();
		}
		default: {	// this should never occur since we validate above
			set_verif_error(0,fmt.Sprintf("Unknown validation mode %v",mode));
			ethbot_instance.verification.In_progress=false;
			return false
		}
	}
	ethbot_instance.verification.In_progress=false;
	if (ethbot_instance.verification.Failed) {
		return false
	} else {
		return true
	}
}
func stop_verification() {
	ethbot_instance.verification.User_cancelled=true;
}
func stop_verification_completed() {
	log.Info("EthBot: sent cancellation request. Will stop when the loop is completed.");
}
func verification_status(vs *Verification_t) *otto.Object {
	jsre:=console_obj.JSRE()
	vm:=jsre.VM()
	obj_str:=fmt.Sprintf(`({"in_progress":%v,"block_num":%d,"failed":%v,"valtr_id":%d,"cancelled_by_user":%v,"total_threads":%d,"finished_threads":%d,"num_accounts":%v,"num_processed":%v})`,vs.In_progress,vs.Current_block_num,vs.Failed,vs.Failing_valtr_id,vs.User_cancelled,vs.Threads_counter,vs.Finished_threads_counter,vs.Num_accounts,vs.Num_processed);
	object, err := vm.Object(obj_str)
	if (err!=nil) {
		utils.Fatalf("Failed to create object in Javascript VM for verification status object: %v",err)
	}
	object.Set("error_str",vs.Error_str); // in separate Set() to avoid escaping javascript strings in the previous call to VM few lines above
	return object
}
func wait_for_verification_threads(ch chan bool) {
	var counter int=0
	if (ethbot_instance.verification.Threads_counter==0) {
		return
	}
	for true {
		verif_thread_finished:=<-ch
		_=verif_thread_finished
		counter++
		if (counter>=ethbot_instance.verification.Threads_counter) {
			break;
		}
	}
	ethbot_instance.verification.Threads_counter=0;
}
func verify_SQL_data_1() {
	var i int=0
	var status_write_back chan bool // used to keep track of how many verification go_routines has been spawn
	var bptr *types.Block;

	status_write_back=make(chan bool)
	blockchain:=ethbot_instance.ethereum.BlockChain()
	end:=ethbot_instance.verification.Ending_block;

	for ethbot_instance.verification.Current_block_num=ethbot_instance.verification.Starting_block;ethbot_instance.verification.Current_block_num<=end;ethbot_instance.verification.Current_block_num++ {
		bptr=blockchain.GetBlockByNumber(uint64(ethbot_instance.verification.Current_block_num))
		if (bptr==nil) {
			log.Error(fmt.Sprintf("EthBot: invalid block number specified by user: %v",ethbot_instance.verification.Current_block_num))
			set_verif_error(0,fmt.Sprintf("EthBot: invalid block number specified by user: %v",ethbot_instance.verification.Current_block_num))
			return;
		}
		statedb, err := blockchain.StateAt(bptr.Root())
        if err != nil {
			log.Error(fmt.Sprintf("EthBot: stateAt(%v) failed",bptr.Number().Uint64()))
			set_verif_error(0,fmt.Sprintf("EthBot: StateAt(%v) failed",bptr.Number().Uint64()));
			return
        }
		var accounts_counter int=0;
		var balances_map map[Account_id_t]*big.Int
		balances_map=make(map[Account_id_t]*big.Int)
		var account_ids string;
		var account_id Account_id_t;
		ethbot_instance.verification.Num_accounts=-1
		ethbot_instance.verification.Num_processed=0;
		icount:=1;
		iterator:=statedb.GetNewIterator()
		var cur_account_addr common.Address
		var cur_account_balance=big.NewInt(-1)
		for statedb.GetNextAccount(iterator,&cur_account_addr,cur_account_balance) {
			address_str:=hex.EncodeToString(cur_account_addr.Bytes())
			icount++
			account_id=lookup_account(&cur_account_addr);
			if (account_id==0) {
				set_verif_error(0,fmt.Sprintf("account %s exists in LevelDB but doesn't exist in SQL. Reported at block_num=%d",address_str,ethbot_instance.verification.Current_block_num))
				ethbot_instance.verification.Num_processed++
				continue;
			}
			if len(account_ids)>0 {
				account_ids=account_ids+","	// comma separated for the IN() function of PG SQL query
			}
			account_id_string:=strconv.FormatInt(int64(account_id),10);
			account_ids=account_ids+account_id_string
			balance:=big.NewInt(0);
			balance.Set(cur_account_balance)
			balances_map[account_id]=balance
			accounts_counter++
			ethbot_instance.verification.Num_processed++

			if accounts_counter>=VERIFICATION_LOT_SIZE  {
				var block_num Block_num_t=Block_num_t(ethbot_instance.verification.Current_block_num)
				go sql_verify_account_lot(status_write_back,block_num,account_ids,balances_map); // we are launching parallel SQL queries to postgres here 
				ethbot_instance.verification.Threads_counter++;
				if (ethbot_instance.verification.Threads_counter>=VERIFICATION_NUM_THREADS) {// when we reach maximum number of threads, we start listening for them to finish the job
					wait_for_verification_threads(status_write_back)
				}
				accounts_counter=0
				account_ids=""
				balances_map=make(map[Account_id_t]*big.Int)
			}
			if (ethbot_instance.verification.Failed) {
				break;
			}
			if (ethbot_instance.verification.User_cancelled) {
				ethbot_instance.verification.User_cancelled=false;
				break;
			}
			i++
		} // end for (accounts)
		wait_for_verification_threads(status_write_back)// wait for threads, if there are any pending
		if (ethbot_instance.verification.Failed) {
			break;
		}
		if (ethbot_instance.verification.User_cancelled) {
			ethbot_instance.verification.User_cancelled=false;
			break;
		}
	}
	return;
}
func verify_SQL_data_2() {
	var bptr *types.Block
	var block_num Block_num_t
	blockchain:=ethbot_instance.ethereum.BlockChain()
	end:=ethbot_instance.verification.Ending_block;
	for i:=ethbot_instance.verification.Starting_block;i<=end;i++ {
		block_num=Block_num_t(i)
		bptr=blockchain.GetBlockByNumber(uint64(i))
		statedb, err := blockchain.StateAt(bptr.Root() )
		if (err!=nil) {
			utils.Fatalf("Can't get state at block %v error=%v",i,err);
		}

		sql_verify_sql_accounts_against_blockchain(block_num,statedb);
		if (ethbot_instance.verification.Failed) {
			return;
		}
	}
	return;
}
func verify_balance(chain *core.BlockChain,addr *common.Address,account_id Account_id_t,entry *Acct_history_t) bool {
	if (addr==nil) {
		return true;	// the account does not exist in the blockchain (EthBot internal account) so we return true by default
	}
	block:=chain.GetBlockByNumber(uint64(entry.block_num))
	if (block==nil) {
		log.Error(fmt.Sprintf("EthBot: can't find block %v in the blockchain",entry.block_num))
		return false
	}
	statedb, err := chain.StateAt(block.Root())
	if err != nil {
		log.Error(fmt.Sprintf("EthBot: StateAt(%v) failed",block.Number().Uint64()))
		return false
	}
	balance_on_chain:=statedb.GetBalance(*addr)
	sql_balance:=get_entry_balance(entry,account_id);
	if balance_on_chain.Cmp(sql_balance)==0 {
		return true
	}
	set_verif_error(entry.valtr_id,fmt.Sprintf("Balances does not match (blockchain) %v != %v (SQL). account_id=%v, block_num=%v",balance_on_chain.String(),sql_balance.String(),account_id,entry.block_num))
	return false
}
func verify_balance2(vparam *Verif_acct_param_t,entry *Acct_history_t) bool {
	if (vparam.account_id<0) {
		return true;	// the account does not exist in the blockchain (EthBot internal account) so we return true by default
	}
	sql_balance:=get_entry_balance(entry,vparam.account_id);
	if vparam.balance.Cmp(sql_balance)==0 {
		return true
	} else {
		set_verif_error(entry.valtr_id,fmt.Sprintf("Balances does not match (blockchain) %v != %v (SQL). account_id=%v, block_num=%v",vparam.balance.String(),sql_balance.String(),vparam.account_id,entry.block_num))
		return false
	}
}
func get_balance_from_state(chain *core.BlockChain,addr common.Address,p_block_num Block_num_t) (*big.Int,bool) {
	block_num:=uint64(p_block_num)
	block:=chain.GetBlockByNumber(block_num)
	if (block==nil) {
		log.Error(fmt.Sprintf("EthBot: can't find block %v in the blockchain",block_num))
		return nil,false
	}
	statedb, err := chain.StateAt(block.Root())
	if err != nil {
		log.Error(fmt.Sprintf("EthBot: StateAt(%v) failed",block.Number().Uint64()))
		return nil,false
	}
	balance_on_chain:=statedb.GetBalance(addr)
	return balance_on_chain,true
}
func get_entry_balance(entry *Acct_history_t ,account_id Account_id_t) *big.Int {

	if (entry.from_id==entry.to_id) { // self transfer. in a self transfer transaction the `To` balance is more valid than `'From` balance
		retval:=big.NewInt(0)
		retval.SetString(entry.to_balance,10)
		check_balance:=big.NewInt(0)
		check_balance.SetString(entry.from_balance,10)
		if entry.kind!=7 {	// suicide balances never match 'from' and to 'balances' (in this case for to_addr=from_addr) because 'to' balance will be 0 always (statedb.Suicide() sets it to 0)
			if check_balance.Cmp(retval)!=0 {
				retval.SetString("-1",10)			// mark balance as invalid
			}
		}
		return retval
	} else {
		if (entry.from_id==account_id) {
			retval:=big.NewInt(0)
			retval.SetString(entry.from_balance,10)
			return retval
		}
		if (entry.to_id==account_id) {
			retval:=big.NewInt(0);
			retval.SetString(entry.to_balance,10)
			return retval
		}
	}
	return big.NewInt(0)
}
func get_entry_accumulated_balance(prev_entry,entry *Acct_history_t ,account_id Account_id_t) *big.Int {

	prev_balance:=get_entry_balance(prev_entry,account_id)
	if (entry.from_id==entry.to_id) { // self transfer
		retval:=big.NewInt(0)
		if entry.kind==VALTRANSF_CONTRACT_SELFDESTRUCT {
			// on SELFDESTRUCT the balance will be 0 always (if from=to)
		} else {
			retval.Set(prev_balance)
		}
		return retval
	} else {
		if (entry.from_id==account_id) {
			retval:=big.NewInt(0)
			val:=big.NewInt(0)
			val.SetString(entry.value,10)
			retval.Sub(prev_balance,val)
			return retval
		}
		if (entry.to_id==account_id) {
			retval:=big.NewInt(0);
			val:=big.NewInt(0)
			val.SetString(entry.value,10)
			retval.Add(prev_balance,val)
			return retval
		}
	}
	return big.NewInt(0)
}
func set_correct_entry_balance(entry *Acct_history_t ,account_id Account_id_t,correct_balance *big.Int) {

	if (entry.from_id==entry.to_id) { // self transfer
		entry.to_balance=correct_balance.String()
		entry.from_balance=correct_balance.String()
	} else {
		if (entry.from_id==account_id) {
			entry.from_balance=correct_balance.String()
		}
		if (entry.to_id==account_id) {
			entry.to_balance=correct_balance.String()
		}
	}
}
func balances_match(chain *core.BlockChain,addr *common.Address,account_id Account_id_t,entry *Acct_history_t ,accum_balance *big.Int) bool {
	last_balance:=get_entry_balance(entry,account_id)
	diff:=big.NewInt(0)
	diff.Sub(last_balance,accum_balance)
	if accum_balance.Cmp(last_balance)!=0 {
		set_verif_error(entry.valtr_id,fmt.Sprintf("Accumulated balance does not match: (SQL accum in memory) %v != %v (SQL accum on disk). block_num=%v, account_id=%v",accum_balance.String(),last_balance.String(),entry.block_num,account_id))
		return false;
	}
	if addr==nil {	// internal EthBot account (possibly BLOCKHAIN account with account_id=-1)
		return true // thee is no State entry in the State that is accumulating all Ether created so far
	}
	blockchain_balance,done:=get_balance_from_state(chain,*addr,entry.block_num)
	if (done) {
		if accum_balance.Cmp(blockchain_balance)==0 {
			return true
		} else {
			set_verif_error(entry.valtr_id,fmt.Sprintf("Balance on SQL doest match the blockchain (SQL) %v != %v (Blockchain) , block_num=%v, account_id=%v",accum_balance,blockchain_balance,entry.block_num,account_id))
			return false
		}
	} else {
		log.Error("EthBot: failed to get balance from State")
		set_verif_error(entry.valtr_id,fmt.Sprintf("Failed to get state , block_num=%v, account_id=%v",entry.block_num,account_id))
		return false
	}
}
func balances_match2(vparam *Verif_acct_param_t,entry *Acct_history_t ,accum_balance *big.Int) bool {
	last_balance:=get_entry_balance(entry,vparam.account_id)
	diff:=big.NewInt(0)
	diff.Sub(last_balance,accum_balance)
	if accum_balance.Cmp(last_balance)!=0 {
		set_verif_error(entry.valtr_id,fmt.Sprintf("Accumulated balance does not match: (SQL accum in memory) %v != %v (SQL accum on disk). block_num=%v, account_id=%v",accum_balance.String(),last_balance.String(),entry.block_num,vparam.account_id))
		return false;
	}
	if vparam.account_id<0 {	// internal EthBot account (possibly BLOCKHAIN account with account_id=-1)
		return true // thee is no State entry in the State that is accumulating all Ether created so far
	}
	if accum_balance.Cmp(&vparam.balance)==0 {
		return true
	} else {
		set_verif_error(entry.valtr_id,fmt.Sprintf("Balance on SQL doest match the blockchain (SQL) %v != %v (Blockchain) , block_num=%v, account_id=%v",accum_balance,vparam.balance.String(),entry.block_num,vparam.account_id))
		return false
	}
}
func verify_account(vparam *Verif_acct_param_t,block_num Block_num_t) bool {
	history:=sql_get_account_history(vparam.account_id,block_num)
	end:=len(history)
	if end==0 {
		return true
	}
	if end==1 {
		entry:=history[0]
		return verify_balance2(vparam,entry)
	}
	var i int=1
	var prev_i int=0
	var errors_found bool = false
	accumulated_balance:=big.NewInt(0)
	db_stored_cur_balance:=big.NewInt(0)
	entry:=history[i]
	for i<end {
		prev_entry:=history[prev_i]
		entry=history[i]
		db_stored_cur_balance=get_entry_balance(entry,vparam.account_id)
		accumulated_balance=get_entry_accumulated_balance(prev_entry,entry,vparam.account_id)
		diff:=big.NewInt(0)
		diff.Sub(accumulated_balance,db_stored_cur_balance)
		if db_stored_cur_balance.Cmp(accumulated_balance)!=0 {
			log.Info(fmt.Sprintf("Value %v\tStored: \t%v\tAccumulated: \t%v\tDiff: %v",entry.value,db_stored_cur_balance.String(),accumulated_balance.String(),diff.String()))
			errors_found=true
			break
		}
		i++
		prev_i++
	}
	if errors_found {
		set_verif_error(entry.valtr_id,fmt.Sprintf("account (account_id=%v %v) balance accumulated per block doesnt match at valtr_id=%v, (in memory correct balance) %v != %v (wrong balance stored in SQL), block_num=%v",vparam.account_id,hex.EncodeToString(vparam.addr.Bytes()),entry.valtr_id,accumulated_balance.String(),db_stored_cur_balance.String(),entry.block_num))
		return false
	}
	match:=balances_match2(vparam,entry,accumulated_balance);
	return match
}
func verify_single_account(addr_str string,block_num Block_num_t) bool {
	vparam:=&Verif_acct_param_t{}
	a:=common.HexToAddress(addr_str)
	vparam.addr.SetBytes(a.Bytes())
	vparam.account_id,_,_=lookup_account_SQL(addr_str)
	if vparam.account_id==0 {
		log.Error("EthBot: account address not found, usage: verifyAccount(account_address)")
		ethbot_instance.verification.In_progress=false
		return false
	}
	chain:=ethbot_instance.ethereum.BlockChain()
	block:=chain.GetBlockByNumber(uint64(block_num))
	if (block==nil) {
		log.Error(fmt.Sprintf("EthBot: block number %v was not found in the blockchain",int(block_num)))
		ethbot_instance.verification.In_progress=false
		return false
	}
	statedb, err := chain.StateAt(block.Root())
    if err != nil {
		log.Error(fmt.Sprintf("EthBot: StateAt(%v) failed",block.Number().Uint64()))
		ethbot_instance.verification.In_progress=false
		return false
    }
	vparam.balance.Set(statedb.GetBalance(vparam.addr))
	ethbot_instance.verification.In_progress=true
	retval:=verify_account(vparam,block_num)
	ethbot_instance.verification.In_progress=false
	if retval {
		return true
	} else {
		return false
	}
}
func check_input_verify_account(arg_account_address otto.Value,arg_block_num otto.Value) (string,Block_num_t,error) {
	var err error
	account_addr_str,err:=arg_account_address.ToString()
	if (err!=nil) {
		err_text:="Invalid input value for `account_address` parameter: does not look like a string"
		log.Error(err_text)
		return account_addr_str,0,errors.New(err_text)
	}
	if len(account_addr_str)!=(common.AddressLength*2) {
		if account_addr_str!="0" {
			err_text:="Invalid input value for `account_address` parameter: 40 character HEX string is required, without 0x prepended"
			log.Error(err_text)
			return account_addr_str,0,errors.New(err_text)
		} else {
			// single zero is a special case for NONEXISTENT account
		}
	}
	block_num,err:=arg_block_num.ToInteger()
	if (err!=nil) {
		err_text:="Invalid input value for `block_num` parameter: positive integer values or -1 are allowed"
		log.Error(err_text)
		return account_addr_str,0,errors.New(err_text)
	}
	if (block_num < -1) {
		err_text:="Invalid input value for `block_num` parameter: positive integer or -1 values are allowed"
		log.Error(err_text)
		return account_addr_str,0,errors.New(err_text)
	}
	return account_addr_str,Block_num_t(block_num),nil
}
func check_input_verify_all_accounts(arg_block_num otto.Value) (Block_num_t,error) {

	block_num,err:=arg_block_num.ToInteger()
	if (err!=nil) {
		err_text:="Invalid input value for `block_num` parameter: positive integer values or -1 are allowed"
		log.Error(err_text)
		return 0,errors.New(err_text)
	}
	if (block_num < -1) {
		err_text:="Invalid input value for `block_num` parameter: positive integer values or -1 are allowed"
		log.Error(err_text)
		return 0,errors.New(err_text)
	}
	return Block_num_t(block_num),nil
}
func adjust_block_num(block_num Block_num_t) Block_num_t {
	chain:=ethbot_instance.ethereum.BlockChain()
	if (block_num<0) {
		block:=chain.CurrentBlock()
		block_num=Block_num_t(block.NumberU64())
	}
	return block_num
}
func verify_account_thread(c chan *Verif_acct_param_t,block_num Block_num_t) {
	var vparam *Verif_acct_param_t
	for true {
		vparam = <-c
		if vparam==nil {	// exit singnal
			break
		}
		result:=verify_account(vparam,block_num)
		if !result {
			ethbot_instance.verification.In_progress=false
		}
		ethbot_instance.verification.Num_processed++
	}
	wg.Done()
}
func verify_all_accounts(block_num Block_num_t) bool {
	start_ts:=time.Now().UnixNano() / int64(time.Millisecond)
	chain:=ethbot_instance.ethereum.BlockChain()
	block:=chain.GetBlockByNumber(uint64(block_num))
	if (block==nil) {
		log.Error(fmt.Sprintf("EthBot: block number %v was not found in the blockchain",int(block_num)))
		ethbot_instance.verification.In_progress=false
		return false
	}
	statedb, err := chain.StateAt(block.Root())
    if err != nil {
		log.Error(fmt.Sprintf("EthBot: StateAt(%v) failed",block.Number().Uint64()))
		ethbot_instance.verification.In_progress=false
		return false
    }
	init_verification_status(0,block_num,block_num)
	ethbot_instance.verification.In_progress=true
	iterator:=statedb.GetNewIterator()
	ethbot_instance.verification.Num_accounts=0
	var chan_array []chan *Verif_acct_param_t
	chan_array=make([]chan *Verif_acct_param_t,VERIFICATION_LOT_SIZE,VERIFICATION_LOT_SIZE)
	for i:=0;i<VERIFICATION_LOT_SIZE;i++ {
		wg.Add(1)
		chan_array[i]=make(chan *Verif_acct_param_t,16)
		go verify_account_thread(chan_array[i],block_num)
	}
	var tid int=0
	var counter int64 = 0
	vparam_ptr:=&Verif_acct_param_t{}
	for statedb.GetNextAccount(iterator,&vparam_ptr.addr,&vparam_ptr.balance) {
		ethbot_instance.verification.Num_accounts++
		vparam_ptr.account_id=lookup_account(&vparam_ptr.addr)
		if (vparam_ptr.account_id>0) {
			tid=int(counter % int64(VERIFICATION_LOT_SIZE))
			counter++
			ch:=chan_array[tid]
			ch <- vparam_ptr
		}
		if	ethbot_instance.verification.In_progress==false {
			break;
		}
		vparam_ptr=&Verif_acct_param_t{}
	}
	for i:=0;i<VERIFICATION_LOT_SIZE;i++ {		// send exit signal to go routines
		ch:=chan_array[i]
		ch <- nil
	}
	wg.Wait()
	if ethbot_instance.verification.Failed && (!ethbot_instance.verification.In_progress) {
		return false
	}
	result:=verify_single_account("BLOCKCHAIN",block_num)
	if !result {
		ethbot_instance.verification.In_progress=false
		return false
	}
	result=verify_single_account("REFUNDS",block_num)
	ethbot_instance.verification.In_progress=false
	process_time:=calc_time(start_ts)
	per_account_time:=process_time/int64(ethbot_instance.verification.Num_processed)

	log.Info(fmt.Sprintf("EthBot: verification time: %v ms, per account: %v ms",process_time,per_account_time))
	return result
}
func verify_single_token_account(block_num Block_num_t, contract_id,account_id Account_id_t,contract_address,account_address string,amount string) bool {

	blockchain := ethbot_instance.ethereum.BlockChain()
	block:=blockchain.GetBlockByNumber(uint64(block_num))
    statedb, err := blockchain.StateAt(block.Root())
    if err != nil {
		log.Error("EthBot: can't get StateAt()","block_num",block.NumberU64())
        return false
    }

	balance,_:=sql_get_token_previous_balance(contract_id,block_num,account_id)
	amount_big:=big.NewInt(0)
	amount_big.SetString(amount,10)
	if balance.Cmp(amount_big)!=0 {
		set_verif_error(0,fmt.Sprintf("Balance in `tokop` (%v) doesn't match balance in `tokholder` (%v)",balance.String(),amount))
		//return false
	}
	// now verify what does the StateDB has
	contract_addr:=common.HexToAddress(contract_address)
	acct_addr:=common.HexToAddress(account_address)
	vm_balance:=get_ERC20_token_balance_from_EVM(blockchain,statedb,block,&contract_addr,&acct_addr);
	if vm_balance==nil {
		set_verif_error(0,fmt.Sprintf("Error occurred when getting token balance from VM for account %v",account_address))
		//return false
	}

	if vm_balance.Cmp(balance)!=0 {
		set_verif_error(0,fmt.Sprintf("Token balance for account %v (id=%v) (contract_id=%v) in EVM does not match SQL (SQL %v!=%v EVM)",account_address,account_id,contract_id,balance.String(),vm_balance.String()))
	} else {
		log.Info(fmt.Sprintf("Account %v match!",account_address))
	}
	return true
}
func verify_single_token(contract_address string,contract_id Account_id_t,block_num Block_num_t) bool {
	var query string

	query=`SELECT sum(amount) AS sum FROM ft_hold WHERE contract_id=$1`
	row:=db.QueryRow(query,contract_id)
	var sum sql.NullFloat64
	err:=row.Scan(&sum)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Error at Scan() for query %v: %v",query,err))
	}
	if sum.Valid {
		if sum.Float64!=0 {
			set_verif_error(0,fmt.Sprintf("The holdings for contract (%v) don't sum up to 0, probably some transfers are missing. Missing holdings sum up the amount of %v",contract_address,sum.Float64))
			return false
		}
	}

	query=`SELECT DISTINCT tokacct_id,a.address FROM (
			(SELECT DISTINCT from_id AS tokacct_id FROM tokop WHERE contract_id=$1) 
			UNION ALL 
			(SELECT DISTINCT to_id AS tokacct_id FROM tokop WHERE contract_id=$1)
		) AS o,tokacct AS a 
		WHERE o.tokacct_id=a.account_id`
	subrows,err:=db.Query(query,contract_id)
	defer subrows.Close()
	if err!=nil {
		utils.Fatalf("Querying %v received error %v",query,err)
	}
	for subrows.Next() {
		var tokacct_id Account_id_t
		var account_address string
		suberr:=subrows.Scan(&tokacct_id,&account_address)
		if suberr!=nil {
			utils.Fatalf("Error scanning token account record: %v",suberr)
		}
		if tokacct_id==NONEXISTENT_ADDRESS_ACCOUNT_ID { // token emission happened, probably during contract creation, we do not verify this because the VM doesn't have a 0x0 account
			continue
		}
		amount,balance_block_num:=sql_get_token_previous_balance(contract_id,block_num,tokacct_id)
		if (balance_block_num>block_num) || (balance_block_num==-1) {	// prevents verification of accounts created in future blocks, case that occurrs during verification of past blocks
			continue
		}
		amount_str:=amount.String()
		if !verify_single_token_account(block_num,contract_id,tokacct_id,contract_address,account_address,amount_str) {
			ethbot_instance.verification.In_progress=false
			return false
		}
	}
	return true
}
func verify_token(contract_address string,block_num Block_num_t) bool {
	contract_id,_,_:=lookup_account_SQL(contract_address)
	if contract_id<1 {
		log.Error(fmt.Sprintf("VerifyToken: account %v does not exist in SQL",contract_address))
		return false
	}
	return verify_single_token(contract_address,contract_id,block_num)
}
func verify_all_tokens(block_num Block_num_t) bool {

	blockchain := ethbot_instance.ethereum.BlockChain()
	block:=blockchain.GetBlockByNumber(uint64(block_num))
	if block==nil {
		log.Info(fmt.Sprintf("Block %v doesn't exist",block_num))
		return false
	}
    statedb, err := blockchain.StateAt(block.Root())
    if err != nil {
		log.Error("EthBot: can't get StateAt()","block_num",block.NumberU64())
        return false
    }

	init_verification_status(0,block_num,block_num)
	var query string
	query=`SELECT count(*) AS num_tokens FROM token`
	row:=db.QueryRow(query)
	var num_tokens int
	err=row.Scan(&num_tokens)
	if err!=nil {
		utils.Fatalf("Error geting number of tokens: %v",err)
	}
	if num_tokens==0 {
		log.Info("No tokens found")
		return true
	}

	ethbot_instance.verification.Num_accounts=num_tokens
	query=`SELECT t.contract_id,c.address as contract_address,t.total_supply,t.name,t.symbol,ti.i_ERC20,ti.nc_ERC20 FROM token AS t,token_info AS ti,account AS c WHERE t.contract_id=c.account_id and t.contract_id=ti.contract_id ORDER by contract_id`
	rows,err:=db.Query(query)
	defer rows.Close()
	if (err!=nil) {
		utils.Fatalf("failed to execute query: %v, error=%v",query,err)
	}
	minus_one:=big.NewInt(-1)
	var contract_id Account_id_t
	var total_supply_str,name,symbol,contract_address string
	var i_ERC20,nc_ERC20 bool
	for rows.Next() {
		err=rows.Scan(&contract_id,&contract_address,&total_supply_str,&name,&symbol,&i_ERC20,&nc_ERC20)
		log.Info(fmt.Sprintf("processing token %v (%v)",contract_address,symbol))
		if err==nil {
		} else {
			utils.Fatalf(fmt.Sprintf("Error at Scan(): %v",err))
		}
		contract_balance:=statedb.GetBalanceIfExists(common.HexToAddress(contract_address))
		if contract_balance.Cmp(minus_one)==0 {
			continue			// contract wasn't created yet at this block
		}
		if !i_ERC20 {
			continue			// token is not ERC20
		}
		if nc_ERC20 {
			log.Info(fmt.Sprintf("token %v (%v) is not complying with ERC20 standard, skipping verification",contract_address,symbol))
			continue			// token not compliant with the standard, abort verification
		}
		if verify_token(contract_address,block_num) {
			ethbot_instance.verification.Num_processed++
		} else {
			break;
		}
	}
	log.Info(fmt.Sprintf("EthBot: total ERC20-compliant tokens verified in during this process: %v tokens",ethbot_instance.verification.Num_processed))
	ethbot_instance.verification.In_progress=false
	return true
}
func fix_account_vt_balances(account_address string) bool {

	account_id,_,_:=lookup_account_SQL(account_address)
	if account_id==0 {
		log.Error(fmt.Sprintf("account %v not found in SQL DB",account_address))
		return false
	}
	ret:=sql_fix_vt_balances4account(account_id)
	return ret
}
func (bot *EthBot_t) verify_last_balances() bool {
	last_block_num:=sql_get_last_block_num();
	if (last_block_num<0) {
		log.Error("EthBot: last_block is < 0 (invalid value)")
		return false;
	}
	chain:=bot.ethereum.BlockChain()
	block:=chain.GetBlockByNumber(uint64(last_block_num))
	statedb, err := chain.StateAt(block.Root())
    if err != nil {
		log.Error(fmt.Sprintf("EthBot: StateAt(%v) failed",last_block_num))
		return false
	}
	return sql_verify_last_balances(statedb);
}
