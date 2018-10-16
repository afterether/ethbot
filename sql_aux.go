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
	"database/sql"
	"github.com/ethereum/go-ethereum/log"
    "github.com/ethereum/go-ethereum/cmd/utils"
	_ "github.com/lib/pq"
)
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
