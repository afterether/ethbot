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
	"time"
	"net"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
    "github.com/ethereum/go-ethereum/cmd/utils"
	"database/sql"
	_ "github.com/lib/pq"
)
func report_time(start_ts int64,desc string) {
	var end_ts int64
	end_ts=time.Now().UnixNano() / int64(time.Millisecond)
	log.Info(fmt.Sprintf("%v time: %v ms",desc,(end_ts-start_ts)))
}
func calc_time(start_ts int64) int64 {
	var end_ts int64
	end_ts=time.Now().UnixNano() / int64(time.Millisecond)
	result:=(end_ts-start_ts)
	return result
}
func init_postgres() {
	var err error
	log.Info(fmt.Sprintf("EthBot: connecting to PostgreSQL database: %v@%v/%v",os.Getenv("ETHBOT_USERNAME"),os.Getenv("ETHBOT_HOST"),os.Getenv("ETHBOT_DATABASE")))
	host,port,err:=net.SplitHostPort(os.Getenv("ETHBOT_HOST"))
	if (err!=nil) {
		host=os.Getenv("ETHBOT_HOST")
		port="5432"
	}
	conn_str:="user='"+os.Getenv("ETHBOT_USERNAME")+"' dbname='"+os.Getenv("ETHBOT_DATABASE")+"' password='"+os.Getenv("ETHBOT_PASSWORD")+"' host='"+host+"' port='"+port+"'";
	db,err=sql.Open("postgres",conn_str);
	if (err!=nil) {
		log.Error("EthBot: can't connect to PostgreSQL database. Check that you have set ETHBOT_USERNAME,ETHBOT_PASSWORD,ETHBOT_DATABASE and ETHBOT_HOST environment variables");
	} else {
	}
	db.SetMaxOpenConns(VERIFICATION_NUM_THREADS+128)
	db.SetMaxIdleConns(VERIFICATION_NUM_THREADS+128)
	db.SetConnMaxLifetime(0)
	row := db.QueryRow("SELECT now()")
	var now string
	err=row.Scan(&now);
	if (err!=nil) {
		log.Error("EthBot: can't connect to PostgreSQL database. Check that you have set ETHBOT_USERNAME,ETHBOT_PASSWORD,ETHBOT_DATABASE and ETHBOT_HOST environment variables");
		utils.Fatalf("error: %v",err);
	} else {
		log.Info("EthBot: connected to Postgres successfuly");
	}

	block_num:=sql_get_last_block_num();
	if (block_num==-2) {
		utils.Fatalf("can't get block_num from `last_block` table")
	} else {
		log.Info("EthBot: last exported block is: ","block_num",block_num)
	}
	token_block_num:=sql_get_last_token_block_num();
	if (token_block_num==-2) {
		utils.Fatalf("can't get block_num from `last_block` table")
	} else {
		log.Info("EthBot: last TOKEN export done at","block_num",token_block_num)
	}
}
func lookup_block_by_hash(hash string) (Block_num_t,bool) {
	var query string
	query="SELECT block_num FROM block WHERE block_hash=$1"
	var block_num Block_num_t=-1
	err:=db.QueryRow(query,hash).Scan(&block_num);
	if (err!=nil) {
		if (err==sql.ErrNoRows) {
			return -1,false;
		} else {
			utils.Fatalf("Error looking up block by hash: %v",err);
		}
	}
	return block_num,true
}
func lookup_account(addr *common.Address) Account_id_t {
	// locking code is only effective when verification process is being run, in parallel to the export process
	if (addr==nil) {
		utils.Fatalf("lookup account with null address")
	}
	addr_str:=hex.EncodeToString(addr.Bytes())
	ac_lock.RLock()		// locking is required only because verifySQLdata1() process uses accounts_cache
	account_id,exists:=accounts_cache[*addr]
	ac_lock.RUnlock()
	if exists {
		return account_id
	} else {
		account_id,_,_:=lookup_account_SQL(addr_str)
		if (account_id!=0) {
			// the cache is wirtten only by lookup_account() or account_update_deleted_attribute() functions
			ac_lock.Lock()
			accounts_cache[*addr]=account_id
			ac_lock.Unlock()
		}
		return account_id
	}
}
func lookup_account_SQL(addr_str string) (account_id Account_id_t,owner_id Account_id_t,deleted int16) {
	query:="SELECT account_id,owner_id,deleted FROM account WHERE address=$1"
	var start_ts int64=0
	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}
	row:=db.QueryRow(query,addr_str);
	err:=row.Scan(&account_id,&owner_id,&deleted);
	if debug_sql_exec_time {
		report_time(start_ts,"lookup_account()")
	}
	if err!=nil {
		if (err==sql.ErrNoRows) {
			return 0,0,0
		} else {
		}
		utils.Fatalf("Can't execute lookup account query: %v",err)
	}
	return account_id,owner_id,deleted
}
func lookup_account_data(addr_str string) (account_id Account_id_t,owner_id Account_id_t,balance string) {
	query:="SELECT account_id,owner_id,last_balance FROM account WHERE address=$1"
	var start_ts int64=0
	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}
	row:=db.QueryRow(query,addr_str);
	err:=row.Scan(&account_id,&owner_id,&balance);
	if debug_sql_exec_time {
		report_time(start_ts,"lookup_account_data()")
	}
	if (err==sql.ErrNoRows) {
		return 0,0,balance
	} else {
		return account_id,owner_id,balance
	}
}
func lookup_account_by_id(account_id Account_id_t) (string,bool) {
	query:="SELECT address FROM account WHERE account_id=$1"
	row:=db.QueryRow(query,account_id);
	var address string
	err:=row.Scan(&address);
	if (err==sql.ErrNoRows) {
		return "",false
	}
	return address,true
}
func account_update_deleted_attribute(addr *common.Address,deleted int16,block_num Block_num_t) {

	if addr==nil {
		utils.Fatalf("Ethbot: account_update_deleted_attribute() received nil address")
	}
	var query string
	query="UPDATE account SET deleted=$1,block_del=$3 WHERE address=$2"
	block_del:=Block_num_t(0)
	if deleted==1 {
		block_del=block_num
	}
	addr_str:=hex.EncodeToString(addr.Bytes())
	_,err:=db.Exec(query,deleted,addr_str,block_del)
	if err!=nil {
		utils.Fatalf("Ethbot: error updating account status deleted=1 : %v: addr=%v",err,hex.EncodeToString(addr.Bytes()))
	}
}
func lookup_block_by_num(block_num Block_num_t) Block_num_t {
	var query string
	query="SELECT block_num FROM block WHERE block_num=$1"

	var bnum Block_num_t
	err:=db.QueryRow(query,block_num).Scan(&bnum);
	if (err!=nil) {
		if (err==sql.ErrNoRows) {
			return -1;
		} else {
			utils.Fatalf("Error looking up block by num: %v",err);
		}
	}

	return bnum
}
func lookup_transaction_by_hash(hash string) int64 {
	var query string
	var tx_id int64;
	query="SELECT tx_id FROM transaction WHERE tx_hash=$1"
	row:=db.QueryRow(query,hash)
	err:=row.Scan(&tx_id)
	if err!=nil {
		log.Info(fmt.Sprintf("transaction with hash %v not found",hash))
		return -1
	}
	return tx_id
}
func lookup_txinfo_by_hash(hash string) (int64,Block_num_t) {
	var query string
	var tx_id int64;
	var block_num Block_num_t
	query="SELECT tx_id,block_num FROM transaction WHERE tx_hash=$1"
	row:=db.QueryRow(query,hash)
	err:=row.Scan(&tx_id,&block_num)
	if err!=nil {
		log.Info(fmt.Sprintf("transaction with hash %v not found",hash))
		return -1,-1
	}
	return tx_id,block_num
}
func lookup_account_block_created(address string) Block_num_t {
	var query string
	query=`SELECT block_created FROM account WHERE address=$1`
	row:=db.QueryRow(query,address)
	var block_num Block_num_t
	err:=row.Scan(&block_num)
	if err!=nil {
		if err==sql.ErrNoRows {
			return -1
		}
		utils.Fatalf("lookup account created block failed: %v",err)
	}
	return block_num
}
func sql_query_get_balance() string { // version 2, designed after hitting 100 million records in value_transfers tablrae
	return `
	SELECT valtr_id,block_num,from_id,to_id,from_balance,to_balance FROM
	(
		    (
				SELECT valtr_id,block_num,bnumvt,from_id,to_id,from_balance,to_balance
				FROM value_transfer v
				WHERE (v.bnumvt<$1) AND (v.from_id = $2)
			) UNION ALL (
				SELECT valtr_id,block_num,bnumvt,from_id,to_id,from_balance,to_balance
				FROM value_transfer v
				WHERE (v.bnumvt<$1) AND (v.to_id = $2)
			)
	) AS subtable
	ORDER BY bnumvt DESC
	LIMIT 1

	`
}
func sql_query_get_account_history() string { // version 2, designed after hitting 100 million records in value_transfers tablrae
	return `
	SELECT block_num,valtr_id,from_id,to_id,from_balance,to_balance,value,kind FROM
	(
		    (
				SELECT block_num,bnumvt,valtr_id,from_id,to_id,from_balance,to_balance,value,kind
				FROM value_transfer v
				WHERE (v.bnumvt<$1) AND (v.from_id = $2)
				ORDER BY bnumvt
			) UNION ALL (
				SELECT block_num,bnumvt,valtr_id,from_id,to_id,from_balance,to_balance,value,kind
				FROM value_transfer v
				WHERE (v.bnumvt<$1) AND (v.to_id = $2)
				ORDER BY bnumvt
			)
	) AS subtable
	ORDER BY bnumvt

	`
}
func sql_query_get_account_history_full() string { // version 2, designed after hitting 100 million records in value_transfers table
	return `
	SELECT block_num,valtr_id,from_id,to_id,from_balance,to_balance,value,kind FROM
	(
		    (
				SELECT block_num,valtr_id,from_id,to_id,from_balance,to_balance,value,kind
				FROM value_transfer v
				WHERE v.from_id = $1
				ORDER BY bnumvt
			) UNION ALL (
				SELECT block_num,valtr_id,from_id,to_id,from_balance,to_balance,value,kind
				FROM value_transfer v
				WHERE v.to_id = $1
				ORDER BY bnumvt
			)
	) AS subtable
	ORDER BY bnumvt

	`
}
func sql_query_get_balance_non_existent_account() string { // version 2, designed after hitting 100 million records in value_transfers tablrae
	return `
	SELECT valtr_id,block_num,from_id,to_id,from_balance,to_balance
	FROM value_transfer v
	WHERE (v.bnumvt<$1) AND (v.from_id = $2)
	ORDER BY bnumvt DESC
	LIMIT 1

	`
}
