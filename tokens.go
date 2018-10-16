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
	"bytes"
	"encoding/hex"
	"errors"
	"time"
	"runtime"
	"strconv"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
    "github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/robertkrimen/otto"
)
func is_ERC20_token_transfer(signature []byte) bool {
	if 0==bytes.Compare(signature,erc20_transfer_event_signature) {
		return true
	}
	if 0==bytes.Compare(signature,erc20_approval_event_signature) {
		return true
	}
	return false
}
func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
func event2transfer_record(event *Event_t) *Token_transf_t {

	if event==nil {
		return nil
	}
	if len(event.Topics)==0 {
		return nil
	}
	kind:=TOKENOP_UNKNOWN
	if 0==bytes.Compare(event.Topics[0].Bytes(),erc20_transfer_event_signature) {
		kind=TOKENOP_TRANSFER
	}
	if 0==bytes.Compare(event.Topics[0].Bytes(),erc20_approval_event_signature) {
		kind=TOKENOP_APPROVAL
	}
	if kind==TOKENOP_UNKNOWN {
		return nil
	}
	if len(event.Topics)<3 {
		return nil		// not enough indexed variables for a transfer event
	}
	log.Info(fmt.Sprintf("data hex=%v",event.Data))
	transf:=&Token_transf_t {
		From:common.BytesToAddress(event.Topics[1].Bytes()),
		To: common.BytesToAddress(event.Topics[2].Bytes()),
		Value: big.NewInt(0).SetBytes(event.Data),
		Kind: kind,
	}
	return transf
}
func extract_transfer_record(event *Event_t,input_token_op int,input_from,input_to *common.Address,input_value *big.Int) (*Token_transf_t,bool,string) { // returns: tranfer record, non_compliant,non_compliance-err
	var non_compliant bool
	var non_compliance_err string
	if len(event.Topics)==0 {
		return nil,false,""
	}
	kind:=TOKENOP_UNKNOWN
	if 0==bytes.Compare(event.Topics[0].Bytes(),erc20_transfer_event_signature) {
		kind=TOKENOP_TRANSFER
	}
	if 0==bytes.Compare(event.Topics[0].Bytes(),erc20_approval_event_signature) {
		kind=TOKENOP_APPROVAL
	}
	if kind==TOKENOP_UNKNOWN {
		return nil,false,""
	}
	if len(event.Topics)<3 {
		return nil,false,""		// not enough indexed variables for a transfer event
	}

	transf:=&Token_transf_t {
		From:common.BytesToAddress(event.Topics[1].Bytes()),
		To: common.BytesToAddress(event.Topics[2].Bytes()),
		Value: big.NewInt(0).SetBytes(event.Data),
		Kind: kind,
	}
	if input_token_op!=0 {	// token operation is known and supported
		log.Info(fmt.Sprintf("input_token_op=%v",input_token_op))
		if (*input_from)!=transf.From {
			non_compliant=true
			non_compliance_err=`Input's From doesn't match Event's From`
			transf.From.SetBytes(input_from.Bytes())
		}
		if transf.To!=(*input_to) {
			non_compliant=true
			non_compliance_err=`Input's To doesn't match Event's To`
		}
		if input_value.Cmp(transf.Value)!=0 {
			if len(event.Data)==0 {		// event's data wasn't indexed
				transf.Value.Set(input_value)
				non_compliance_err=`Event's value field has 'indexed' attribute`
			} else {
				if transf.Value.Cmp(zero)==0 {	// event has 0 in value field
					transf.Value.Set(input_value)
					non_compliance_err=`Input's Value doesn't match Event's Value`
				} else {
					non_compliance_err=`Input's Value doesn't match Event's Value`
				}
			}
			non_compliant=true
		}
	}
	return transf,non_compliant,non_compliance_err
}
func (bot *EthBot_t) export_block_tokens(ethereum *eth.Ethereum, block *types.Block) error {
	var new_tokens int=0;

	tok_VTs:=int(0)
	blockchain := ethereum.BlockChain()
	bconf:=blockchain.Config()
	block_num:=Block_num_t(block.NumberU64())
	sql_delete_tokens_for_block(block_num)
	prepare_token_processing(block_num)
	timestamp:=int(block.Header().Time.Int64())
    if block_num == 0 {
		mark_tokens_as_processed(0)
		return nil;	// There can't be any token on the genesis block
    }

    statedb, err := blockchain.StateAt(block.Root())
    if err != nil {
		log.Error("EthBot: can't get StateAt()","block_num",block.NumberU64())
        return err
    }

    if bconf.DAOForkSupport && bconf.DAOForkBlock != nil && bconf.DAOForkBlock.Cmp(block.Number()) == 0 {
        misc.ApplyDAOHardFork(statedb)
    }
	zero:=big.NewInt(0)
	new_tokens=0;

	var query string
	query=`
		SELECT vt.valtr_id,vt.kind::int,vt.value::text,input,output,vt.tx_id,vt.from_id,vt.to_id,src.address AS src_address,dst.address AS dst_address FROM value_transfer AS vt
		LEFT JOIN account AS src on vt.from_id=src.account_id
		LEFT JOIN account AS dst ON vt.to_id=dst.account_id,
		vt_extras AS vte
		WHERE
			vt.valtr_id=vte.valtr_id AND
			vt.block_num=$1 AND
			octet_length(error)=0 AND
			(
				(vt.kind IN (5,6)) OR
				((vt.kind=2) AND (dst.owner_id>0))
			)
		ORDER BY bnumvt
	`
	rows,err:=db.Query(query,block_num)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Error querying valtransfers for block %v : %v",block_num,err))
	}
	defer rows.Close()
	for rows.Next() {
		var (
			valtr_id			int64
			tx_id				int64
			kind				int
			value				string
			input				[]byte
			output				[]byte
			from_id				Account_id_t
			contract_id			Account_id_t
			from_address		string
			contract_address	string
		)

		err=rows.Scan(&valtr_id,&kind,&value,&input,&output,&tx_id,&from_id,&contract_id,&from_address,&contract_address)
		if err!=nil {
			utils.Fatalf(fmt.Sprintf("Scan() of value transfers int token processing failed: %v",err))
		}
		event_logs:=sql_get_events(valtr_id)
		var owner_balance *big.Int=big.NewInt(0)
		var contract_balance *big.Int=big.NewInt(0)
		var owner_has_holdings,contract_has_holdings bool
		log.Info(fmt.Sprintf("valtr_id: %v; tx_id=%v, kind=%v, contract=%v (%v) %v->%v for %v, %v events",valtr_id,tx_id,kind,contract_address,contract_id,from_address,contract_address,value,len(event_logs)))

		contract_addr:=common.HexToAddress(contract_address)
		from_addr:=common.HexToAddress(from_address)
		if kind==VALTRANSF_CONTRACT_CREATION {
				log.Info(fmt.Sprintf("TX for contract creation, contract_addr=%v, id=%v",contract_address,contract_id))
				tok:=Token_t {
					contract_addr:&contract_addr,
					tx_id: tx_id,
				}
				token_info:=Token_info_t{}
				token_info.Contract_id=contract_id
				token_info.block_created=block_num
				if bot.extract_token_info(blockchain,block,statedb,bconf,&from_addr,&contract_addr,event_logs,&tok,&token_info) {
					contract_id=sql_insert_token(&tok,&token_info)
					owner_balance=get_ERC20_token_balance_from_EVM(blockchain,statedb, block ,&contract_addr,&from_addr)
					contract_balance=get_ERC20_token_balance_from_EVM(blockchain,statedb, block ,&contract_addr,&contract_addr)
					new_tokens++
				} else {
					sql_insert_token_info(&token_info)		// store info about non-compliancy
				}
		} else {
		}
		var tokop_id int64
		token_op,_,iface,values:=decode_token_input(input)
		log.Info(fmt.Sprintf("contract %v: token_op=%v num_logs=%v",hex.EncodeToString(contract_addr.Bytes()),token_op,len(event_logs)))
		for _,event:=range event_logs {
			// following code only processes ERC20 token transfers
			token_op_param:=token_op
			var input_from common.Address=common.Address{}
			var input_to common.Address=common.Address{}
			var input_value *big.Int=nil
			if token_op==TOKENOP_TRANSFER_FROM {
				if len(values)<3 {
					utils.Fatalf("transferFrom() has less than 3 parameters")
					continue		// invalid input
				}
				input_from.SetBytes((values[0].(common.Address)).Bytes())
				input_to.SetBytes((values[1].(common.Address)).Bytes())
				input_value=values[2].(*big.Int)
			}
			if (token_op==TOKENOP_TRANSFER) || (token_op==TOKENOP_APPROVAL) {
				if len(values)<2 {
					log.Info(fmt.Sprintf("contract input=%v",hex.EncodeToString(input)))
					log.Info(fmt.Sprintf("len(values)=%v, values=%v",len(values),values))
					utils.Fatalf("transfer() / approve() has leass than 2 parameters")
					continue
				}
				input_from.SetBytes(from_addr.Bytes())
				log.Info(fmt.Sprintf("new from=%v",hex.EncodeToString(input_from.Bytes())))
				input_to.SetBytes((values[0].(common.Address)).Bytes())
				input_value=values[1].(*big.Int)
			}
			transfer_obj,non_compliant,non_compliance_err:=extract_transfer_record(event,token_op,&input_from,&input_to,input_value)
			if transfer_obj==nil {
				continue
			} else {
				transfer_obj.dump()
				if !contract_has_holdings{
					if contract_addr==transfer_obj.To {
						contract_has_holdings=true
					}
				}
				if !owner_has_holdings {
					if from_addr==transfer_obj.To {
						owner_has_holdings=true
					}
				}
			}
			if token_op_param==TOKENOP_APPROVAL {
				if transfer_obj==nil {
					utils.Fatalf("Exit on error")
				}
				transfer_obj.dump()
				tokop_id=sql_insert_token_approval(contract_id,values,event,transfer_obj,&from_addr,tx_id,block_num,timestamp,non_compliant,non_compliance_err)
			} else {
				if token_op_param==0 {
					if transfer_obj.Kind!=TOKENOP_UNKNOWN {
						token_op_param=transfer_obj.Kind
					}
				}
				if transfer_obj==nil {
					utils.Fatalf("transfer_obj is nil")
				}
				transfer_obj.dump()
				tokop_id=sql_insert_token_operation(token_op_param,iface,values,contract_id,event,transfer_obj,block_num,timestamp,tx_id,non_compliant,non_compliance_err)
			}
			if tokop_id!=0 {
				tok_VTs++
			}
		}
		if kind==VALTRANSF_CONTRACT_CREATION {
			if (!contract_has_holdings) && (contract_balance.Cmp(zero)!=0) {	// non compliant token found, contract has tokens, but Transfer event was not emitted
					token_op=TOKENOP_TRANSFER
					transf_obj:=Token_transf_t {
						From: common.Address{},
						To: contract_addr,
						Value: contract_balance,
						Kind: token_op,
						Non_compliant: true,
						Non_compliance_str: "Transfer event wasn't emitted for this address",
					}
					tokop_id=sql_insert_token_operation(token_op,TOKIFACE_UNKNOWN,values,contract_id,nil,&transf_obj,block_num,timestamp,tx_id,false,"")
					if tokop_id!=0 {
						tok_VTs++
					}
			}
			if (!owner_has_holdings) && (owner_balance.Cmp(zero)!=0) {	// non compliant token found, owner has tokens, but Transfer event was not emitted
					tokop_id=TOKENOP_TRANSFER
					transf_obj:=Token_transf_t {
						From: common.Address{},
						To: from_addr,
						Value: contract_balance,
						Kind: token_op,
						Non_compliant: true,
						Non_compliance_str: "Transfer event wasn't emitted for this address",
					}
					tokop_id=sql_insert_token_operation(token_op,TOKIFACE_UNKNOWN,values,contract_id,nil,&transf_obj,block_num,timestamp,tx_id,false,"")
					if tokop_id!=0 {
						tok_VTs++
					}
			}
		}
	}
	mark_tokens_as_processed(block_num)
	log.Info(fmt.Sprintf("EthBot: processed tokens for %v",block_num),"new tok",new_tokens,"TokVTs",tok_VTs)
	return nil
}
func (bot *EthBot_t) adjust_token_starting_block_num(starting_block Block_num_t) Block_num_t {
	if (starting_block==-1) {
		starting_block = sql_get_last_token_block_num()
		if (starting_block==-1) {
			return 0;		// SQL database is emtpy, return genesis block (block_num=0)
		} else {
			return starting_block+1
		}
	}
	return starting_block
}
func (bot *EthBot_t) token_export_start(starting_block Block_num_t,ending_block Block_num_t) error {
	if (bot.token_export.In_progress) {
		log.Error("EthBot: token export already in progress");
		return errors.New("Token export already in progress");
	}
	starting_block=bot.adjust_token_starting_block_num(starting_block)
	bot.token_export.In_progress=true
	bot.token_export.Range_export=false
	bot.token_export.Starting_block=starting_block
	bot.token_export.Ending_block=ending_block
	bot.token_export.Listening_mode=false
	bot.token_export.Direction=1
	if bot.token_export.Ending_block!=-1 {
		if (bot.token_export.Starting_block<bot.token_export.Ending_block) {
			bot.token_export.Direction=1
		} else {
			if (bot.token_export.Starting_block<bot.token_export.Ending_block) {
				bot.token_export.Direction=-1
			} else {
				bot.token_export.Direction=0
			}
		}
	} else {
		bot.token_export.Listening_mode=true
		bot.token_export.Ending_block=bot.adjust_ending_block_num(bot.token_export.Ending_block)
		log.Info(fmt.Sprintf("EthBot: will export tokens from %v to %v, after that, will enter listening mode",bot.token_export.Starting_block,bot.token_export.Ending_block))
	}
	go bot.export_tokens()

	return nil
}
func (bot *EthBot_t) token_export_stop() bool {

	if (!bot.token_export.In_progress) {
		log.Error("EthBot: token export process is not running")
		return false
	}

	bot.token_export.In_progress=false;
	return true
}
func (bot *EthBot_t) export_tokens() {
	// exporting process uses 2 functions:
	// export_tokens() which feeds the blocks through the channel bot->token_blocks
	// process_tokens() which does the actual export to SQL 
	// 
    chain := bot.ethereum.BlockChain()
    go bot.process_tokens()
	bot.token_export.Cur_block_num=bot.token_export.Starting_block
	for true {
		if !bot.token_export.In_progress { // the process was probably aborted by user
			break;
		}
		block:=chain.GetBlockByNumber(uint64(bot.token_export.Cur_block_num))
		if block!=nil {
			bot.token_export.blocks<-block
		} else {
			log.Info("EthBot: null block received from the blockchain, aborting")
			break;
		}
		tmp_num:=Block_num_t(chain.CurrentBlock().Number().Uint64())
		if (bot.token_export.Cur_block_num==tmp_num) {	// we reached the end of the blockchain, stored in the DB
			break
		}
		if (bot.token_export.Direction>0) {
			bot.token_export.Cur_block_num++
			if (bot.token_export.Cur_block_num>bot.token_export.Ending_block) {
				break
			}
		} else {
			if bot.token_export.Direction<0 {
				bot.token_export.Cur_block_num--
				if (bot.token_export.Cur_block_num<bot.token_export.Ending_block) {
					break
				}
			} else {
				break;		// if Direction is 0 it means we are processing a single block, so we exit loop
			}
		}
	}
	// at this point we processed all the blocks up to the ending_block
	// so, if user specified listening mode, enter into it and continue exporting incoming blocks
	if (bot.token_export.In_progress) {	// the user can interrupt export , so we check again
		if (bot.token_export.Listening_mode) {
		    bot.token_export.head_ch= make(chan core.ChainHeadEvent)
			bot.token_export.head_sub=chain.SubscribeChainHeadEvent(bot.token_export.head_ch)
			log.Info("Subscribed to channel for tokens")
			go bot.listen4token_blocks();
		} else {
			bot.token_export.In_progress=false
		}
	}
}
func (bot *EthBot_t) process_tokens() {
	if bot.token_process_started {	// additional security check, to do not launch this twice
		return
	}
	bot.token_process_started=true
  again:
	var block *types.Block
    block = <-bot.token_export.blocks
	if (block==nil) {
		log.Error("EthBot: null block received")
		goto again;
	}
	block_num:=Block_num_t(block.Number().Uint64())
	done:=sql_block_export_finished(block_num)
	if (!done) {	// wait for block to be fully exported by main blockchain export
		time.Sleep(500 * time.Millisecond)
		go func(blk *types.Block) {
			bot.token_export.blocks <- blk
		}(block)
		goto again
	}
	err := bot.export_block_tokens(bot.ethereum, block)
    if err != nil {
		if err==ErrAncestorNotFound {	// probably a chain split occurred, fix it
			bot.token_repair_chain_split(block)
		} else {
	       log.Error("EthBot: Unable to export tokens","block", block.Number().Uint64(),"err",err)
		}
        goto again;
    }
	sql_set_last_token_block_num(block_num)
	bot.token_export.Exported_block=block_num
	goto again
}
func (bot *EthBot_t) listen4token_blocks() {

	if (bot.token_export.listening_started) {
		return
	}
	bot.token_export.listening_started=true
    chain := bot.ethereum.BlockChain()
	log.Info("EthBot: entering listening mode for tokens, to export tokens as blocks arrive")
  again:
	evt:= <-bot.token_export.head_ch
	log.Info(fmt.Sprintf("token listen4blocks: got block %v",evt.Block.NumberU64()))
	if (evt.Block==nil) {	// this routine exits on receiving null block
		bot.token_export.listening_started=false
		return;
	}
	new_token_block_num:=Block_num_t(evt.Block.NumberU64())
	last_token_block_num:=sql_get_last_token_block_num()
	var block *types.Block
	for block_num:=(last_token_block_num+1);block_num<=new_token_block_num;block_num++ {
		block=chain.GetBlockByNumber(uint64(block_num))
		bot.token_export.blocks<-block
	}
	if (bot.token_export.In_progress==false) {
		bot.token_export.listening_started=false
		return
	}
	goto again;
}
func (bot *EthBot_t) token_repair_chain_split(block *types.Block) {
	// scan back until we find the last valid parent and then re-insert all the blocks from that number
	chain:=bot.ethereum.BlockChain()
	block_num:=Block_num_t(block.NumberU64())
	last_block_num:=block_num
	for(block_num>0) {
		parent_block_num,block_found:=lookup_block_by_hash(hex.EncodeToString(block.Header().Hash().Bytes()));
		if block_found { // we found our first valid parent
			log.Info(fmt.Sprintf("EthBot: fixing chain split from block %v",parent_block_num))
			bot.token_repair_chain_split_insert_blocks(parent_block_num,last_block_num)
			return
		}
		block_num--
		block=chain.GetBlockByNumber(uint64(block_num))
		if (block==nil) {
			utils.Fatalf("correct block wasn't found , block_num=",block_num)
		}
	}
}
func (bot *EthBot_t) token_repair_chain_split_insert_blocks(from_num Block_num_t,to_num Block_num_t) {
	chain:=bot.ethereum.BlockChain()
	i:=from_num
	for ;i<=to_num;i++ {
		block:=chain.GetBlockByNumber(uint64(i))
		if (block!=nil) {
			err:=bot.export_block_tokens(bot.ethereum,block)
			if (err!=nil) {
				log.Error(fmt.Sprintf("EthBot: repairing chain split for TOKENs: error found inserting block %v :  %v",i,err))
			}
		} else {
			log.Error(fmt.Sprintf("EthBot: can't get block number %v",i))
		}
	}
}
func (bot *EthBot_t) token_export_status(exp *TokenExport_t) *otto.Object {
	jsre:=console_obj.JSRE()
	vm:=jsre.VM()
	obj_str:=fmt.Sprintf(`({"starting_block":%d,"current_block":%d,"ending_block":%d,"direction":%d,"listening_mode":%v,"user_cancelled":%v,"in_progress":%v})`,exp.Starting_block,exp.Cur_block_num,exp.Ending_block,exp.Direction,exp.Listening_mode,exp.User_cancelled,exp.In_progress);
	object, err := vm.Object(obj_str)
	if (err!=nil) {
		utils.Fatalf("Failed to create object in Javascript VM for token export status: %v",err)
	}
	return object
}
func (this *Token_transf_t) dump() {
	if this==nil {
		log.Info(fmt.Sprintf("transfer object is nil"))
		return
	}
	log.Info(fmt.Sprintf("transfer->From: %v",hex.EncodeToString(this.From.Bytes())))
	log.Info(fmt.Sprintf("transfer->To: %v",hex.EncodeToString(this.To.Bytes())))
	log.Info(fmt.Sprintf("transfer->Value: %v",this.Value.String()))
	log.Info(fmt.Sprintf("transfer->Kind: %v",this.Kind))
}
