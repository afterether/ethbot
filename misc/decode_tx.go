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
	"bufio"
	"fmt"
	"encoding/hex"
	"strings"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)
func main() {

	var tx *types.Transaction

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	new_line_idx:=strings.IndexRune(input,'\n')
	if new_line_idx==-1 {
		fmt.Println(`The input string is not finished by '\n' character (NEWLINE)`)
		return
	}
	encoded_tx_str:=input[0:new_line_idx]
	fmt.Println("Transaction raw hex:")
	fmt.Println(encoded_tx_str)

	fmt.Println("Decoded transaction:")
	raw_tx,err := hex.DecodeString(encoded_tx_str)
    if err!=nil {
		fmt.Println(fmt.Sprintf("Can't decode HEX string: %v",err))
		return
	}
	err=rlp.DecodeBytes(raw_tx, &tx)
	if err!=nil {
		fmt.Println(fmt.Sprintf("Can't decode RLP data: %v",err))
		return
	}
	// get the signer of transaction
	chain_id:=tx.ChainId()
	signer:=types.NewEIP155Signer(chain_id)
	src_addr,err:=signer.Sender(tx)
	if err!=nil {
		fmt.Println(fmt.Sprintf("Cant get sender's address: %v",err))
		return
	}
	fmt.Println(fmt.Sprintf("From: %v",hex.EncodeToString(src_addr.Bytes())))
	dst_addr:=common.Address{}
	tx_to:=tx.To()
	if tx_to!=nil {
		dst_addr.SetBytes(tx_to.Bytes())
	}
	fmt.Println(fmt.Sprintf("Hash: %v",hex.EncodeToString(tx.Hash().Bytes())))
	fmt.Println(fmt.Sprintf("To: %v",hex.EncodeToString(dst_addr.Bytes())))
	fmt.Println(fmt.Sprintf("Gas limit: %v",tx.Gas()))
	fmt.Println(fmt.Sprintf("Gas price: %v",tx.GasPrice()))
	fmt.Println(fmt.Sprintf("Value: %v",tx.Value()))
	fmt.Println(fmt.Sprintf("Nonce: %v",tx.Nonce()))
	fmt.Println(fmt.Sprintf("Extra data: %v",hex.EncodeToString(tx.Data())))
}
