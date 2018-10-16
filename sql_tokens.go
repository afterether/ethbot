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
	"encoding/json"
	"strings"
	"strconv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/cmd/utils"
	"database/sql"
)
func lookup_token_account(addr_str string) (account_id Account_id_t) {
	query:="SELECT account_id FROM tokacct WHERE address=$1"
	row:=db.QueryRow(query,addr_str);
	err:=row.Scan(&account_id);
	if err!=nil {
		if (err==sql.ErrNoRows) {
			account_id=0
		} else {
			utils.Fatalf("Error in looking up token account: %v",err)
		}
	} else {
		// account found
	}
	return
}
func sql_get_last_token_block_num() Block_num_t {

	var query string
	query="SELECT block_num FROM last_token_block LIMIT 1";
	row := db.QueryRow(query)
	var null_block_num sql.NullInt64
	var err error
	err=row.Scan(&null_block_num);
	if (err!=nil) {
		utils.Fatalf("Error in get_last_token_block_num(): %v",err)
	}
	if (null_block_num.Valid) {
		return Block_num_t(null_block_num.Int64)
	} else {
		return -2 // we use -2 and not -1 because -1 is already used when database is initialized
	}
}
func sql_set_last_token_block_num(block_num Block_num_t) {
	var bnum int = int(block_num)
	_,err:=db.Exec("UPDATE last_token_block SET block_num=$1 WHERE block_num < $1",bnum)
	if (err!=nil) {
		utils.Fatalf("sql_set_last_token_block_num() failed: %v",err);
	}
}
func lookup_token(contract_id Account_id_t) (Account_id_t,int) {
	var query string
	query=`SELECT contract_id,decimals FROM token WHERE contract_id=$1`
	row:=db.QueryRow(query,contract_id)
	var id Account_id_t
	var decimals int
	err:=row.Scan(&id,&decimals)
	if err!=nil {
		if (err==sql.ErrNoRows) {
			return 0,0
		}
		utils.Fatalf("lookup_token error: %v",err)
	}
	return id,decimals
}
func lookup_token_by_addr(account_address string) Account_id_t {
	var query string
	query=`SELECT contract_id FROM token AS t,account AS a WHERE a.account_id=t.contract_id AND a.address=$1`
	row:=db.QueryRow(query,account_address)
	var id Account_id_t
	err:=row.Scan(&id)
	if err!=nil {
		if err==sql.ErrNoRows {
			return 0
		}
		utils.Fatalf("lookup_token error: %v",err)
	}
	return id
}
func sql_insert_token_account(address string,block_num Block_num_t,tx_id int64,timestamp int) (account_id Account_id_t) {
	addr:=common.HexToAddress(address)
	state_acct_id:=lookup_account(&addr)	// looks up account in StateDB i.e., the `account` table
	var query string
	query=`INSERT INTO tokacct(tx_id,state_acct_id,ts_created,block_created,address) VALUES($1,$2,$3,$4,$5) RETURNING account_id`
	row:=db.QueryRow(query,tx_id,state_acct_id,timestamp,block_num,address)
	err:=row.Scan(&account_id)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Scan token account_id failed after insert: %v",err))
	}
	return
}
func token_prev_balance_query() string {
	return `
		SELECT 
			o.block_num,
			o.from_id,
			o.to_id,
			o.from_balance::text,
			o.to_balance::text 
		FROM tokop o
		WHERE 
			(o.contract_id=$1) AND (o.block_num<=$2) AND
			(
				(o.to_id=$3) OR
				(o.from_id=$3)
			)
		ORDER BY
			o.block_num DESC,o.tokop_id DESC
		LIMIT 1
		`
}
func sql_insert_token_approval(contract_id Account_id_t,decoded_input []interface{}, event_log *Event_t, transfer *Token_transf_t, from *common.Address,tx_id int64,block_num Block_num_t,block_ts int,non_compliant bool,non_compliance_err string) int64 {
	log.Info("Entering insert_token_approval")
	var from_id,to_id Account_id_t
	var query string
	var value *big.Int = decoded_input[1].(*big.Int)
	if transfer.From==(common.Address{}) { // token creation 
		from_id=NONEXISTENT_ADDRESS_ACCOUNT_ID
	} else {
		from_id=lookup_token_account(hex.EncodeToString(transfer.From.Bytes()))
		if (from_id==0) {
			from_id=sql_insert_token_account(hex.EncodeToString(transfer.From.Bytes()),block_num,tx_id,block_ts)
			if (from_id==0) {
				utils.Fatalf("Can't insert token account")
			}
		}
	}
	if transfer.To==(common.Address{}) {	// some tokens burn tokens by assigning them to 0 address
		to_id=NONEXISTENT_ADDRESS_ACCOUNT_ID
	} else {
		to_id=lookup_token_account(hex.EncodeToString(transfer.To.Bytes()))
		if(to_id==0) {
			to_id=sql_insert_token_account(hex.EncodeToString(transfer.To.Bytes()),block_num,tx_id,block_ts)
			if (to_id==0) {
				utils.Fatalf("Can't insert token account")
			}
		}
	}

	query=`
		INSERT INTO approval (tx_id,contract_id,block_num,block_ts,from_id,to_id,value,non_compliant,non_compliance_err)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING approval_id
	`
	res:=db.QueryRow(query,
		tx_id,
		contract_id,
		block_num,
		block_ts,
		from_id,
		to_id,
		value.String(),
		non_compliant,
		non_compliance_err)
	var approval_id int64
	err:=res.Scan(&approval_id)
	log.Info(fmt.Sprintf("approval insert result: %v, approval_id=%v",err,approval_id))
	if err!=nil {
		utils.Fatalf("Error during aproval INSERT: %v",err)
	}
	if event_log.Event_id>0 {
		insert_event_tokop(true,event_log.Event_id,approval_id)
	}
	query=`SELECT tokop_id,from_id,to_id,value::text FROM tokop WHERE approval_tx_id=$1`
	rows,err:=db.Query(query,tx_id)
	if err!=nil {
		if err!=sql.ErrNoRows {
			utils.Fatalf(fmt.Sprintf("Error for query %v: %v",query,err))
		}
	}
	defer rows.Close()
	query=`INSERT INTO tokop_approval VALUES($1,$2,$3,$4,$5,$6)`
	for rows.Next() {		// this loop only executes if we update a block in the past, so it has to update all consumed approvals in tokop table
		var tokop_id int64
		var v_from_id,v_to_id int
		var v_value string
		err=rows.Scan(&tokop_id,&v_from_id,&v_to_id,&v_value)
		if err!=nil {
			utils.Fatalf(fmt.Sprintf("Error at Scan() tokop_id: %v",err))
		}
		result,err:=db.Exec(query,tokop_id,approval_id,contract_id,v_from_id,v_to_id,v_value)
		if err!=nil {
			utils.Fatalf(fmt.Sprintf("Error inserting tokop_approval(%v,%v) link: %v",tokop_id,approval_id,err))
		}
		rows_affected,err:=result.RowsAffected()
		if err!=nil {
			utils.Fatalf(fmt.Sprintf("Error geting rows_affected for tokop_approval() insert: %v",err))
		}
		if rows_affected==0 {
			utils.Fatalf(fmt.Sprintf("Rows_affected = 0 for insert of tokop_approval(%v,%v)",tokop_id,approval_id))
		}
	}
	return approval_id
}
func sql_insert_token_operation(token_op int,iface int,values []interface{},contract_id Account_id_t,event_log *Event_t,transfer *Token_transf_t, block_num Block_num_t,timestamp int,tx_id int64,non_compliant bool,non_compliance_err string) int64  {
	var query string

	var from_id,to_id Account_id_t

	if contract_id<1 {
		utils.Fatalf(fmt.Sprintf("Invalid contract_id=%v during processing of block %v",contract_id,block_num))
	}
	if token_op==TOKENOP_UNKNOWN {
		return 0
	}
	if transfer.From==(common.Address{}) { // token creation 
		from_id=NONEXISTENT_ADDRESS_ACCOUNT_ID
	} else {
		from_id=lookup_token_account(hex.EncodeToString(transfer.From.Bytes()))
		if (from_id==0) {
			from_id=sql_insert_token_account(hex.EncodeToString(transfer.From.Bytes()),block_num,tx_id,timestamp)
			if (from_id==0) {
				utils.Fatalf("Can't insert token account")
			}
		}
	}
	if transfer.To==(common.Address{}) {	// some tokens burn tokens by assigning them to 0 address
		to_id=NONEXISTENT_ADDRESS_ACCOUNT_ID
	} else {
		to_id=lookup_token_account(hex.EncodeToString(transfer.To.Bytes()))
		if(to_id==0) {
			to_id=sql_insert_token_account(hex.EncodeToString(transfer.To.Bytes()),block_num,tx_id,timestamp)
			if (to_id==0) {
				utils.Fatalf("Can't insert token account")
			}
		}
	}
	to_balance:=big.NewInt(0)
	from_balance,_:=sql_get_token_previous_balance(contract_id,block_num,from_id);
	to_balance,_=sql_get_token_previous_balance(contract_id,block_num,to_id);

	neg_value:=big.NewInt(0)
	neg_value.Neg(transfer.Value)
	if (token_op==TOKENOP_TRANSFER) || (token_op==TOKENOP_TRANSFER_FROM) {	// only this kind of token operation affects balance
		if (to_id==from_id) { // self transfer 
			to_balance.Set(from_balance)
		} else {
			from_balance.Add(from_balance,neg_value)
			to_balance.Add(to_balance,transfer.Value)
		}
	}
	value_str:=transfer.Value.String();
	from_balance_str:=from_balance.String()
	to_balance_str:=to_balance.String()

	var approval_id int64 = 0
	var approval_tx_id int64 = 0
	if token_op==TOKENOP_TRANSFER_FROM {
		query=`SELECT approval_id,tx_id,value::text FROM approval WHERE (contract_id=$1) AND (to_id=$2) AND (from_id=$3) AND (expired=FALSE)`
		dquery:=fmt.Sprintf(`SELECT approval_id,tx_id,value::text FROM approval WHERE (contract_id=%v) AND (to_id=%v) AND (from_id=%v) AND (expired=FALSE)`,contract_id,to_id,from_id)
		log.Info(fmt.Sprintf(`Approval seeking query: %v`,dquery))
		rows,err:=db.Query(query,contract_id,to_id,from_id)
		if err!=nil {
			utils.Fatalf(fmt.Sprintf("Error getting approval in insert_tokop(): %v",err))
		}
		defer rows.Close()
		var approved_value_str string
		if rows.Next() {
			err:=rows.Scan(&approval_id,&approval_tx_id,&approved_value_str)
			if err!=nil {
				utils.Fatalf(fmt.Sprintf("Scan() at getting approval_id failed: %v",err))
			}
			if rows.Next() {
				utils.Fatalf(fmt.Sprintf("Too many rows for contract_id=%v, to_id=%v. can't register corresponding aproval",contract_id,to_id))
			}
			val:=values[2].(*big.Int)
			approved_value:=big.NewInt(0)
			approved_value.SetString(approved_value_str,10)
			if approved_value.Cmp(val)<0 {
				utils.Fatalf(fmt.Sprintf("Approved value is less than transferred value contract_id=%v, from_id=%v, to_id=%v",contract_id,from_id,to_id))
			}
			log.Info(fmt.Sprintf("Found approval_id=%v, value_approved=%v, value_consumed=%v",approval_id,approved_value_str,val.String()))
		} else {
			non_compliant=true
			non_compliance_err="No approval operation was found for this transferFrom()";
		}
	}
	query=`INSERT INTO tokop(tx_id,approval_tx_id,contract_id,block_num,block_ts,from_id,to_id,value,from_balance,to_balance,kind,non_compliant,non_compliance_err) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING tokop_id`
	d_query:=fmt.Sprintf(`INSERT INTO tokop(tx_id,approval_tx_id,contract_id,block_num,block_ts,from_id,to_id,value,from_balance,to_balance,kind,non_compliant,non_compliance_err) VALUES(%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v) RETURNING tokop_id`,tx_id,approval_tx_id,contract_id,block_num,timestamp,from_id,to_id,value_str,from_balance_str,to_balance_str,token_op,non_compliant,non_compliance_err)

	log.Info(fmt.Sprintf("query=%v",d_query))
	if event_log!=nil {
		log.Info(fmt.Sprintf("before inserting event_id=%v",event_log.Event_id))
	}
	var tokop_id int64;
	err:=db.QueryRow(query,
		tx_id,
		approval_tx_id,
		contract_id,
		block_num,
		timestamp,
		from_id,
		to_id,
		value_str,
		from_balance_str,
		to_balance_str,
		token_op,
		non_compliant,
		non_compliance_err).Scan(&tokop_id);
	if (err!=nil) {
		utils.Fatalf("Error inserting into `tokop` table: %v",err);
	}
	if event_log!=nil {
		if event_log.Event_id>0 {
			insert_event_tokop(false,event_log.Event_id,tokop_id)
		}
	}
	if approval_id>0 {
		query=`INSERT INTO tokop_approval VALUES($1,$2,$3,$4,$5,$6)`
		res,err:=db.Exec(query,tokop_id,approval_id,contract_id,from_id,to_id,value_str)
		if err!=nil {
			utils.Fatalf(fmt.Sprintf("inserting tokop_approval(%v,%v) link failed: %v",tokop_id,approval_id,err))
		}
		rows_affected,err:=res.RowsAffected()
		if err!=nil {
			utils.Fatalf(fmt.Sprintf("Error getting rows affected after inserting tokop_approval link: %v",err))
		}
		if rows_affected==0 {
			utils.Fatalf(fmt.Sprintf("Rows affected after inserting tokop_approval(%v,%v) link failed",tokop_id,approval_id))
		}
	}
	return tokop_id;
}
func insert_event_tokop(approval bool,event_id,tokop_id int64) {
	var query string

	if approval {
		query=`INSERT INTO event_approval(event_id,approval_id) VALUES ($1,$2)`
	} else {
		query=`INSERT INTO event_tokop(event_id,tokop_id) VALUES ($1,$2)`
	}
	res,err:=db.Exec(query,event_id,tokop_id)
	if err!=nil {
		msg:=fmt.Sprintf("Query: %v failed , params: (%v,%v)",query,event_id,tokop_id)
		utils.Fatalf(fmt.Sprintf("%v: %v",msg,err))
	}
	rows_affected,err:=res.RowsAffected()
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Error geting affected rows on insert into 'event_tokop': %v, approval=%v ",err,approval))
	}
	if rows_affected==0 {
		utils.Fatalf(fmt.Sprintf("Failed to insert event_tokop(%v,%v), affected rows=0 approval=%v",event_id,tokop_id,approval))
	}
}
func sql_get_token_previous_balance(contract_id Account_id_t,block_num Block_num_t,account_id Account_id_t) (*big.Int,Block_num_t) {

	var query string
	var aux_from,aux_to Account_id_t
	var aux_from_bal_str,aux_to_bal_str string
	var balance_block_num Block_num_t = -1

	balance:=big.NewInt(0)

	query=token_prev_balance_query();
	row:=db.QueryRow(query,contract_id,block_num,account_id);
	err:=row.Scan(&balance_block_num,&aux_from,&aux_to,&aux_from_bal_str,&aux_to_bal_str)
	if (err!=nil) {
		if err==sql.ErrNoRows {
			// nothing, balance is kept at 0
		} else {
			log.Error(fmt.Sprintf("EthBot: getting TOKEN balance for account_id %v failed",account_id));
			utils.Fatalf("error",err);
		}
	} else { // rows with previous balances were found
		if aux_from==aux_to { // self transfer
			if aux_to==account_id {
				balance.SetString(aux_to_bal_str,10)
			} else {
				utils.Fatalf(fmt.Sprintf("EthBot: unknown use case 1 at sql_get_token_previous_balance() for TOKEN account_id=%v, block_num=%v ",account_id,block_num))
			}
		} else {
			if aux_from==account_id {
				balance.SetString(aux_from_bal_str,10)
			} else {
				if aux_to==account_id {
					balance.SetString(aux_to_bal_str,10)
				} else {
					utils.Fatalf(fmt.Sprintf("EthBot: unknown use case 2 at sql_get_token_previous_balance() for TOKEN account_id=%v, block_num=%v ",account_id,block_num))
				}
			}
		}
	}
	return balance,balance_block_num
}
func sql_insert_token(t *Token_t,ti *Token_info_t) Account_id_t {
	var query string

	account_id:=lookup_account(t.contract_addr)
	if(account_id==0) {
		utils.Fatalf(fmt.Sprintf("Token's contract address %v not found in the database",hex.EncodeToString(t.contract_addr.Bytes())))
	}

	query=`DELETE FROM token WHERE contract_id=$1`
	_,err:=db.Exec(query,account_id)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Error deleting token record: %v",err))
	}

	total_supply_str:="0"
	if (ti.m_ERC20_total_supply) && (t.total_supply!=nil) {
		total_supply_str=t.total_supply.String()
	}
	query=`INSERT INTO token(contract_id,block_created,created_tx_id,decimals,total_supply,symbol,name) VALUES($1,$2,$3,$4,$5,$6,$7)`
	dquery:=strings.Replace(query,`$1`,strconv.Itoa(int(account_id)),-1)
	dquery=strings.Replace(dquery,`$2`,strconv.Itoa(int(ti.block_created)),-1)
	dquery=strings.Replace(dquery,`$3`,strconv.Itoa(int(t.tx_id)),-1)
	dquery=strings.Replace(dquery,`$4`,strconv.Itoa(int(t.decimals)),-1)
	dquery=strings.Replace(dquery,`$5`,total_supply_str,-1)
	dquery=strings.Replace(dquery,`$6`,t.symbol,-1)
	dquery=strings.Replace(dquery,`$7`,t.name,-1)
	log.Info(fmt.Sprintf("INSERT query: %v",dquery))
	_,err=db.Exec(query,
		account_id,
		ti.block_created,
		t.tx_id,
		t.decimals,
		total_supply_str,
		t.symbol,
		t.name)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Error inserting token: %v",err))
	}
	sql_insert_token_info(ti)
	return account_id
}
func sql_insert_token_info(ti *Token_info_t) {
	var query string
	query=`
		INSERT INTO token_info(
			contract_id,block_created,fully_discovered,name,symbol,
			i_ERC20,i_burnable,i_mintable,i_ERC223,i_ERC677,i_ERC721,i_ERC777,i_ERC827,
			nc_ERC20,nc_ERC223,nc_ERC677,nc_ERC721,nc_ERC777,nc_ERC827,
			m_ERC20_name,m_ERC20_symbol,m_ERC20_decimals,m_ERC20_total_supply,m_ERC20_balance_of,m_ERC20_allowance,
			m_ERC20_transfer,m_ERC20_approve,m_ERC20_transfer_from,
			m_ERC20_ext_burn,m_ERC20_ext_mint,m_ERC20_ext_freeze,m_ERC20_ext_unfreeze,
			e_ERC20,e_ERC223,e_ERC677,e_ERC721,e_ERC777,e_ERC827)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38)
	`
	res,err:=db.Exec(query,ti.Contract_id,ti.block_created,ti.fully_discovered,ti.name,ti.symbol,
		ti.i_ERC20,ti.i_burnable,ti.i_mintable,ti.i_ERC223,ti.i_ERC677,ti.i_ERC721,ti.i_ERC777,ti.i_ERC827,
		ti.nc_ERC20,ti.nc_ERC223,ti.nc_ERC677,ti.nc_ERC721,ti.nc_ERC777,ti.nc_ERC827,
		ti.m_ERC20_name,ti.m_ERC20_symbol,ti.m_ERC20_decimals,ti.m_ERC20_total_supply,ti.m_ERC20_balance_of,ti.m_ERC20_allowance,
		ti.m_ERC20_transfer,ti.m_ERC20_approve,ti.m_ERC20_transfer_from,ti.m_ERC20_ext_burn,ti.m_ERC20_ext_mint,ti.m_ERC20_ext_freeze,ti.m_ERC20_ext_unfreeze,
		ti.e_ERC20,ti.e_ERC223,ti.e_ERC677,ti.e_ERC721,ti.e_ERC777,ti.e_ERC827)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Error inserting token info: %v",err))
	}
	rows_affected,err:=res.RowsAffected()
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Error getting affected rows: %v",err))
	}
	if rows_affected==0 {
		utils.Fatalf(fmt.Sprintf("After inserting token_info() record rows_affected is 0"))
	}
}
func sql_insert_event(contract_id Account_id_t,tokop_id int64, event_log *types.Log,block_num Block_num_t,tx_id int64) int64 {

	if tx_id<1 {
		tx_hash:=hex.EncodeToString(event_log.TxHash.Bytes())
		tx_id=lookup_transaction_by_hash(tx_hash)
		if tx_id==-1 {
			utils.Fatalf("can't locate transaction %v",tx_hash)
		}
	}
	topics,err:=json.Marshal(event_log.Topics)
	if err!=nil {
		utils.Fatalf("json encoding of event topics failed %v",topics)
	}
	data,err:=json.Marshal(event_log.Data)
	if err!=nil {
		utils.Fatalf("json encoding of event data failed %v",data)
	}
	var query string
	query=`INSERT INTO event(tx_id,block_num,contract_id,tokop_id,log_index,topics,data) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING event_id`
	row:=db.QueryRow(query,tx_id,block_num,contract_id,tokop_id,event_log.Index,topics,data)
	var event_id int64
	err=row.Scan(&event_id)
	if err!=nil {
		utils.Fatalf("Error inserting event: %v",err)
	}
	return event_id
}
func sql_delete_tokens_for_block(block_num Block_num_t) {
	var query string
	query=`DELETE FROM tokens_processed WHERE block_num=$1`
	_,err:=db.Exec(query,block_num)
	if err!=nil {
		utils.Fatalf("error deleting tokop records: %v",err)
	}
}
func mark_tokens_as_processed(block_num Block_num_t) {
	var query string

	query=`UPDATE tokens_processed SET done=TRUE WHERE block_num=$1`
	res,err:=db.Exec(query,block_num)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Error at marking tokens as processed: %v",err))
	}
	rows_affected,err:=res.RowsAffected()
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Error getting RowsAffected after UPDATE tokens_processed: %v",err))
	}
	if rows_affected==0 {
		utils.Fatalf(fmt.Sprintf("Could not update tokens_processed status for block %v",block_num))
	}
}
func prepare_token_processing(block_num Block_num_t) {
	var query string

	query=`INSERT INTO tokens_processed(block_num) VALUES($1)`
	_,err:=db.Exec(query,block_num)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Error at marking tokens as processed: %v",err))
	}
}
func are_tokens_processed(block_num Block_num_t) bool {
	var query string

	query=`SELECT done FROM tokens_processed WHERE block_num=$1`
	row:=db.QueryRow(query,block_num);
	var done bool
	err:=row.Scan(&done)
	if err!=nil {
		if err==sql.ErrNoRows {
			return false
		}
		utils.Fatalf(fmt.Sprintf("Can't get row at are_tokens_processed(): %v",err))
	}
	return true
}
func sql_get_events(valtr_id int64) ([]*Event_t) {
	var query string
	query=`SELECT event_id,topics,data FROM event WHERE valtr_id=$1`

	rows,err:=db.Query(query,valtr_id)
	if err!=nil {
		if err==sql.ErrNoRows {
			return nil
		}
		utils.Fatalf(fmt.Sprintf("Error for query %v: %v",query,err))
	}
	var events []*Event_t
	for rows.Next() {
		event:=&Event_t{}
		var topics_str string
		err=rows.Scan(&event.Event_id,&topics_str,&event.Data)
		if err!=nil {
			utils.Fatalf(fmt.Sprintf("Scan() error at getting event data: %v",err))
		}
		tmp_topics:=make([]common.Hash,0,3)
		err=json.Unmarshal([]byte(topics_str),&tmp_topics)
		if err!=nil {
			utils.Fatalf(fmt.Sprintf("Unmarshaling topics gave an error: %v",err))
		}
		event.Topics=tmp_topics
		events=append(events,event)
	}
	return events
}
