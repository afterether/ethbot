package main

import (
	"os"
	"fmt"
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
var db *sql.DB
func init_postgres() {
	conn_str:="user='"+os.Getenv("ETHBOT_USERNAME")+"' dbname='"+os.Getenv("ETHBOT_DATABASE")+"' password='"+os.Getenv("ETHBOT_PASSWORD")+"' host='"+os.Getenv("ETHBOT_HOST")+"'";
	var err error
	db,err=sql.Open("postgres",conn_str);
	if (err!=nil) {
		log.Error("Can't connect to PostgreSQL database. Check that you have set ETHBOT_USERNAME,ETHBOT_PASSWORD,ETHBOT_DATABASE and ETHBOT_HOST environment variables");
	} else {
	}
	row := db.QueryRow("SELECT now()")
	var now string
	err=row.Scan(&now);
	if (err!=nil) {
		log.Error("Can't connect to PostgreSQL database. Check that you have set ETHBOT_USERNAME,ETHBOT_PASSWORD,ETHBOT_DATABASE and ETHBOT_HOST environment variables");
		utils.Fatalf("error: %v",err);
	} else {
		log.Info("Connected to Postgres successfuly");
	}
}
func lookup_block_by_hash(hash string) Block_id_t {
	var query string
	query="SELECT block_id FROM block WHERE block_hash=$1"
	var block_id Block_id_t
	err:=db.QueryRow(query,hash).Scan(&block_id);
	if (err!=nil) {
		if (err==sql.ErrNoRows) {
			return -1;
		} else {
			utils.Fatalf("Error looking up block by hash: %v",err);
		}
	}
	return block_id
}
func lookup_account(addr_str string) (account_id Account_id_t) {
	query:="SELECT account_id FROM account WHERE address=$1"
	row:=db.QueryRow(query,addr_str);
	err:=row.Scan(&account_id);
	if (err==sql.ErrNoRows) {
		return 0
	} else {
		return account_id
	}
}
func block2sql(chain *core.BlockChain,block *types.Block) (Block_id_t,error) {
	var total_dif *big.Int
	block_num:=Block_num_t(block.NumberU64())
	if block_num==0 {
		total_dif=block.Header().Difficulty;
	} else {
		total_dif=chain.GetTdByHash(block.Hash());
	}
	sql_delete_block(int(block_num),block.Uncles());	// remove all the previous data for this block

	block_id,err:=sql_insert_block(block.Header(),total_dif);
	if (err!=nil) {
		return 0,err;
	}

	// Inserts not only the main block, but uncles too, uncles have the same block_num, but different uncle_pos. main block has uncle_pos=0
	uncles:=block.Uncles();
	for i,uncle_hdr:=range uncles {
		uncle_pos:=i+1
		_,err:=sql_insert_uncle(uncle_hdr,total_dif,uncle_pos);
		if (err!=nil) {
			return Block_id_t(0),err;
		}
	}
	return block_id,nil;
}
func sql_insert_block(hdr *types.Header,total_dif *big.Int) (Block_id_t,error) {
	var err error
	var miner_id Account_id_t
	var query string

	block_hash:=hdr.Hash().String();
	block_num:=Block_num_t(hdr.Number.Uint64())
	parent_id:=lookup_block_by_hash(hdr.Hash().String());
	miner_id,err=sql_insert_account(&hdr.Coinbase);
	if (err!=nil) {
		return 0,err
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
			rcpt_hash,
			mix_hash,
			bloom,
			extra
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17) 
		RETURNING block_id`;
	var block_id Block_id_t
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
		hdr.UncleHash.String(),
		hdr.Root.String(),
		hdr.TxHash.String(),
		hdr.ReceiptHash.String(),
		hdr.MixDigest.String(),
		hdr.Bloom.Bytes(),
		extra).Scan(&block_id);
	if (err!=nil) {
		utils.Fatalf("Error inserting into `blocks` table: %v",err);
	}
	return block_id,nil;
}
func sql_insert_uncle(hdr *types.Header,total_dif *big.Int,uncle_pos int) (Block_id_t,error) {
	var err error
	var miner_id Account_id_t
	var query string

	block_hash:=hdr.Hash().String();
	block_num:=Block_num_t(hdr.Number.Uint64())
	parent_id:=lookup_block_by_hash(hdr.Hash().String());
	miner_id,err=sql_insert_account(&hdr.Coinbase);
	if (err!=nil) {
		return 0,err
	}
	total_dif_str:=total_dif.String();
	time_str:=hdr.Time.String();
	extra:=hdr.Extra

	nonce_str:=fmt.Sprintf("%d",hdr.Nonce.Uint64());

	query=`
		INSERT INTO uncle(
			parent_id,
			block_num,
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
	var uncle_id Block_id_t
	err=db.QueryRow(query,
		parent_id,
		block_num,
		time_str,
		miner_id,
		uncle_pos,
		hdr.Difficulty.String(),
		total_dif_str,
		hdr.GasLimit.String(),
		hdr.GasUsed.String(),
		nonce_str,
		block_hash,
		hdr.UncleHash.String(),
		hdr.Root.String(),
		hdr.TxHash.String(),
		hdr.ReceiptHash.String(),
		hdr.MixDigest.String(),
		hdr.Bloom.Bytes(),
		extra).Scan(&uncle_id);
	if (err!=nil) {
		utils.Fatalf("Error inserting into `blocks` table: %v",err);
	}
	return uncle_id,nil;
}
func sql_insert_transaction(msg *types.Message,tx *types.Transaction,block_id Block_id_t,block_num Block_num_t,tx_index int,timestamp int) (int64,error) {
	from:=msg.From()
	from_id,err:=sql_insert_account(&from)
	if (err!=nil) {
		return 0,err
	}
	to:=msg.To();
	var to_id Account_id_t
	if (to==nil) {
		to_id=CONTRACT_CREATION_ACCOUNT_ID
	} else {
		to_id,err=sql_insert_account(to)
		if (err!=nil) {
			return 0,err
		}
	}
	var transaction_id int64;
	var query string;
	v,r,s:=tx.RawSignatureValues();
	payload:=tx.Data();
	query=`
		INSERT INTO transaction(
			from_id,
			to_id,
			gas,
			tx_value,
			gas_price,
			nonce,
			block_id,
			block_num,
			tx_index,
			tx_ts,
			v,
			r,
			s,
			tx_hash,
			payload
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) 
		RETURNING tx_id`;
	var nonce int=int(msg.Nonce())
	err=db.QueryRow(query,
		from_id,
		to_id,
		msg.Gas().String(),
		msg.Value().String(),
		msg.GasPrice().String(),
		nonce,
		block_id,
		block_num,
		tx_index,
		timestamp,
		v.String(),
		r.String(),
		s.String(),
		tx.Hash().String(),
		payload).Scan(&transaction_id);
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
func get_previous_balance(block_num Block_num_t,account_id Account_id_t) *big.Int {

	var query string
	var aux_from,aux_to Account_id_t
	var aux_from_bal_str,aux_to_bal_str string

	balance:=big.NewInt(0)

	query=sql_query_get_balance()
	row:=db.QueryRow(query,block_num,account_id);
	err:=row.Scan(&aux_from,&aux_to,&aux_from_bal_str,&aux_to_bal_str)
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
					utils.Fatalf(fmt.Sprintf("unknown use case 1 at get_previous_balance() for account_id=%v, block_num=%v ",account_id,block_num))
			}
		} else {
			if aux_from==account_id {
				balance.SetString(aux_from_bal_str,10)
			} else {
				if aux_to==account_id {
					balance.SetString(aux_to_bal_str,10)
				} else {
					utils.Fatalf(fmt.Sprintf("unknown use case 2 at get_previous_balance() for account_id=%v, block_num=%v ",account_id,block_num))
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
		from_id,err=sql_insert_account(transf.src)
		if (err!=nil) {
			return 0,err
		}
	}
	if transf.dst==nil {
		to_id=NONEXISTENT_ADDRESS_ACCOUNT_ID
	} else {
		to_id,err=sql_insert_account(transf.dst)
		if (err!=nil) {
			return 0,err
		}
	}
	from_balance:=get_previous_balance(transf.block_num,from_id)
	to_balance := big.NewInt(0)
	if (to_id==from_id) { // self transfer 
		to_balance.Set(from_balance)
	} else {
		to_balance=get_previous_balance(transf.block_num,to_id)
		neg_value:=big.NewInt(0)
		neg_value.Neg(transf.value)
		from_balance.Add(from_balance,neg_value)
		to_balance.Add(to_balance,transf.value)
	}

	value_str:=transf.value.String();
	from_balance_str:=from_balance.String()
	to_balance_str:=to_balance.String()
	block_id:=int(transf.block_id)
	query="INSERT INTO value_transfer(tx_id,block_id,block_num,from_id,to_id,value,from_balance,to_balance,kind) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING valtr_id"
	var row *sql.Row
	if (transaction_id==-1) { // this is the case when a value_transfer record is not linked to a transaction record
		var null_var sql.NullInt64
		transaction_id_param:=null_var
		row=db.QueryRow(query,transaction_id_param,block_id,transf.block_num,from_id,to_id,value_str,from_balance_str,to_balance_str,transf.kind)
	} else {
		transaction_id_param:=transaction_id
		row=db.QueryRow(query,transaction_id_param,block_id,transf.block_num,from_id,to_id,value_str,from_balance_str,to_balance_str,transf.kind)
	}
	err=row.Scan(&valtr_id)
	if (err!=nil) {
		utils.Fatalf("Can't insert value_transfer, error=",err);
	}
	return valtr_id,nil;
}
func sql_insert_account(addr *common.Address) (Account_id_t,error) {
	var query string
	var account_id Account_id_t;
	query="SELECT account_id FROM account WHERE address=$1";
	addr_str:=get_hex_addr(addr);
	row:=db.QueryRow(query,addr_str);
	err:=row.Scan(&account_id)

	if (err==sql.ErrNoRows) {
		query="INSERT INTO account(address) VALUES($1) RETURNING account_id";
		err:=db.QueryRow(query,addr_str).Scan(&account_id);
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
				valtr_id=0
				last_block_num=block_num
				from_id=account_id
				to_id=account_id
				sql_account_from_balance_str="0"
				sql_account_to_balance_str="0"
			} else {
				utils.Fatalf("failed to execute query: %v, error=%v",query,err)
			}
		}

		sql_account_from_balance.SetString(sql_account_from_balance_str,10);
		sql_account_to_balance.SetString(sql_account_to_balance_str,10);

		if from_id==to_id {	// self transfer
			if sql_account_to_balance.Cmp(balance)!=0 {
				set_verif_error(valtr_id,fmt.Sprintf("(selftransfer) account_id=%v for block_num<=%v expected balance=%v, balance in SQL DB=%v",account_id,last_block_num,balance.String(),sql_account_to_balance.String()))
				break;
			}
		} else {
			if (to_id==account_id) {
			  if sql_account_to_balance.Cmp(balance)!=0 {
				set_verif_error(valtr_id,fmt.Sprintf("(to) account_id=%v for block_num<=%v expected balance=%v, balance in SQL DB=%v",account_id,last_block_num,balance.String(),sql_account_to_balance.String()))
				break;
			  }
			}
			if (from_id==account_id) {
			  if sql_account_from_balance.Cmp(balance)!=0 {
				set_verif_error(valtr_id,fmt.Sprintf("(from) account_id=%v for block_num<=%v expected balance=%v, balance in SQL DB=%v",account_id,last_block_num,balance.String(),sql_account_from_balance.String()))
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
	if (err!=nil) {
		utils.Fatalf("failed to execute query: %v, block_num=%v, error=%v",query,block_num,err)
	}
	ethbot_instance.verification.Num_accounts=-1
	defer rows.Close()
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
		block_hash:=uncle_hdr.Hash().String();
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
	if (err!=nil) {
		utils.Fatalf("failed to execute query: %v, account_id=%v, block_num=%v, error=%v",query,account_id,block_num,err)
	}
	defer rows.Close()
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
				log.Error(fmt.Sprintf("Scan failed at sql_get_account_history(): %v",err))
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
