package main

import (
	"os"
	"fmt"
	"time"
	"net"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
    "github.com/ethereum/go-ethereum/cmd/utils"
	"database/sql"
	_ "github.com/lib/pq"
)
const debug_sql_exec_time = false
var db *sql.DB

func report_time(start_ts int64,desc string) {
	var end_ts int64
	end_ts=time.Now().UnixNano() / int64(time.Millisecond)
	log.Info(fmt.Sprintf("%v time: %v ms",desc,(end_ts-start_ts)))
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
}
func lookup_block_by_hash(hash string) (Block_id_t,Block_num_t) {
	var query string
	query="SELECT block_id,block_num FROM block WHERE block_hash=$1"
	var block_id Block_id_t
	var block_num Block_num_t
	var start_ts int64=0

	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}
	err:=db.QueryRow(query,hash).Scan(&block_id,&block_num);
	if debug_sql_exec_time {
		report_time(start_ts,"lookup_block_by_hash()")
	}
	if (err!=nil) {
		if (err==sql.ErrNoRows) {
			return -1,-1;
		} else {
			utils.Fatalf("Error looking up block by hash: %v",err);
		}
	}
	return block_id,block_num
}
func lookup_account(addr *common.Address) Account_id_t {

	if (addr==nil) {
		utils.Fatalf("lookup account with null address")
	}
	account_id,exists:=accounts_cache[*addr]
	if exists {
		return account_id
	} else {
		addr_str:=hex.EncodeToString(addr.Bytes())
		account_id,_=lookup_account_SQL(addr_str)
		if account_id!=0 {
			accounts_cache[*addr]=account_id
		}
		return account_id
	}
}
func lookup_account_SQL(addr_str string) (account_id Account_id_t,owner_id Account_id_t) {
	query:="SELECT account_id,owner_id FROM account WHERE address=$1"
	var start_ts int64=0
	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}
	row:=db.QueryRow(query,addr_str);
	err:=row.Scan(&account_id,&owner_id);
	if debug_sql_exec_time {
		report_time(start_ts,"lookup_account()")
	}
	if (err==sql.ErrNoRows) {
		return 0,0
	} else {
		return account_id,owner_id
	}
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
func block2sql(chain *core.BlockChain,block *types.Block,num_tx int) (Block_id_t,error) {
	var total_dif *big.Int
	block_num:=Block_num_t(block.NumberU64())
	if block_num==0 {
		total_dif=block.Header().Difficulty;
	} else {
		total_dif=chain.GetTdByHash(block.Hash());
	}
	sql_delete_block(int(block_num),block.Uncles());	// remove all the previous data for this block

	uncles:=block.Uncles();

	block_id,err:=sql_insert_block(block.Header(),total_dif,len(uncles),block.Size().Int64());
	if (err!=nil) {
		return 0,err;
	}

	// Inserts not only the main block, but uncles too, uncles have the same block_num, but different uncle_pos. main block has uncle_pos=0
	for i,uncle_hdr:=range uncles {
		uncle_pos:=i+1
		_,err:=sql_insert_uncle(block_id,uncle_hdr,total_dif,uncle_pos);
		if (err!=nil) {
			return Block_id_t(0),err;
		}
	}
	return block_id,nil;
}
func sql_insert_block(hdr *types.Header,total_dif *big.Int,num_uncles int,size int64) (Block_id_t,error) {
	var err error
	var miner_id Account_id_t
	var query string

	block_hash:=hex.EncodeToString(hdr.Hash().Bytes());
	block_num:=Block_num_t(hdr.Number.Uint64())
	parent_id,_:=lookup_block_by_hash(hex.EncodeToString(hdr.ParentHash.Bytes()));
	if (parent_id == -1) {
		if block_num!=0 {
			return -1,ErrAncestorNotFound
		}
	}
	miner_id=lookup_account(&hdr.Coinbase);
	if (miner_id==0) {
		var err error
		miner_id,err=sql_insert_account(&hdr.Coinbase,0)
		if (err!=nil) {
			return 0,err
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
		RETURNING block_id`;
	var block_id Block_id_t
	var start_ts int64=0
	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}
	err=db.QueryRow(query,
		parent_id,
		block_num,
		time_str,
		miner_id,
		hdr.Difficulty.String(),
		total_dif_str,
		hdr.GasLimit.String(),
		hdr.GasUsed.String(),
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
		extra).Scan(&block_id);
	if debug_sql_exec_time {
		report_time(start_ts,"INSERT into block")
	}
	if (err!=nil) {
		utils.Fatalf("Error inserting into `blocks` table: %v",err);
	}
	return block_id,nil;
}
func sql_insert_uncle(block_id Block_id_t,hdr *types.Header,total_dif *big.Int,uncle_pos int) (Block_id_t,error) {
	var err error
	var miner_id Account_id_t
	var query string
	var parent_block_num Block_num_t = -1

	block_hash:=hex.EncodeToString(hdr.Hash().Bytes());
	block_num:=Block_num_t(hdr.Number.Uint64())
	parent_id,tmp_parent_num:=lookup_block_by_hash(hex.EncodeToString(hdr.ParentHash.Bytes()));
	if (parent_id == -1) {
		if block_num!=0 {
			utils.Fatalf("Parent for uncle not found. block=",block_num)
		}
	} else {
		parent_block_num=tmp_parent_num
	}
	miner_id=lookup_account(&hdr.Coinbase)
	if (miner_id==0) {
		miner_id,err=sql_insert_account(&hdr.Coinbase,0);
		if (err!=nil) {
			return 0,err
		}
	}
	total_dif_str:=total_dif.String();
	time_str:=hdr.Time.String();
	extra:=hdr.Extra

	nonce_str:=fmt.Sprintf("%d",hdr.Nonce.Uint64());

	query=`
		INSERT INTO uncle(
			block_id,
			parent_id,
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
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20) 
		RETURNING uncle_id`;
	var uncle_id Block_id_t
	var start_ts int64=0
	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}
	err=db.QueryRow(query,
		block_id,
		parent_id,
		block_num,
		parent_block_num,
		time_str,
		miner_id,
		uncle_pos,
		hdr.Difficulty.String(),
		total_dif_str,
		hdr.GasLimit.String(),
		hdr.GasUsed.String(),
		nonce_str,
		block_hash,
		hex.EncodeToString(hdr.UncleHash.Bytes()),
		hex.EncodeToString(hdr.Root.Bytes()),
		hex.EncodeToString(hdr.TxHash.Bytes()),
		hex.EncodeToString(hdr.ReceiptHash.Bytes()),
		hex.EncodeToString(hdr.MixDigest.Bytes()),
		hdr.Bloom.Bytes(),
		extra).Scan(&uncle_id);
	if debug_sql_exec_time {
		report_time(start_ts,"INSERT into uncle")
	}
	if (err!=nil) {
		utils.Fatalf("Error inserting into `uncles` table: %v",err);
	}
	return uncle_id,nil;
}
func sql_insert_transaction(tx_invalid bool,from *common.Address,tx *types.Transaction,receipt *types.Receipt,tx_err error,block_id Block_id_t,block_num Block_num_t,tx_index int,timestamp int,num_VTs int,amount_transferred *big.Int) (int64,error) {
	var from_id Account_id_t
	if tx_invalid {
		from_id=WRONG_TRANSACTION_SENDER_ACCOUNT_ID
	} else {
		from_id=lookup_account(from)
	}
	if (from_id==0) {
		var err error
		from_id,err=sql_insert_account(from,0)
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
			to_id,err=sql_insert_account(to,0)
			if (err!=nil) {
				return 0,err
			}
		}
	}
	var transaction_id int64;
	var query string;
	v,r,s:=tx.RawSignatureValues();
	var payload []byte
	//payload=tx.Data();	// commented because we don't really need this in the SQL database , it is stored in LevelDB anyway
	tx_err_str:=""
	if (tx_err!=nil) {
		tx_err_str=tx_err.Error()
	}

	query=`
		INSERT INTO transaction(
			from_id,
			to_id,
			gas_limit,
			gas_used,
			tx_value,
			gas_price,
			nonce,
			block_id,
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
			payload
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20) 
		RETURNING tx_id`;
	var nonce int=int(tx.Nonce())
	var err error
	var tx_status uint=0
	if receipt!=nil { // error in transaction
		tx_status=receipt.Status
	}
	var gas_used string="0"
	if (receipt!=nil) {
		gas_used=receipt.GasUsed.String()
	}
	var start_ts int64=0
	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}
	err=db.QueryRow(query,
		from_id,
		to_id,
		tx.Gas().String(),
		gas_used,
		tx.Value().String(),
		tx.GasPrice().String(),
		nonce,
		block_id,
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
		payload).Scan(&transaction_id);
	if debug_sql_exec_time {
		report_time(start_ts,"INSERT into transaction")
	}
	if (err!=nil) {
		utils.Fatalf("Can't insert transaction. error=%v",err);
	}
	return transaction_id,nil;
}
func sql_query_get_balance() string {
		return `
		SELECT 
			v.from_id,
			v.to_id,
			v.from_balance::text,
			v.to_balance::text 
		FROM value_transfer v
		WHERE 
			(v.block_num<=$1) AND
			(
				(v.to_id=$2) OR
				(v.from_id=$2)
			)
		ORDER BY
			v.block_num DESC,v.valtr_id DESC
		LIMIT 1
		`
}
func get_previous_balance(block_num Block_num_t,account_id Account_id_t,addr *common.Address) *big.Int { // both parameters must refer to the same account
	if block_num==0 { // Genesis block
		if account_id==NONEXISTENT_ADDRESS_ACCOUNT_ID {
			if ethbot_instance.export.non_existent_balance==nil {
				ethbot_instance.export.non_existent_balance=big.NewInt(0)
			}
			return ethbot_instance.export.non_existent_balance
		}
		return big.NewInt(0)
	}
	if account_id==NONEXISTENT_ADDRESS_ACCOUNT_ID {
		if ethbot_instance.export.non_existent_balance==nil {
			ethbot_instance.export.non_existent_balance=get_previous_balance_SQL(block_num,account_id)
			return ethbot_instance.export.non_existent_balance
		} else {
			return ethbot_instance.export.non_existent_balance
		}
	}
	prev_balance,balance_exists:=modified_balances[account_id]
	if balance_exists {
		return prev_balance // there were some operations on this account during this block
	} else {
		if (addr==nil) {
			utils.Fatalf("get_previous_balance, addr=nil, block_num=%v, account_id=%v",block_num,account_id)
		}
		var balance_on_chain *big.Int
		if (previous_state.Exist(*addr)) {
			balance_on_chain=previous_state.GetBalance(*addr)
		} else {
			balance_on_chain=big.NewInt(0)
		}
		new_bal:=big.NewInt(0)
		new_bal.Set(balance_on_chain)
		modified_balances[account_id]=new_bal
		return new_bal
	}
}
func get_previous_balance_SQL(block_num Block_num_t,account_id Account_id_t) *big.Int {

	var query string
	var aux_from,aux_to Account_id_t
	var aux_from_bal_str,aux_to_bal_str string

	balance:=big.NewInt(0)

	query=sql_query_get_balance()
	var start_ts int64=0
	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}
	row:=db.QueryRow(query,block_num,account_id);
	err:=row.Scan(&aux_from,&aux_to,&aux_from_bal_str,&aux_to_bal_str)
	if debug_sql_exec_time {
		report_time(start_ts,fmt.Sprintf("get_previous_balance(%v,%v)",account_id,account_id))
	}
	if (err!=nil) {
		if err==sql.ErrNoRows {
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
func sql_insert_value_transfer(transf *Value_transfer_t, transaction_id int64) (int64,error) {
	var valtr_id int64
	var query string

	var err error
	var from_id,to_id Account_id_t
	if transf.src==nil {
		from_id=NONEXISTENT_ADDRESS_ACCOUNT_ID
	} else {
		from_id,err=sql_insert_account(transf.src,0)
		if (err!=nil) {
			return 0,err
		}
	}
	if transf.dst==nil {
		to_id=NONEXISTENT_ADDRESS_ACCOUNT_ID
	} else {
		var owner_id Account_id_t = 0
		if transf.kind==VALTRANSF_CONTRACT_CREATION {
			owner_id=from_id
		}
		to_id,err=sql_insert_account(transf.dst,owner_id)
		if (err!=nil) {
			return 0,err
		}
	}
	from_balance:=get_previous_balance(transf.block_num,from_id,transf.src)
	to_balance := big.NewInt(0)
	if transf.kind==VALTRANSF_CONTRACT_SELFDESTRUCT { // SELFDESTRUCT (SUICIDE) transfers the remaining balance to recipient
		to_id=lookup_account(transf.dst)
		to_balance=get_previous_balance(transf.block_num,to_id,transf.dst)
		transf.value.Set(from_balance)
	}
	neg_value:=big.NewInt(0)
	neg_value.Neg(transf.value)
	if (to_id==from_id) { // self transfer 
		to_balance.Set(from_balance)
	} else {
		if transf.kind==VALTRANSF_CONTRACT_SELFDESTRUCT {
			// to_balance was already set in the if() block above
		} else {
			to_balance=get_previous_balance(transf.block_num,to_id,transf.dst)
		}
		from_balance.Add(from_balance,neg_value)
		to_balance.Add(to_balance,transf.value)
	}
	value_str:=transf.value.String();
	from_balance_str:=from_balance.String()
	to_balance_str:=to_balance.String()
	block_id:=int(transf.block_id)
	vt_err_str:=transf.err_str
	query="INSERT INTO value_transfer(tx_id,block_id,block_num,from_id,to_id,value,from_balance,to_balance,kind,error) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING valtr_id"
	var start_ts int64=0
	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}
	var row *sql.Row
	if (transaction_id==-1) { // this is the case when a value_transfer record is not linked to a transaction record
		var null_var sql.NullInt64
		transaction_id_param:=null_var
		row=db.QueryRow(query,transaction_id_param,block_id,transf.block_num,from_id,to_id,value_str,from_balance_str,to_balance_str,transf.kind,vt_err_str)
	} else {
		transaction_id_param:=transaction_id
		row=db.QueryRow(query,transaction_id_param,block_id,transf.block_num,from_id,to_id,value_str,from_balance_str,to_balance_str,transf.kind,vt_err_str)
	}
	err=row.Scan(&valtr_id)
	if debug_sql_exec_time {
		report_time(start_ts,"INSERT into value_transfer")
	}
	if (err!=nil) {
		utils.Fatalf("Can't insert value_transfer, error=",err);
	}

	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}
	if debug_sql_exec_time {
		report_time(start_ts,"UPDATE last_balance of account table")
	}
	return valtr_id,nil;
}
func sql_insert_account(addr *common.Address,owner_id Account_id_t) (Account_id_t,error) {
	var query string
	var account_id Account_id_t;
	query="SELECT account_id FROM account WHERE address=$1";
	addr_str:=get_hex_addr(addr);
	var start_ts int64=0
	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}
	row:=db.QueryRow(query,addr_str);
	err:=row.Scan(&account_id)
	if debug_sql_exec_time {
		report_time(start_ts,"sql_insert_account(SELECT account_id from account)")
	}
	if (err==sql.ErrNoRows) {
		query="INSERT INTO account(address,owner_id) VALUES($1,$2) RETURNING account_id";
		err:=db.QueryRow(query,addr_str,owner_id).Scan(&account_id);
		if (err!=nil) {
			utils.Fatalf("Error: can't insert into `accounts` table. error=%v",err);
		}
	} else {
		if (err==nil) {
			// nothing
		} else {
			utils.Fatalf("Error: can't insert into `accounts` table. error=%v",err);
		}
	}
	return account_id,nil
}
func sql_verify_account_lot(status_write_back chan bool, block_num Block_num_t,account_ids string,blockchain_balances map[Account_id_t]*big.Int ) {
	var query string;

	for account_id,balance:= range blockchain_balances {
		// the following query gets last balance for the account at that time in the past
		query=`
		SELECT 
			v.valtr_id,
			v.block_num,
			v.from_id,
			v.to_id,
			v.from_balance::text,
			v.to_balance::text 
		FROM value_transfer v
		WHERE 
			(v.block_num<=$1) AND
			(
				(v.to_id=$2) OR
				(v.from_id=$2)
			)
		ORDER BY
			v.block_num DESC,v.valtr_id DESC
		LIMIT 1
		`
		rows:=db.QueryRow(query,block_num,account_id);

		var valtr_id int64
		var last_block_num Block_num_t
		var from_id,to_id Account_id_t
		var sql_account_from_balance_str,sql_account_to_balance_str string
		var sql_account_from_balance,sql_account_to_balance big.Int

		err:=rows.Scan(&valtr_id,&last_block_num,&from_id,&to_id,&sql_account_from_balance_str,&sql_account_to_balance_str);
		if err!=nil {
			if err==sql.ErrNoRows {
				valtr_id=-1
				last_block_num=block_num
				from_id=account_id
				to_id=account_id
				sql_account_from_balance_str="0"
				sql_account_to_balance_str="0"
				set_verif_error(valtr_id,fmt.Sprintf("SQL DB does not have records for account_id=%v, last block_num<=%v with expected balance=%v, balance in SQL DB=NULL",account_id,last_block_num,balance.String()))
				break;
			} else {
				utils.Fatalf("failed to execute query: %v, error=%v",query,err)
			}
		}

		sql_account_from_balance.SetString(sql_account_from_balance_str,10);
		sql_account_to_balance.SetString(sql_account_to_balance_str,10);

		if from_id==to_id {	// self transfer
			if sql_account_to_balance.Cmp(balance)!=0 {
				set_verif_error(valtr_id,fmt.Sprintf("(selftransfer) account_id=%v with balance=%v in statedb does not match the balance in SQL DB=%v (last block in SQL = %v)",account_id,balance.String(),sql_account_to_balance.String(),last_block_num))
				break;
			}
		} else {
			if (to_id==account_id) {
			  if sql_account_to_balance.Cmp(balance)!=0 {
				set_verif_error(valtr_id,fmt.Sprintf("(to) account_id=%v, with balance=%v in statedb does not match the balance in SQL DB=%v (last block in SQL = %v",account_id,balance.String(),sql_account_to_balance.String(),last_block_num))
				break;
			  }
			}
			if (from_id==account_id) {
			  if sql_account_from_balance.Cmp(balance)!=0 {
				set_verif_error(valtr_id,fmt.Sprintf("(from) account_id=%v with balance=%v in statedb does not match the balance in SQL DB=%v (last block in SQL = %v",account_id,balance.String(),sql_account_from_balance.String(),last_block_num))
				break;
			  }
			}
		}
	}
	status_write_back <- true
}
func sql_verify_sql_accounts_against_blockchain(block_num Block_num_t,accounts map[string]state.DumpAccount) {

	var query string
	query=`
		SELECT 
			valtr_id,
			from_account.address AS from_address,
			to_account.address AS to_address,
			from_balance::string,to_balance::string 
		LEFT JOIN account AS from_account ON value_transfer.from_id=from_account.account_id 
		LEFT JOIN account AS   to_account ON value_transfer.to_id  =to_account.account_id
		FROM value_transfer 
		WHERE block_num=$1"
`
	rows,err:=db.Query(query,block_num);
	defer rows.Close()
	if (err!=nil) {
		utils.Fatalf("failed to execute query: %v, block_num=%v, error=%v",query,block_num,err)
	}
	ethbot_instance.verification.Num_accounts=-1
	for rows.Next() {

		var valtr_id int64;
		err:=rows.Scan(&valtr_id)
		if (err!=nil) {
			utils.Fatalf("failed to get 'valtr_id' in sql_verify_sql_accounts_against_blockchain()");
		}

		var from_address string;
		err=rows.Scan(&from_address);
		if (err!=nil) {
			utils.Fatalf("failed to get 'from_address' in sql_verify_sql_accounts_against_blockchain()");
		}

		var to_address string;
		err=rows.Scan(&to_address);
		if (err!=nil) {
			utils.Fatalf("failed to get 'to_address' in sql_verify_sql_accounts_against_blockchain()");
		}

		var sql_account_from_balance_str string
		err=rows.Scan(&sql_account_from_balance_str)
		if (err!=nil) {
			utils.Fatalf("failed to get balance in sql_verify_sql_accounts_against_blockchain()");
		}

		var sql_account_to_balance_str string
		err=rows.Scan(&sql_account_to_balance_str)
		if (err!=nil) {
			utils.Fatalf("failed to get balance in sql_verify_sql_accounts_against_blockchain()");
		}

		var sql_account_from_balance,sql_account_to_balance big.Int
		sql_account_from_balance.SetString(sql_account_from_balance_str,10)
		sql_account_to_balance.SetString(sql_account_to_balance_str,10)

		var balance big.Int
		var account_dump state.DumpAccount
		var exists bool
		account_dump,exists=accounts[from_address];
		if !exists {
			set_verif_error(valtr_id,fmt.Sprintf("'from_account' does not exist, block_num=%v, from_account=%v",block_num,from_address))
			return
		} else {
			balance.SetString(account_dump.Balance,10)
			if sql_account_from_balance.Cmp(&balance)!=0 {
				set_verif_error(valtr_id,fmt.Sprintf("'from_account' balance mismatch. block_num=%v, correct_balance=%v, wrong balance=%v",block_num,account_dump.Balance,sql_account_from_balance));
				return;
			}
		}
		account_dump,exists=accounts[to_address];
		if !exists {
			set_verif_error(valtr_id,fmt.Sprintf("'to_account' does not exist. block_num=%v, to_account=%v",block_num,to_address));
			return
		} else {
			balance.SetString(account_dump.Balance,10)
			if sql_account_to_balance.Cmp(&balance)!=0 {
				set_verif_error(valtr_id,fmt.Sprintf("'to_account' balance mismatch, block_num=%v","correct_balance=%v,wrong balance=%v",block_num,account_dump.Balance,sql_account_to_balance));
				return;
			}
		}
		ethbot_instance.verification.Num_processed++
	}
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
func sql_get_account_history(account_id Account_id_t, block_num Block_num_t) []*Acct_history_t {

	var query string
	query=`SELECT block_id,block_num,valtr_id,from_id,to_id,from_balance::text,to_balance::text,value::text from value_transfer WHERE ((from_id=$1) OR (to_id=$1)) AND (block_num<=$2) ORDER BY block_num,valtr_id`

	var history []*Acct_history_t =make([]*Acct_history_t,0)
	rows,err:=db.Query(query,int(account_id),int(block_num));
	defer rows.Close()
	if (err!=nil) {
		utils.Fatalf("EthBot: failed to execute query: %v, account_id=%v, block_num=%v, error=%v",query,account_id,block_num,err)
	}
	for rows.Next() {
		h:=&Acct_history_t {}
		var (
			block_id		int
			block_num		int
			valtr_id		int64
			from_id			int
			to_id			int
		)

		err:=rows.Scan(&block_id,&block_num,&valtr_id,&from_id,&to_id,&h.from_balance,&h.to_balance,&h.value)
		if (err!=nil) {
			if err==sql.ErrNoRows {
				return history;
			} else {
				log.Error(fmt.Sprintf("EthBot: scan failed at sql_get_account_history(): %v",err))
				os.Exit(2)
			}
		}
		h.block_id=Block_id_t(block_id)
		h.block_num=Block_num_t(block_num)
		h.valtr_id=valtr_id
		h.from_id=Account_id_t(from_id)
		h.to_id=Account_id_t(to_id)
		history=append(history,h)
	}
	return history
}
func rpc_get_account_value_transfers(acct_addr string) string {
	var value_transfers string=""
	var json_output string
	addr:=common.HexToAddress(acct_addr)
	account_id:=lookup_account(&addr)
	var query string
	query=`
		SELECT 
			v.valtr_id,
			v.block_num,
			v.from_id,
			v.to_id,
			src.address,
			dst.address,
			v.from_balance::text,
			v.to_balance::text,
			value::text,
			t.tx_hash
		FROM 
			value_transfer AS v
		LEFT JOIN account AS src ON v.from_id=src.account_id
		LEFT JOIN account AS dst ON v.to_id=src.account_id
		LEFT JOIN transaction AS t ON v.tx_id=t.tx_id
		WHERE 
			(
				(from_id=$1) OR (to_id=$1)
			)  
			ORDER BY block_num,valtr_id
		`
	rows,err:=db.Query(query,int(account_id));
	defer rows.Close()
	if (err!=nil) {
		utils.Fatalf("failed to execute query: %v, account_id=%v, (%v), error=%v",query,account_id,acct_addr,err)
	}
	var jvt Json_value_transfer_t
	comma:=""
	for rows.Next() {
		err:=rows.Scan(&jvt.Valtr_id,&jvt.Block_num,&jvt.From_id,&jvt.To_id,&jvt.From_addr,&jvt.To_addr,&jvt.From_balance,&jvt.To_balance,&jvt.Value,&jvt.Tx_hash)
		if (err!=nil) {
			if err==sql.ErrNoRows {
				break;
			} else {
				log.Error(fmt.Sprintf("EthBot: scan failed at rpc_get_account_value_transfers(): %v",err))
				os.Exit(2)
			}
			if len(value_transfers)>0 {
				comma=","
			}
			jvt_tmp,err:=json.Marshal(jvt)
			if (err!=nil) {
				utils.Fatalf("EthBot: error in unmarshaling. jvt=%v",jvt)
			}
			value_transfers=value_transfers+comma+string(jvt_tmp)
		}
	}
	json_output=`{"result":0,"error":"","value_transfers":[` + value_transfers + `]`;
	return json_output
}
func sql_update_main_stats(last_block Block_num_t) bool {
	var block_num_upper int = int(last_block)
	block_num_lower:=last_block-1000
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

	query="SELECT avg(num_tx)::text as avg_tx FROM block WHERE block_num>=$1 AND block_num<=$2"
	row = db.QueryRow(query,block_num_lower,last_block)
	var avg_num_tx_aux sql.NullString
	var avg_num_tx string="0"
	err=row.Scan(&avg_num_tx_aux);
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error in avg(num_tx) query at Scan() : %v",err))
		utils.Fatalf("error: %v",err);
	}
	if avg_num_tx_aux.Valid {
		avg_num_tx=avg_num_tx_aux.String
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

	query="SELECT abs(last_balance)::text FROM account where account_id=$1"
	row = db.QueryRow(query,NONEXISTENT_ADDRESS_ACCOUNT_ID)
	var supply_str string="0"
	err=row.Scan(&supply_str);
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error in getting supply at Scan() : %v",err))
		utils.Fatalf("error: %v",err);
	}

	query="SELECT avg(difficulty)::text as difficulty FROM block WHERE block_num>=$1 AND block_num<=$2"
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

	query=`
		UPDATE mainstats SET
			hash_rate=$1,
			block_time=$2,
			tx_per_block=$3,
			gas_price=$4,
			tx_cost=$5,
			supply=$6,
			difficulty=$7,
			last_block=$8
		`
	_,err=db.Exec(query,hashrate_str,blocktime_str,avg_num_tx,gas_price_str,tx_cost_str,supply_str,difficulty_str,last_block);
	if (err!=nil) {
		utils.Fatalf("Update mainstats failed failed: %v",err);
	}


	return true
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
func sql_get_block_VTs(block_num Block_num_t) {
	var query string

	query="select count(*) from value_transfer where block_num=$1"
	row := db.QueryRow(query,block_num)
	var count sql.NullInt64
	err:=row.Scan(&count);
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error in sql_get_block_VTs() at Scan() : %v",err))
		utils.Fatalf("error: %v",err);
	}
	if (count.Valid) {
		log.Info(fmt.Sprintf("num value transfers: %v",count.Int64))
	} else {
		log.Info("Null count received in get_block_VTs()")
	}
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
	var prev_i=0
	prev_balance:=get_entry_balance(history[prev_i],account_id)
	accum_value:=big.NewInt(0)
	for i<end {
		prev_entry:=history[prev_i]
		entry:=history[i]
		stored_balance:=get_entry_balance(prev_entry,account_id)
		accumulated_balance:=big.NewInt(0)
		accumulated_balance.Add(prev_balance,accum_value)
		if stored_balance.Cmp(accumulated_balance)!=0 {
				// fix errors
				fix_balance_in_vt(account_id,entry,accumulated_balance)
		}
		value:=big.NewInt(0)
		if (entry.from_id!=entry.to_id) { // not a selftransfer
			value.SetString(entry.value,10)
			if (account_id==entry.from_id) { // if it is a withdrawal, negate the `value`
				value.Neg(value)
			}
		}
		accum_value.Add(accum_value,value)
		i++
		prev_i++
	}
	set_verif_error(0,fmt.Sprintf("Error in verification algorithm"))
	return false	// this is never executed
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
	_,err:=db.Exec(query,correct_balance.String(),entry.valtr_id);
	if (err!=nil) {
		utils.Fatalf("Update value_transfer to correct balance failedfailed: %v",err);
	}
}
func sql_get_last_block_num() Block_num_t {

	var query string
	//query="SELECT block_num FROM block ORDER BY block_num DESC LIMIT 1"; // this should not be used as it is not guarantee completness on process_block() abortion
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
	_,err:=db.Exec("UPDATE last_block SET block_num=$1",bnum)
	if (err!=nil) {
		utils.Fatalf("sql_set_last_block_num() failed: %v",err);
	}
}
func sql_update_block_stats(block_id Block_id_t,value_transferred *big.Int,miner_reward *big.Int,num_TXs int,num_VTs int) {

	_,err:=db.Exec("UPDATE block SET val_transferred=$2,miner_reward=$3,num_tx=$4,num_vt=$5 WHERE block_id=$1",block_id,value_transferred.String(),miner_reward.String(),num_TXs,num_VTs)
	if (err!=nil) {
		utils.Fatalf("sql_set_last_block_num() failed: %v",err);
	}
}
func sql_fix_last_balances(accounts *map[common.Address]*big.Int) bool {
	var query string;
	for addr,dump_balance:=range *accounts {
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
func sql_verify_last_balances(accounts *map[common.Address]*big.Int) bool {
	var query string;
	var balance_str string
	var retval bool=true
	balance:=big.NewInt(0)
	for addr,dump_balance:=range *accounts {
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
