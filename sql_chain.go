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
	"time"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"math/big"
	"strings"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
    "github.com/ethereum/go-ethereum/cmd/utils"
	"database/sql"
	_ "github.com/lib/pq"
)
func block2sql(chain *core.BlockChain,block *types.Block,num_tx int) error {
	var total_dif *big.Int
	block_num:=Block_num_t(block.NumberU64())
	if block_num==0 {
		total_dif=block.Header().Difficulty;
	} else {
		total_dif=chain.GetTdByHash(block.Hash());
	}
	sql_delete_block(int(block_num),block.Uncles());	// remove all the previous data for this block

	uncles:=block.Uncles();
	bsize_int64:=int64(block.Size())
	err:=sql_insert_block(block.Header(),total_dif,len(uncles),bsize_int64);
	if (err!=nil) {
		return err;
	}

	// Inserts not only the main block, but uncles too, uncles have the same block_num, but different uncle_pos. main block has uncle_pos=0
	for i,uncle_hdr:=range uncles {
		uncle_pos:=i+1
		err:=sql_insert_uncle(uncle_hdr,total_dif,uncle_pos);
		if (err!=nil) {
			return err;
		}
	}
	return nil;
}
func sql_insert_block(hdr *types.Header,total_dif *big.Int,num_uncles int,size int64) error {
	var err error
	var miner_id Account_id_t
	var query string

	block_hash:=hex.EncodeToString(hdr.Hash().Bytes());
	block_num:=Block_num_t(hdr.Number.Uint64())
	parent_id,block_found:=lookup_block_by_hash(hex.EncodeToString(hdr.ParentHash.Bytes()));
	if !block_found {
		if block_num!=0 {
			return ErrAncestorNotFound
		}
	}
	timestamp:=int(hdr.Time.Int64())
	miner_id=lookup_account(&hdr.Coinbase);
	if (miner_id==0) {
		var err error
		miner_id,err=sql_insert_account(&hdr.Coinbase,0,0,timestamp,block_num)
		if (err!=nil) {
			return err
		}
	}
	total_dif_str:=total_dif.String();
	time_str:=hdr.Time.String();
	extra:=hdr.Extra

	nonce_str:=fmt.Sprintf("%d",hdr.Nonce.Uint64());

	query=`
		INSERT INTO block(
			parent_id,
			block_num,
			block_ts,
			miner_id,
			difficulty,
			total_dif,
			gas_limit,
			gas_used,
			nonce,
			block_hash,
			uncle_hash,
			state_root,
			tx_hash,
			num_uncles,
			size,
			rcpt_hash,
			mix_hash,
			bloom,
			extra
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19) 
		`;
	result,err:=db.Exec(query,
		parent_id,
		block_num,
		time_str,
		miner_id,
		hdr.Difficulty.String(),
		total_dif_str,
		hdr.GasLimit,
		hdr.GasUsed,
		nonce_str,
		block_hash,
		hex.EncodeToString(hdr.UncleHash.Bytes()),
		hex.EncodeToString(hdr.Root.Bytes()),
		hex.EncodeToString(hdr.TxHash.Bytes()),
		num_uncles,
		size,
		hex.EncodeToString(hdr.ReceiptHash.Bytes()),
		hex.EncodeToString(hdr.MixDigest.Bytes()),
		hdr.Bloom.Bytes(),
		extra);
	if (err!=nil) {
		utils.Fatalf("Error inserting into `blocks` table: %v",err);
	}
	rows_affected,err:=result.RowsAffected()
	if err==nil {
		if rows_affected==0 {
			utils.Fatalf(fmt.Sprintf("Couldn't insert block into the database: %v",err))
		}
	} else {
		utils.Fatalf(fmt.Sprintf("Error in block insertion: %v",err))
	}
	return nil;
}
func sql_insert_uncle(hdr *types.Header,total_dif *big.Int,uncle_pos int) error {
	var err error
	var miner_id Account_id_t
	var query string
	var parent_block_num Block_num_t = -1

	block_hash:=hex.EncodeToString(hdr.Hash().Bytes());
	block_num:=Block_num_t(hdr.Number.Uint64())
	tmp_parent_num,block_found:=lookup_block_by_hash(hex.EncodeToString(hdr.ParentHash.Bytes()));
	if !block_found {
		utils.Fatalf("Parent for uncle not found. block=",block_num)
	} else {
		parent_block_num=tmp_parent_num
	}
	timestamp:=int(hdr.Time.Int64())
	miner_id=lookup_account(&hdr.Coinbase)
	if (miner_id==0) {
		miner_id,err=sql_insert_account(&hdr.Coinbase,0,0,timestamp,block_num);
		if (err!=nil) {
			return err
		}
	}
	total_dif_str:=total_dif.String();
	time_str:=hdr.Time.String();
	extra:=hdr.Extra

	nonce_str:=fmt.Sprintf("%d",hdr.Nonce.Uint64());

	query=`
		INSERT INTO uncle(
			block_num,
			parent_num,
			block_ts,
			miner_id,
			uncle_pos,
			difficulty,
			total_dif,
			gas_limit,
			gas_used,
			nonce,
			block_hash,
			uncle_hash,
			state_root,
			tx_hash,
			rcpt_hash,
			mix_hash,
			bloom,
			extra
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18) 
		RETURNING uncle_id`;
	result,err:=db.Exec(query,
		block_num,
		parent_block_num,
		time_str,
		miner_id,
		uncle_pos,
		hdr.Difficulty.String(),
		total_dif_str,
		hdr.GasLimit,
		hdr.GasUsed,
		nonce_str,
		block_hash,
		hex.EncodeToString(hdr.UncleHash.Bytes()),
		hex.EncodeToString(hdr.Root.Bytes()),
		hex.EncodeToString(hdr.TxHash.Bytes()),
		hex.EncodeToString(hdr.ReceiptHash.Bytes()),
		hex.EncodeToString(hdr.MixDigest.Bytes()),
		hdr.Bloom.Bytes(),
		extra);
	if (err!=nil) {
		utils.Fatalf("Error inserting into `uncles` table: %v",err);
	}
	rows_affected,err:=result.RowsAffected()
	if err==nil {
		if rows_affected==0 {
			utils.Fatalf(fmt.Sprintf("Couldn't insert uncle into the database: %v",err))
		}
	} else {
		utils.Fatalf(fmt.Sprintf("Error in uncle insertion: %v",err))
	}
	return nil;
}
func sql_insert_transaction(tx_status int,from *common.Address,tx *types.Transaction,receipt *types.Receipt,tx_err error,vm_err error,block_num Block_num_t,tx_index int,timestamp int,num_VTs int,amount_transferred *big.Int) (int64,error) {
	var from_id Account_id_t
	from_id=lookup_account(from)
	if (from_id==0) {
		var err error
		from_id,err=sql_insert_account(from,0,0,timestamp,block_num)
		if (err!=nil) {
			return 0,err
		}
	}
	to:=tx.To();
	var to_id Account_id_t
	if (to==nil) {
		to_id=CONTRACT_CREATION_ACCOUNT_ID
	} else {
		var err error
		to_id=lookup_account(to)
		if (to_id==0) {
			to_id,err=sql_insert_account(to,0,0,timestamp,block_num)
			if (err!=nil) {
				return 0,err
			}
		}
	}
	var transaction_id int64;
	var query string;
	v,r,s:=tx.RawSignatureValues();
	var payload []byte
	//payload=tx.Data();	// commented because we don't really need this in the SQL database , it is stored in LevelDB anyway, also value_transfer_extras have it per evm.Call()
	tx_err_str:=""
	if (tx_err!=nil) {
		tx_err_str=tx_err.Error()
	}
	vm_err_str:=""
	if (vm_err!=nil) {
		vm_err_str=vm_err.Error()
	}

	query=`
		INSERT INTO transaction(
			bnumtx,
			from_id,
			to_id,
			gas_limit,
			gas_used,
			tx_value,
			gas_price,
			nonce,
			block_num,
			tx_index,
			tx_ts,
			num_vt,
			val_transferred,
			v,
			r,
			s,
			tx_status,
			tx_hash,
			tx_error,
			vm_error,
			payload
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
		RETURNING tx_id`;
	var nonce int=int(tx.Nonce())
	var err error
	gas_used:= uint64(0)
	if (receipt!=nil) {
		gas_used=receipt.GasUsed
	}
	var start_ts int64=0
	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}
	var bnumtx int64
	bnumtx=(int64(block_num)<<12) | int64(tx_index)
	err=db.QueryRow(query,
		bnumtx,
		from_id,
		to_id,
		tx.Gas(),
		gas_used,
		tx.Value().String(),
		tx.GasPrice().String(),
		nonce,
		block_num,
		tx_index,
		timestamp,
		num_VTs,
		amount_transferred.String(),
		v.String(),
		r.String(),
		s.String(),
		tx_status,
		hex.EncodeToString(tx.Hash().Bytes()),
		tx_err_str,
		vm_err_str,
		payload).Scan(&transaction_id);
	if debug_sql_exec_time {
		report_time(start_ts,"INSERT into transaction")
	}
	if (err!=nil) {
		utils.Fatalf("Can't insert transaction. error=%v",err);
	}
	return transaction_id,nil;
}
func get_previous_balance_SQL(block_num Block_num_t,account_id Account_id_t) *big.Int {

	var query string
	var aux_from,aux_to Account_id_t
	var aux_from_bal_str,aux_to_bal_str string
	var aux_valtr_id int64;
	var aux_block_num Block_num_t;

	balance:=big.NewInt(0)

	if (account_id==NONEXISTENT_ADDRESS_ACCOUNT_ID) || (account_id==REFUNDS_ACCOUNT_ID) {	// BLOCKCHAIN account
		query=sql_query_get_balance_non_existent_account()
	} else {
		query=sql_query_get_balance()
	}
	var bnumvt int64
	bnumvt=(int64(block_num)<<18)
	dquery:=strings.Replace(query,`$1`,strconv.Itoa(int(bnumvt)),-1)
	dquery=strings.Replace(dquery,`$2`,strconv.Itoa(int(account_id)),-1)
	var start_ts int64=0
	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}
	row:=db.QueryRow(query,bnumvt,account_id);
	err:=row.Scan(&aux_valtr_id,&aux_block_num,&aux_from,&aux_to,&aux_from_bal_str,&aux_to_bal_str)
	if debug_sql_exec_time {
		report_time(start_ts,fmt.Sprintf("get_previous_balance_SQL(account_id=%v,block_num=%v)",account_id,block_num))
	}
	if (err!=nil) {
		if err==sql.ErrNoRows {
			if (account_id==NONEXISTENT_ADDRESS_ACCOUNT_ID) || (account_id==REFUNDS_ACCOUNT_ID) {	// BLOCKCHAIN account
				return balance
			}
			// nothing, balance is kept at 0
		} else {
			log.Error(fmt.Sprintf("EthBot: getting balance for account_id %v failed",account_id));
			utils.Fatalf("error",err);
		}
	} else { // rows with previous balances were found
		if aux_from==aux_to { // self transfer
			if aux_to==account_id {
				balance.SetString(aux_to_bal_str,10)
			} else {
				utils.Fatalf(fmt.Sprintf("EthBot: unknown use case 1 at get_previous_balance() for account_id=%v, block_num=%v ",account_id,block_num))
			}
		} else {
			if aux_from==account_id {
				balance.SetString(aux_from_bal_str,10)
			} else {
				if aux_to==account_id {
					balance.SetString(aux_to_bal_str,10)
				} else {
					utils.Fatalf(fmt.Sprintf("EthBot: unknown use case 2 at get_previous_balance() for account_id=%v, block_num=%v ",account_id,block_num))
				}
			}
		}
	}
	return balance
}
func sql_insert_value_transfer(transfer *vm.Ethbot_EVM_VT_t, transaction_id int64,tx *types.Transaction,timestamp int,block_num Block_num_t) (int64,Account_id_t,Account_id_t,error) {
	var valtr_id int64
	var query string
	var err error
	var from_id,to_id Account_id_t

	var balance_from *big.Int
	value:=big.NewInt(0)
	value.Set(&transfer.Value)
	err_str:=""
	if (transfer.Err!=nil) {
		err_str=fmt.Sprintf(`{"vm_err":"%v","src":"%v","dst":"%v","value":%v}`,transfer.Err.Error(),hex.EncodeToString(transfer.From.Bytes()),hex.EncodeToString(transfer.To.Bytes()),value.String())
		value.Set(big.NewInt(0))		// value_transfers table only contains valid VTs, so we clear the value to make it valid
	}
	var owner_id Account_id_t = 0
	var deleted int16 = 0
	if transfer.Kind==VALTRANSF_CONTRACT_CREATION {
		owner_id=from_id
		if len(err_str)>0 { // contract creation resulted in error, mark account as deleted
			deleted=1
		}
	}

	if (transfer.Kind==VALTRANSF_CONTRACT_CREATION) && (transfer.Err!=nil) && (transfer.To==(common.Address{})) {	// error in contract creation, no address assigned
		to_id=CONTRACT_CREATION_ACCOUNT_ID
	} else {
		to_id=lookup_account(&transfer.To)
		if to_id==0 {
			if len(err_str)>0 {
				deleted=1
			}
			to_id,err=sql_insert_account(&transfer.To,owner_id,deleted,timestamp,block_num)
			if (err!=nil) {
				utils.Fatalf(fmt.Sprintf("Can't insert account for address %v: %v",hex.EncodeToString(transfer.To.Bytes()),err))
			}
			if deleted==1 {
				account_update_deleted_attribute(&transfer.To,1,block_num)		// this actually doesn't updates deleted attribute but the deleted block_num
			}
		} else {
			if (transfer.Err!=nil)  && (tx!=nil) {
				if (transfer.Kind==VALTRANSF_TRANSACTION) {
					to:=tx.To();
					if (to!=nil) {
						if *to==transfer.To {
							if len(tx.Data())>0 { // special case to catch EthBotError()s from the vm
								acct_block_created:=lookup_account_block_created(hex.EncodeToString(transfer.To.Bytes()))
								if acct_block_created==block_num {	// we have to update delted attribute because (despite the error), the account was inserted during the INSERT in 'transaction` table
									deleted=1
									account_update_deleted_attribute(&transfer.To,1,block_num)
								}
							}
						}
					}
				}
			}
		}
	}
	if (transfer.Kind==VALTRANSF_GENESIS) || (transfer.Kind==VALTRANSF_BLOCK_REWARD) {
		from_id=NONEXISTENT_ADDRESS_ACCOUNT_ID
		balance_from=ethbot_instance.get_cached_balance(block_num,from_id)
		balance_from.Sub(balance_from,value)
	} else {
		from_id=lookup_account(&transfer.From)
		if (from_id==0) {	// account does not exist
			from_id,err=sql_insert_account(&transfer.From,0,0,timestamp,block_num)
			if (err!=nil) {
				utils.Fatalf(fmt.Sprintf("Can't insert account for address %v: %v",hex.EncodeToString(transfer.From.Bytes()),err))
			}
		}
		balance_from=&transfer.From_balance
	}

	value_str:=value.String();
	from_balance_str:=balance_from.String()
	to_balance_str:=transfer.To_balance.String()
	gas_refund_str:=transfer.Gas_refund.String()
	if vt_debug {
		log.Info(fmt.Sprintf("SQL INSERT: from (%v) %v to %v (%v) value %v",from_id,from_balance_str,to_balance_str,to_id,value_str))
	}
	var bnumvt int64
	bnumvt=(int64(block_num)<<18) | int64(per_block_VT_counter)
	per_block_VT_counter++
	query="INSERT INTO value_transfer(tx_id,bnumvt,block_num,from_id,to_id,value,from_balance,to_balance,gas_refund,kind,depth,error) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING valtr_id"
	var row *sql.Row
	if (transaction_id==-1) { // this is the case when a value_transfer record is not linked to a transaction record
		var null_var sql.NullInt64
		transaction_id_param:=null_var
		row=db.QueryRow(query,transaction_id_param,bnumvt,block_num,from_id,to_id,value_str,from_balance_str,to_balance_str,gas_refund_str,transfer.Kind,transfer.Depth,err_str)
	} else {
		transaction_id_param:=transaction_id
		row=db.QueryRow(query,transaction_id_param,bnumvt,block_num,from_id,to_id,value_str,from_balance_str,to_balance_str,gas_refund_str,transfer.Kind,transfer.Depth,err_str)
	}
	err=row.Scan(&valtr_id)
	if (err!=nil) {
		utils.Fatalf("Can't insert value_transfer, error=",err);
	}

	return valtr_id,from_id,to_id,nil;
}
func sql_insert_value_transfer_extras(evm_vt *vm.Ethbot_EVM_VT_t, valtr_id int64,to_id Account_id_t,tx_id int64) error {
	var query string

	// inserting input/output
	query=`INSERT INTO vt_extras(valtr_id,input,output) VALUES($1,$2,$3)`
	res,err:=db.Exec(query,valtr_id,evm_vt.Input,evm_vt.Output)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Can't insert into 'vt_extras': %v",err))
	}
	rows_affected,err:=res.RowsAffected()
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Error getting rowsAffected after insert into vt_extras: %v",err))
	}
	if rows_affected==0 {
		utils.Fatalf(fmt.Sprintf("rowsAffected after insert into vt_extras is 0, valtr_id=%v",valtr_id))
	}

	// inserting events
	query=`INSERT INTO event(valtr_id,tx_id,block_num,contract_id,log_index,topics,data) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING event_id`
	for _,event_log:=range evm_vt.Logs {
		contract_id:=to_id
		if to_id==0 {
			contract_id=lookup_account(&event_log.Address)
			if contract_id==0 {
				utils.Fatalf(fmt.Sprintf("insert of event failed, account %v wasn't found",hex.EncodeToString(event_log.Address.Bytes())))
			}
		}
		topics,err:=json.Marshal(event_log.Topics)
		if err!=nil {
			utils.Fatalf("json encoding of event topics failed %v",topics)
		}
		row:=db.QueryRow(query,valtr_id,tx_id,Block_num_t(event_log.BlockNumber),contract_id,event_log.Index,topics,event_log.Data)
		var event_id int64
		err=row.Scan(&event_id)
		if err!=nil {
			utils.Fatalf("Error inserting event: %v",err)
		}
	}
	return nil
}
func sql_insert_account(addr *common.Address,owner_id Account_id_t,deleted int16,timestamp int,block_num Block_num_t) (Account_id_t,error) {
	var query string
	var account_id Account_id_t;
	addr_str:=hex.EncodeToString(addr.Bytes())
	block_del:=Block_num_t(0)
	if deleted==1 {
		block_del=block_num
	}
	query="INSERT INTO account(address,owner_id,deleted,block_del,ts_created,block_created) VALUES($1,$2,$3,$4,$5,$6) RETURNING account_id";
	row:=db.QueryRow(query,addr_str,owner_id,deleted,block_del,timestamp,block_num);
	err:=row.Scan(&account_id)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Error in Scan() for query '%v': %v",query,err))
	}
	if account_id==0 {
		utils.Fatalf(fmt.Sprintf("account_id after INSERT is 0"))
	}
	return account_id,nil
}
func sql_delete_block(block_num int,uncles []*types.Header) {
	var query string

	query="DELETE FROM block WHERE block_num=$1";	// ensure previously inserted blocks go away. cascading applied
	_,err:=db.Exec(query,block_num);
	if (err!=nil) {
		utils.Fatalf("DELETE from block failed: %v",err);
	}
	for i,uncle_hdr:=range uncles {
		block_hash:=hex.EncodeToString(uncle_hdr.Hash().Bytes());
		query="DELETE FROM uncle WHERE block_hash=$1";
		db.Exec(query,block_hash);
		if (err!=nil) {
			utils.Fatalf("DELETE from uncle failed: %v",err);
		}
		_=i
	}
}
func sql_update_main_stats(last_block Block_num_t) bool {
	var block_num_upper int = int(last_block)
	block_num_lower:=last_block-6171		// 6171=24hrs * 60mins * 60secs / 14sec  , or in other words, blocks in last 24 hours
	if (block_num_lower<0) {
		block_num_lower=0
	}
	var query string
	query="SELECT get_hashrate($1)::text AS hashrate"
	row := db.QueryRow(query,block_num_upper)
	var hashrate_str string="0"
	var hashrate_aux sql.NullString
	err:=row.Scan(&hashrate_aux);
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error in get_hashrate() query at Scan() : %v",err))
		utils.Fatalf("EthBot: error: %v",err);
	}
	if (hashrate_aux.Valid) {
		hashrate_str=hashrate_aux.String
	}

	query="SELECT get_blocktime($1)::text AS blocktime"
	row= db.QueryRow(query,block_num_upper)
	var blocktime_str string="0"
	var blocktime_aux sql.NullString
	err=row.Scan(&blocktime_aux);
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error in get_blocktime() query at Scan() : %v",err))
		utils.Fatalf("error: %v",err);
	}
	if (blocktime_aux.Valid) {
		blocktime_str=blocktime_aux.String
	}

	query="SELECT avg(num_tx)::text as avg_tx,round(avg(num_tx)/14,2)::text AS tx_sec FROM block WHERE block_num>=$1 AND block_num<=$2"
	row = db.QueryRow(query,block_num_lower,last_block)
	var avg_num_tx_aux,tx_per_sec_aux sql.NullString
	var avg_num_tx string="0"
	var tx_per_sec string="0"
	err=row.Scan(&avg_num_tx_aux,&tx_per_sec_aux);
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error in avg(num_tx) query at Scan() : %v",err))
		utils.Fatalf("error: %v",err);
	}
	if avg_num_tx_aux.Valid {
		avg_num_tx=avg_num_tx_aux.String

	}
	if tx_per_sec_aux.Valid {
		tx_per_sec=tx_per_sec_aux.String
	}

	query="SELECT round(avg(gas_price)/1000000000,2)::text FROM transaction WHERE block_num>=$1 AND block_num<=$2"
	row = db.QueryRow(query,block_num_lower,last_block)
	var gas_price_aux sql.NullString
	var gas_price_str string="0"
	err=row.Scan(&gas_price_aux);
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error in avg(gas_price) at Scan() : %v",err))
		utils.Fatalf("error: %v",err);
	}
	if gas_price_aux.Valid {
		gas_price_str=gas_price_aux.String
	}

	query="SELECT round(avg(gas_price*gas_used)/1000000000000000000,6)::text FROM transaction WHERE block_num>=$1 AND block_num<=$2"
	row = db.QueryRow(query,block_num_lower,last_block)
	var tx_cost_str string="0"
	var tx_cost_aux sql.NullString
	err=row.Scan(&tx_cost_aux);
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error in tx_cost at Scan() : %v",err))
		utils.Fatalf("error: %v",err);
	}
	if tx_cost_aux.Valid {
		tx_cost_str=tx_cost_aux.String
	}

	query="SELECT abs(last_balance)::text FROM account WHERE account_id=$1"
	row = db.QueryRow(query,NONEXISTENT_ADDRESS_ACCOUNT_ID)
	var supply_str string="0"
	err=row.Scan(&supply_str);
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error in getting supply at Scan() : %v",err))
		utils.Fatalf("error: %v",err);
	}

	query="SELECT avg(difficulty)::text AS difficulty FROM block WHERE block_num>=$1 AND block_num<=$2"
	row = db.QueryRow(query,block_num_lower,last_block)
	var difficulty_str string="0"
	var difficulty_aux sql.NullString
	err=row.Scan(&difficulty_aux);
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error in avg(difficulty) at Scan() : %v",err))
		utils.Fatalf("error: %v",err);
	}
	if difficulty_aux.Valid {
		difficulty_str=difficulty_aux.String
	}

	query=`SELECT round(avg(val_transferred)/1000000000000000000,2)::text FROM block WHERE block_num>=$1 AND block_num<=$2`
	row = db.QueryRow(query,block_num_lower,last_block)
	var volume_str string="0"
	var volume_aux sql.NullString
	err=row.Scan(&volume_aux);
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error in avg(volume) at Scan() : %v",err))
		utils.Fatalf("error: %v",err);
	}
	if volume_aux.Valid {
		volume_str=volume_aux.String
	}

	query=`
		SELECT count(*)::text as activity FROM (
			SELECT DISTINCT account_id FROM (
				(SELECT DISTINCT to_id AS account_id FROM value_transfer WHERE block_num>=$1 AND block_num<=$2)
				UNION ALL
				(SELECT DISTINCT from_id AS account_id FROM value_transfer WHERE block_num>=$1 AND block_num<=$2)
			) AS subdata 
		) AS data
		`
	row = db.QueryRow(query,block_num_lower,last_block)
	var activity_str string="0"
	var activity_aux sql.NullString
	err=row.Scan(&activity_aux);
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error in getting activity at Scan() : %v",err))
		utils.Fatalf("error: %v",err);
	}
	if activity_aux.Valid {
		activity_str=activity_aux.String
	}
	query=`
		UPDATE mainstats SET
			hash_rate=$1,
			block_time=$2,
			tx_per_block=$3,
			tx_per_sec=$4,
			gas_price=$5,
			tx_cost=$6,
			supply=$7,
			difficulty=$8,
			volume=$9,
			activity=$10,
			last_block=$11
		`
	_,err=db.Exec(query,hashrate_str,blocktime_str,avg_num_tx,tx_per_sec,gas_price_str,tx_cost_str,supply_str,difficulty_str,volume_str,activity_str,last_block);
	if (err!=nil) {
		utils.Fatalf("Update mainstats failed failed: %v",err);
	}

	return true
}
func sql_get_last_block_num() Block_num_t {

	var query string
	query="SELECT block_num FROM last_block LIMIT 1";
	row := db.QueryRow(query)
	var null_block_num sql.NullInt64
	var err error
	err=row.Scan(&null_block_num);
	if (err!=nil) {
		utils.Fatalf("Error in get_last_block_num(): %v",err)
	}
	if (null_block_num.Valid) {
		return Block_num_t(null_block_num.Int64)
	} else {
		return -2 // we use -2 and not -1 because -1 is already used when database is initialized
	}
}
func sql_set_last_block_num(block_num Block_num_t) {
	var bnum int = int(block_num)
	res,err:=db.Exec("UPDATE last_block SET block_num=$1 WHERE block_num < $1",bnum)
	if (err!=nil) {
		utils.Fatalf("sql_set_last_block_num() failed: %v",err);
	}
	affected_rows,err:=res.RowsAffected()
	if err!=nil {
		utils.Fatalf(fmt.Sprint("Error getting RowsAffected in sql_set_last_block(): %v",err))
	}
	if affected_rows>0 {
		ethbot_instance.export.Last_block_num=block_num
	}
}
func sql_mark_account_as_deleted(addr *common.Address,block_num Block_num_t) {

	var query string
	query=`UPDATE account SET deleted=1,block_del=$2 WHERE address=$1`
	_,err:=db.Exec(query,hex.EncodeToString(addr.Bytes()),block_num)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Cant update deleted flag on account %v, err=%v",hex.EncodeToString(addr.Bytes()),err))
	}

}
func sql_update_block_stats(block_num Block_num_t,value_transferred *big.Int,miner_reward *big.Int,num_TXs int,num_VTs int) {

	_,err:=db.Exec("UPDATE block SET done=TRUE,val_transferred=$2,miner_reward=$3,num_tx=$4,num_vt=$5 WHERE block_num=$1",block_num,value_transferred.String(),miner_reward.String(),num_TXs,num_VTs)
	if (err!=nil) {
		utils.Fatalf("sql_set_last_block_num() failed: %v",err);
	}
}
func sql_block_export_finished(block_num Block_num_t) bool {
	var query string
	query=`SELECT done FROM block WHERE block_num=$1`
	row:=db.QueryRow(query,block_num)
	var done bool;
	err:=row.Scan(&done)
	if err!=nil {
		if err==sql.ErrNoRows {
			return false
		} else {
			utils.Fatalf(fmt.Sprintf("Can't get block 'done' status: %v",err))
		}
	}
	return done
}
func sql_set_block_as_processed(block_num Block_num_t) {

	_,err:=db.Exec("UPDATE block SET done=TRUE WHERE block_num=$1",block_num)
	if (err!=nil) {
		utils.Fatalf("sql_set_block_as_processed() failed: %v",err);
	}
}
func sql_get_account_nonce(account_id Account_id_t) uint64 {
	var query string

	query=`SELECT num_tx FROM account WHERE account_id=$1`
	row:=db.QueryRow(query,account_id)
	var nonce uint64
	err:=row.Scan(&nonce)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Error at scanning num_tx: %v",err))
	}
	return nonce
}
