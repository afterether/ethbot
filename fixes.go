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
import(
	"fmt"
	"math/big"
	"encoding/hex"
    "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
    "github.com/ethereum/go-ethereum/cmd/utils"
)
func (bot *EthBot_t) fix_last_balances() bool {
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
	return sql_fix_last_balances(statedb);
}
func sql_update_deleted_attribute4all_accounts(block_num Block_num_t) {
	var query string

	blockchain:=ethbot_instance.ethereum.BlockChain()
	block:=blockchain.GetBlockByNumber(uint64(block_num))
	if block==nil {
		utils.Fatalf("nil block at mark_asll_accounts_as_delted()")
	}

	statedb,err:=blockchain.StateAt(block.Root())
	if err!=nil {
		utils.Fatalf("nil statedb at sql_mark_all_accounts_as_deleted() ",err)
	}
	query="SELECT count(*) FROM account"
	row := db.QueryRow(query)
	var num_rows int;
	err=row.Scan(&num_rows);
	if (err!=nil) {
		utils.Fatalf("failed to execute query %v at sql_verify_sql_accounts_against_blockchain()",query)
	}
	ethbot_instance.verification.Num_accounts=num_rows	// this is not a precise number, since the WHERE in SQL query will lower the amount of rows, the lower the block_num is , it is only precise when block_num=last_block_num
	query=`
		SELECT 
			account_id,
			address,
			deleted
		FROM account`
	rows,err:=db.Query(query);
	defer rows.Close()
	if (err!=nil) {
		utils.Fatalf("failed to execute query: %v, error=%v",query,err)
	}
	counter:=int64(0)
	does_not_exist:=big.NewInt(-1)
	for rows.Next() {
		var (
			account_id		Account_id_t
			address			string
			deleted			int16
		)
		err:=rows.Scan(&account_id,&address,&deleted)
		if (err!=nil) {
			utils.Fatalf("failed to Scan() in sql_verify_sql_accounts_against_blockchain(): %v",err);
		}
		if account_id<2 {
			continue				// internal account, skip it
		}
		addr:=common.HexToAddress(address)
		balance:=statedb.GetBalanceIfExists(addr)
		var exists bool
		if balance.Cmp(does_not_exist)==0 {
			exists=false
		} else {
			exists=true
		}
		if !exists {
			if deleted==0 {
				counter++
				query=`UPDATE account SET deleted=1 WHERE account_id=$1`
				_,err:=db.Exec(query,account_id);
				if (err!=nil) {
					utils.Fatalf(fmt.Sprintf("Update of %v to set deleted flag failed: %v",address,err));
				}
			}
		}
		ethbot_instance.verification.Num_processed++
	}
	log.Info(fmt.Sprintf("EthBot: fixed 'deleted' attribute for %v accounts",counter))
}
func sql_fix_vt_balances4account(account_id Account_id_t) bool { // this proc verifies that the balances are correct in SQL db and if not, fixes then by sending UPDATEs. Also updates last_balance in `account`s table, Returns `false` if the account had incorrect balances, `true` if everything is correct

	history:=sql_get_account_history(account_id,-1)
	end:=len(history)
	if end==0 {
		return true
	}
	if end==1 {
		return true
	}
	var i int=1
	var prev_i int=0
	var counter int = 0
	var errors_found bool = false
	var first_error_block_num Block_num_t = 0
	for i<end {
		prev_entry:=history[prev_i]
		entry:=history[i]
		db_stored_cur_balance:=get_entry_balance(entry,account_id)
		accumulated_balance:=get_entry_accumulated_balance(prev_entry,entry,account_id)
		if db_stored_cur_balance.Cmp(accumulated_balance)!=0 {
			// fix errors
			fix_balance_in_vt(account_id,entry,accumulated_balance)
			if !errors_found {
				errors_found=true
				first_error_block_num=entry.block_num
			}
			counter++
			set_correct_entry_balance(entry,account_id,accumulated_balance)
		}
		i++
		prev_i++
	}
	if errors_found {
		log.Info(fmt.Sprintf("Found balance mismatch in account_id=%v at block %v, fixed %v records",account_id,first_error_block_num,counter))
	}
	return true
}
func fix_balance_in_vt(account_id Account_id_t,entry *Acct_history_t,correct_balance *big.Int) {
	var query string
	if (entry.from_id==entry.to_id) { // self transfer
		query="UPDATE value_transfer SET from_balance=$1,to_balance=$1 WHERE valtr_id=$2"
	} else {
		if (entry.from_id==account_id) {
			query="UPDATE value_transfer SET from_balance=$1 WHERE valtr_id=$2"
		}
		if (entry.to_id==account_id) {
			query="UPDATE value_transfer SET to_balance=$1 WHERE valtr_id=$2"
		}
	}
	res,err:=db.Exec(query,correct_balance.String(),entry.valtr_id);
	if (err!=nil) {
		utils.Fatalf("Update value_transfer to correct balance failedfailed: %v",err);
	}
	rows_affected,err:=res.RowsAffected()
	if err==nil {
		if rows_affected==0 {
			log.Info(fmt.Sprintf("Rows for valtr_id=%v affected is zero, bal=%v ",entry.valtr_id,correct_balance.String()))
		}
	} else {
		log.Info("","error",err)
	}
}
func sql_fix_last_balances(statedb *state.StateDB) bool {
	var query string;
	iterator:=statedb.GetNewIterator()
	var addr common.Address
	dump_balance:=big.NewInt(0)
	for statedb.GetNextAccount(iterator,&addr,dump_balance) {
		address_str:=hex.EncodeToString(addr.Bytes())
		account_id:=lookup_account(&addr)
		if (account_id!=0) {
			query=`UPDATE account SET last_balance=$2 WHERE account_id=$1`
			_,err:=db.Exec(query,account_id,dump_balance.String());
			if (err!=nil) {
				utils.Fatalf("Update from_account failed: %v",err);
			}
		} else {
			log.Error(fmt.Sprintf("EthBot: account %v not found in SQL database",address_str))
		}
	}
	return true;
}
