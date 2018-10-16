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
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
)
func (bot *EthBot_t ) init_pending_tx_gathering() {

	bot.newTXs_ch=make(chan core.NewTxsEvent, 2048)
	bot.newTXs_evt_sub=bot.ethereum.TxPool().SubscribeNewTxsEvent(bot.newTXs_ch)
	go bot.listen_to_pending_transactions()
}
func (bot *EthBot_t) listen_to_pending_transactions() {

	var evt core.NewTxsEvent
	for evt=range bot.newTXs_ch {
		for _,tx:=range evt.Txs {
			sql_insert_pending_tx(tx)
		}
	}
	log.Info("Listening to pending transactions is over")
}
func sql_insert_pending_tx(tx *types.Transaction) {
	var query string

	chain_id:=tx.ChainId()
	signer:=types.NewEIP155Signer(chain_id)
	from,err:=signer.Sender(tx)
	if err!=nil {
		log.Info(fmt.Sprintf("Received bad transaction, can't get signer: hash="+hex.EncodeToString(tx.Hash().Bytes())))
		return
	}
	to:=common.Address{}
	tx_to:=tx.To()
	if tx_to!=nil {
		to.SetBytes(tx_to.Bytes())
	}
	v,r,s:=tx.RawSignatureValues();
	query=`
		UPDATE pending_tx SET
			nonce=$2,
			gas_limit=$3,
			tx_value=$4,
			gas_price=$5,
			v=$6,
			r=$7,
			s=$8,
			from_address=$9,
			to_address=$10
		WHERE
			tx_hash=$1
	`
	res,err:=db.Exec(query,
		hex.EncodeToString(tx.Hash().Bytes()),
		tx.Nonce(),
		tx.Gas(),
		tx.Value().String(),
		tx.GasPrice().String(),
		v.String(),
		r.String(),
		s.String(),
		hex.EncodeToString(from.Bytes()),
		hex.EncodeToString(to.Bytes()))
	if err!=nil {
		log.Error(fmt.Sprintf("EthBot: Error inserting pending transaction %v: %v",hex.EncodeToString(tx.Hash().Bytes()),err))
		return
	}
	rows_affected,err:=res.RowsAffected()
	if err!=nil {
		log.Error(fmt.Sprintf("EthBot: Error getting rows affected for tx %v: %v",hex.EncodeToString(tx.Hash().Bytes()),err))
		return
	}
	if rows_affected>0 {
		return
	}
	query=`
		INSERT INTO pending_tx(
			nonce,
			gas_limit,
			inserted_ts,
			ptx_status,
			validated,
			tx_value,
			gas_price,
			v,
			r,
			s,
			from_address,
			to_address,
			tx_hash
		) VALUES ($1,$2,cast(extract(epoch from current_timestamp) as integer),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`
	_,err=db.Exec(query,
		tx.Nonce(),
		tx.Gas(),
		core.TxStatusQueued,
		true,
		tx.Value().String(),
		tx.GasPrice().String(),
		v.String(),
		r.String(),
		s.String(),
		hex.EncodeToString(from.Bytes()),
		hex.EncodeToString(to.Bytes()),
		hex.EncodeToString(tx.Hash().Bytes()))
	if err!=nil {
		log.Error(fmt.Sprintf("EthBot: Error inserting pending transaction %v: %v",hex.EncodeToString(tx.Hash().Bytes()),err))
		return
	}
}
