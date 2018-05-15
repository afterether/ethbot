package main
import (
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
)
func set_verif_error(valtr_id int64,err_text string) {
	if ethbot_instance.verification.Failed {
		return;	// only the first error is reported
	}
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
		accounts:=statedb.EthBotDump()
		num_accounts:=len(accounts)
		ethbot_instance.verification.Num_accounts=len(accounts)
		ethbot_instance.verification.Num_processed=0;
		icount:=1;
		for key,dump_balance:=range accounts {
			addr:=key
			address_str:=hex.EncodeToString(addr.Bytes())
			icount++

			account_id=lookup_account(&addr);
			if (account_id==0) {
				set_verif_error(0,fmt.Sprintf("account %s exists in the LevelDB but doesn't exist in SQL. Reported at block_num=%d",address_str,ethbot_instance.verification.Current_block_num))
				ethbot_instance.verification.Num_processed++
				continue;
			}
			if len(account_ids)>0 {
				account_ids=account_ids+","	// comma separated for the IN() function of PG SQL query
			}
			account_id_string:=strconv.FormatInt(int64(account_id),10);
			account_ids=account_ids+account_id_string
			balance:=big.NewInt(0);
			balance.Set(dump_balance)
			balances_map[account_id]=balance
			accounts_counter++
			ethbot_instance.verification.Num_processed++

			if (accounts_counter>=VERIFICATION_LOT_SIZE) || (icount==num_accounts) {
				var block_num Block_num_t=Block_num_t(ethbot_instance.verification.Current_block_num)
				go sql_verify_account_lot(status_write_back,block_num,account_ids,balances_map); // we are launching parallel SQL queries to postgres here 
				ethbot_instance.verification.Threads_counter++;
				if (ethbot_instance.verification.Threads_counter>=VERIFICATION_NUM_THREADS) {// when we reach maximum number of threads, we start listening for them to finish the job
					wait_for_verification_threads(status_write_back)
				}
				accounts_counter=0
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
	for i:=ethbot_instance.verification.Starting_block;i<end;i++ {
		block_num=Block_num_t(i)
		bptr=blockchain.GetBlockByNumber(uint64(i))
		statedb, err := blockchain.StateAt(bptr.Root() )
		if (err!=nil) {
			utils.Fatalf("Can't get state at block %v error=%v",i,err);
		}
		dump:=statedb.RawDump();
		accounts:=dump.Accounts
		ethbot_instance.verification.Num_accounts=len(accounts)
		sql_verify_sql_accounts_against_blockchain(block_num,accounts);
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

	if (entry.from_id==entry.to_id) { // self transfer
		retval:=big.NewInt(0)
		retval.SetString(entry.to_balance,10)
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
func balances_match(chain *core.BlockChain,addr *common.Address,account_id Account_id_t,entry *Acct_history_t ,accum_balance *big.Int) bool {
	last_balance:=get_entry_balance(entry,account_id)
	if accum_balance.Cmp(last_balance)!=0 {
		set_verif_error(entry.valtr_id,fmt.Sprintf("Rebuilding balances from SQL data failed, accumulated balance does not match: (SQL accum in memory) %v != %v (SQL accum on disk). block_num=%v, account_id=%v",accum_balance.String(),last_balance.String(),entry.block_num,account_id))
		return false;
	}
	if addr==nil {
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
func verify_account(account_id Account_id_t,addr *common.Address,block_num Block_num_t) bool {
	chain:=ethbot_instance.ethereum.BlockChain()

	history:=sql_get_account_history(account_id,block_num)
	end:=len(history)
	if end==0 {
		return true
	}
	if end==1 {
		entry:=history[0]
		return verify_balance(chain,addr,account_id,entry)
	}
	var i int=1
	var prev_i=0
	prev_balance:=get_entry_balance(history[prev_i],account_id)
	accum_value:=big.NewInt(0)
	for i<end {
		prev_entry:=history[prev_i]
		entry:=history[i]
		if (prev_entry.block_num!=entry.block_num) {
			accum_balance:=big.NewInt(0)
			accum_balance.Add(prev_balance,accum_value)
			if balances_match(chain,addr,account_id,prev_entry,accum_balance) {
				// continue with the loop
			} else {
				return false
			}
			accum_value=big.NewInt(0)
			prev_balance=get_entry_balance(prev_entry,account_id)
		} else {
			// the account had multiple value transfers in this block, wait until we get to the last one, but in the mean time, allow them to accumulate in `accum_value` variable
			// verify that current balance in SQL matches the balance in accum_value
			stored_balance:=get_entry_balance(prev_entry,account_id)
			accumulated_balance:=big.NewInt(0)
			accumulated_balance.Add(prev_balance,accum_value)
			if stored_balance.Cmp(accumulated_balance)!=0 {
				set_verif_error(entry.valtr_id,fmt.Sprintf("account (account_id=%v) balance accumulated per block doesnt match at valtr_id=%v, (in memory correct balance) %v != %v (wrong balance stored in SQL)",account_id,entry.valtr_id,accumulated_balance,stored_balance))
				return false
			}
		}
		value:=big.NewInt(0)
		if (entry.from_id!=entry.to_id) { // not a selftransfer
			value.SetString(entry.value,10)
			if (account_id==entry.from_id) { // if it is a withdrawal, negate the `value`
					value.Neg(value)
			}
		}
		accum_value.Add(accum_value,value)

		if (i+1)==end { // last entry of account history
			accum_balance:=big.NewInt(0)
			accum_balance.Add(prev_balance,accum_value)
			return balances_match(chain,addr,account_id,entry,accum_balance);
		}
		i++
		prev_i++
	}
	set_verif_error(0,fmt.Sprintf("Error in verification algorithm"))
	return false	// this is never executed
}
func verify_single_account(addr_str string,block_num Block_num_t) bool {
	addr:=common.HexToAddress(addr_str)
	var addr_ptr *common.Address
	account_id,_:=lookup_account_SQL(addr_str)
	if account_id==0 {
		log.Error("EthBot: account address not found, usage: verifyAccount(account_address)")
		return false
	}
	if (account_id<0) { // internal EthBot account , `nil` is sent for verification in this case
		addr_ptr=nil
	} else {
		addr_ptr=&addr
	}
	ethbot_instance.verification.In_progress=true
	retval:=verify_account(Account_id_t(account_id),addr_ptr,block_num)
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
func verify_all_accounts(block_num Block_num_t) bool {
	chain:=ethbot_instance.ethereum.BlockChain()
	block:=chain.GetBlockByNumber(uint64(block_num))
	if (block==nil) {
		log.Error(fmt.Sprintf("EthBot: block number %v was not found in the blockchain",int(block_num)))
		return false
	}
	statedb, err := chain.StateAt(block.Root())
    if err != nil {
		log.Error(fmt.Sprintf("EthBot: StateAt(%v) failed",block.Number().Uint64()))
		return false
    }
	init_verification_status(0,0,block_num)
	ethbot_instance.verification.In_progress=true
	accounts:=statedb.EthBotDump()
	ethbot_instance.verification.Num_accounts=len(accounts)
	for addr,_:=range accounts {
		account_id:=lookup_account(&addr)
		if (account_id>0) {
			result:=verify_account(account_id,&addr,Block_num_t(block_num))
			if !result {
				return false
			}
		}
		ethbot_instance.verification.Num_processed++
	}
	result:=verify_single_account("0",block_num) // this is the NONEXISTENT account
	ethbot_instance.verification.In_progress=false
	return result
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
