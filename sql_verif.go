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
	"encoding/hex"
	"strconv"
	"math/big"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
    "github.com/ethereum/go-ethereum/cmd/utils"
    "github.com/ethereum/go-ethereum/core/state"
	"database/sql"
	_ "github.com/lib/pq"
)
func sql_verify_sql_accounts_against_blockchain(block_num Block_num_t,statedb *state.StateDB) {
	var query string

	query="SELECT count(*) FROM account"
	row := db.QueryRow(query)
	var num_rows int;
	err:=row.Scan(&num_rows);
	if (err!=nil) {
		utils.Fatalf("failed to execute query %v at sql_verify_sql_accounts_against_blockchain()",query)
	}
	ethbot_instance.verification.Num_accounts=num_rows	// this is not a precise number, since the WHERE in SQL query will lower the amount of rows, the lower the block_num is , it is only precise when block_num=last_block_num

	query=`
		SELECT 
			account_id,
			address,
			deleted
		FROM account
		WHERE block_created<=$1
		OFFSET $1
		LIMIT 500000
`
	var offset int64 =0
	for true {
		rows,err:=db.Query(query,block_num);
		defer rows.Close()
		if (err!=nil) {
			utils.Fatalf("failed to execute query: %v, block_num=%v, error=%v",query,block_num,err)
		}
		does_not_exist:=big.NewInt(-1)
		for rows.Next() {
			var (
				account_id		Account_id_t
				address			string
				deleted			int16
			)
			err:=rows.Scan(&account_id,&address,&deleted)
			if (err!=nil) {
				if err==sql.ErrNoRows {
					return
				}
				utils.Fatalf("failed to Scan() in sql_verify_sql_accounts_against_blockchain(): %v",err);
			}
			if account_id<2 {
				ethbot_instance.verification.Num_processed++
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
					set_verif_error(0,fmt.Sprintf("account %v does not exist LevelDB but it does exist in SQL",address))
					return
				}
			} else {
				if deleted==1 {
					// We ignore this error because verifySQLdata1() already does catch this data mismatch, 
					// Catching it here is difficult and will work until a resurrected SELFEDSTRUCTED account is found
					// It is difficult because we don't have a history of when the account was SELFDESTRUCTed and resurrected  in StateDB 
				}
			}
			ethbot_instance.verification.Num_processed++
		} // end of scan rows
		offset=offset+500000
	} // lot loop
}
func sql_verify_account_lot(status_write_back chan bool, block_num Block_num_t,account_ids string,blockchain_balances map[Account_id_t]*big.Int ) {
	var query string;

	query=`SELECT * FROM get_many_balances(`+strconv.Itoa(int(block_num))+`,'`+account_ids+`')`;
	rows,err:=db.Query(query);
	defer rows.Close()
	if (err!=nil) {
		utils.Fatalf("failed to execute query: %v, block_num=%v, error=%v",query,block_num,err)
	}
	for rows.Next() {
		// the following query gets last balance for the account at that time in the past
		var (
			account_id						Account_id_t
			valtr_id						int64
			last_block_num					Block_num_t
			from_id,to_id					Account_id_t
			sql_account_from_balance_str	string
			sql_account_to_balance_str		string
			sql_account_from_balance		big.Int
			sql_account_to_balance			big.Int
		)

		err:=rows.Scan(&account_id,&valtr_id,&last_block_num,&from_id,&to_id,&sql_account_from_balance_str,&sql_account_to_balance_str);
		if err!=nil {
			if err==sql.ErrNoRows {
				balance,_:=blockchain_balances[account_id]
				valtr_id=-1
				last_block_num=block_num
				from_id=account_id
				to_id=account_id
				sql_account_from_balance_str="0"
				sql_account_to_balance_str="0"
				addr_str,_:=lookup_account_by_id(account_id)
				set_verif_error(valtr_id,fmt.Sprintf("SQL DB does not have records for (id=%v) %v, last block_num<=%v with expected balance=%v, balance in SQL DB=NULL",account_id,addr_str,last_block_num,balance.String()))
				break;
			} else {
				utils.Fatalf("failed to execute query: %v, error=%v",query,err)
			}
		}
		balance,exists:=blockchain_balances[account_id]
		if !exists {
			utils.Fatalf(fmt.Sprintf("balance for account_id=%v does not exist in array of balances",account_id))
		}
		sql_account_from_balance.SetString(sql_account_from_balance_str,10);
		sql_account_to_balance.SetString(sql_account_to_balance_str,10);

		if from_id==to_id {	// self transfer
			if sql_account_to_balance.Cmp(balance)!=0 {
				addr_str,_:=lookup_account_by_id(account_id)
				set_verif_error(valtr_id,fmt.Sprintf("(selftransfer) (id:%v) %v with balance=%v in statedb does not match the balance in SQL DB=%v (last block in SQL = %v)",account_id,addr_str,balance.String(),sql_account_to_balance.String(),last_block_num))
				break;
			}
		} else {
			if (to_id==account_id) {
				if sql_account_to_balance.Cmp(balance)!=0 {
					addr_str,_:=lookup_account_by_id(account_id)
					set_verif_error(valtr_id,fmt.Sprintf("(to) (id=%v) %v, with balance=%v in statedb does not match the balance in SQL DB=%v (last block in SQL = %v)",account_id,addr_str,balance.String(),sql_account_to_balance.String(),last_block_num))
					break;
				}
			}
			if (from_id==account_id) {
				if sql_account_from_balance.Cmp(balance)!=0 {
					addr_str,_:=lookup_account_by_id(account_id)
					set_verif_error(valtr_id,fmt.Sprintf("(from) (id=%v) %v with balance=%v in statedb does not match the balance in SQL DB=%v (last block in SQL = %v)",account_id,addr_str,balance.String(),sql_account_from_balance.String(),last_block_num))
					break;
				}
			}
		}
	}
	status_write_back <- true
}
func sql_get_account_history(account_id Account_id_t, block_num Block_num_t) []*Acct_history_t {

	var query string

	if block_num==-1 {
		query=sql_query_get_account_history_full()
	} else {
		query=sql_query_get_account_history()
	}

	var history []*Acct_history_t =make([]*Acct_history_t,0)
	var rows *sql.Rows
	var err error
	var bnumvt int64
	bnumvt=(int64(block_num+1)<<18)

	if block_num==-1 {
		rows,err=db.Query(query,int(account_id));
	} else {
		rows,err=db.Query(query,bnumvt,int(account_id));
	}
	defer rows.Close()
	if (err!=nil) {
		utils.Fatalf("EthBot: failed to execute query: %v, account_id=%v, block_num=%v, error=%v",query,account_id,block_num,err)
	}
	for rows.Next() {
		h:=&Acct_history_t {}
		var (
			block_num		int
			valtr_id		int64
			from_id			int
			to_id			int
		)

		err:=rows.Scan(&block_num,&valtr_id,&from_id,&to_id,&h.from_balance,&h.to_balance,&h.value,&h.kind)
		if (err!=nil) {
			if err==sql.ErrNoRows {
				return history;
			} else {
				log.Error(fmt.Sprintf("EthBot: scan failed at sql_get_account_history(): %v",err))
				os.Exit(2)
			}
		}
		h.block_num=Block_num_t(block_num)
		h.valtr_id=valtr_id
		h.from_id=Account_id_t(from_id)
		h.to_id=Account_id_t(to_id)
		history=append(history,h)
	}
	return history
}
func sql_verify_currency_sum(block_num Block_num_t) {
	var query string

	query="SELECT sum(last_balance)::text from account"
	row := db.QueryRow(query)
	var sum_str string="0"
	var sum_null_str sql.NullString
	err:=row.Scan(&sum_null_str);
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error in sum(last_balance) at Scan() : %v",err))
		utils.Fatalf("error: %v",err);
	}
	if sum_null_str.Valid {
		sum_str=sum_null_str.String
		sum:=big.NewInt(0)
		sum.SetString(sum_str,10)
		zero:=big.NewInt(0)
		if sum.Cmp(zero)!=0 {
			log.Error(fmt.Sprintf("EthBot: sum(last_balance) is not zero. block_num=%v",block_num))
		}
	}
}
func sql_verify_last_balances(statedb *state.StateDB) bool {
	var query string;
	var balance_str string
	var retval bool=true
	dump_balance:=big.NewInt(0)
	balance:=big.NewInt(0)
	var addr common.Address
	iterator:=statedb.GetNewIterator()
	for statedb.GetNextAccount(iterator,&addr,dump_balance) {
		address_str:=hex.EncodeToString(addr.Bytes())
		account_id:=lookup_account(&addr)
		if (account_id!=0) {
			query=`SELECT last_balance FROM account WHERE account_id=$1`
			row:=db.QueryRow(query,account_id);
			err:=row.Scan(&balance_str)
			if (err!=nil) {
				utils.Fatalf("Select last_balance failed: %v",err);
			}
			balance.SetString(balance_str,10)
			if dump_balance.Cmp(balance)!=0 {
				log.Error(fmt.Sprintf("EthBot: balance for account %v does not match (Blockchain %v!=%v SQL)",address_str,dump_balance.String(),balance_str))
				retval=false;
			}
		} else {
			log.Error(fmt.Sprintf("EthBot: account %v not found in SQL database",address_str))
		}
	}
	return retval;
}
