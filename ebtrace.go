package main
import (
	"os"
	"math/big"
	"fmt"
	"bytes"
	"encoding/hex"
	"errors"
	"time"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
    "github.com/ethereum/go-ethereum/cmd/utils"
)
var (
    frontierBlockReward  *big.Int = big.NewInt(5e+18)
    byzantiumBlockReward *big.Int = big.NewInt(3e+18)
    maxUncles                     = 2
	modified_balances map[Account_id_t]*big.Int=make(map[Account_id_t]*big.Int)
	previous_state		*state.StateDB
	accounts_cache		map[common.Address]Account_id_t=make(map[common.Address]Account_id_t)
	ErrAncestorNotFound		error = errors.New("Lookup of parent block by hash failed")
)

const debug_exec_time = false

func add_rewards(cfg params.ChainConfig,statedb *state.StateDB, block *types.Block,block_id Block_id_t, ) (int,*big.Int) {
	var (
	    big8  = big.NewInt(8)
	    big32 = big.NewInt(32)
	)
	total_reward:=big.NewInt(0)
	var block_num Block_num_t=Block_num_t(block.NumberU64())
    blockReward := frontierBlockReward
	var num_VTs=0
	if (params.MainnetChainConfig.ChainId.Cmp(big.NewInt(1))==0) { // Ethereum Main Net
	    if cfg.IsByzantium(block.Header().Number) {
		    blockReward = byzantiumBlockReward
		}
	}
	if (params.MainnetChainConfig.ChainId.Cmp(big.NewInt(233))==0) { // AfterEther Main Net
		if (block.NumberU64()>154865) {
			blockReward.Set(big.NewInt(2e+18))
		}
	}
	reward := new(big.Int).Set(blockReward)
    for _, uncle := range block.Uncles() {
		r := new(big.Int)
        r.Add(uncle.Number, big8)
		header:=block.Header();
        r.Sub(r, header.Number)
        r.Mul(r, blockReward)
        r.Div(r, big8)
		uncle_reward:=big.NewInt(0).Set(r)
		transfer:=&Value_transfer_t{
		        depth: 0,
				block_id: block_id,
				block_num: block_num,
		        src: nil,
		        dst: &uncle.Coinbase,
		        value: uncle_reward,
		        kind: VALTRANSF_BLOCK_REWARD,
		}
		_,val_err:=sql_insert_value_transfer(transfer,-1)
		if (val_err!=nil) {
			log.Error(fmt.Sprintf("EthBot: error inserting uncle block reward value transfer for miner %v",hex.EncodeToString(transfer.dst.Bytes())))
			os.Exit(2)
		}
		total_reward.Add(total_reward,uncle_reward)
		num_VTs++
        r.Div(blockReward, big32)
        reward.Add(reward, r)
    }


	block_reward:=big.NewInt(0).Set(reward)
	transfer:=&Value_transfer_t{
        depth: 0,
		block_id: block_id,
		block_num: block_num,
        src: nil,
        dst: &block.Header().Coinbase,
		value: block_reward,
		kind: VALTRANSF_BLOCK_REWARD,
	}
	_,val_err:=sql_insert_value_transfer(transfer,-1)
	if (val_err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error insertint block reward for miner %v",hex.EncodeToString(transfer.dst.Bytes())))
		os.Exit(2)
	}
	total_reward.Add(total_reward,block_reward)
	num_VTs++

	return num_VTs,total_reward
}
func (bot *EthBot_t) trace_block(ethereum *eth.Ethereum, block *types.Block) error {

	modified_balances=make(map[Account_id_t]*big.Int)
	blockchain := ethereum.BlockChain()
	bc_cfg:=blockchain.Config()
	block_num:=Block_num_t(block.NumberU64())
	ethbot_instance.export.Cur_block_num=Block_num_t(block.NumberU64())
    if block_num == 0 {
		statedb, err := blockchain.StateAt(block.Root())
        if err != nil {
			log.Error("EthBot: can't get StateAt()","block_num",block.Number().Uint64())
            return  err
        }
        genesis_state_dump := statedb.RawDump()
        err=process_genesis_block(blockchain,block,genesis_state_dump)
		return err;
    }

    statedb, err := blockchain.StateAt(blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1).Root() )
    if err != nil {
		log.Error("EthBot: can't get StateAt()","block_num",block.NumberU64())
        return err
    }
    previous_state, err = blockchain.StateAt(blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1).Root() )
    if err != nil {
		log.Error("EthBot: can't get StateAt() for previous_state variable","block_num",block.NumberU64())
        return err
    }
	block_transactions:=block.Transactions()
	block_id,err:=block2sql(blockchain,block,len(block_transactions));
	if (err!=nil) {
		return err;
	}
	lcfg:=&vm.LogConfig{
            DisableMemory: true,
            DisableStack: false,
            DisableStorage: true,
            FullStorage: true,
    }
	vm_cfg:=&vm.Config{
        Debug: true,
        EnableJit: false,
        ForceJit: false,
	}

    parent := blockchain.GetBlockByHash(block.ParentHash())
    if parent == nil {
		utils.Fatalf("EthBot: could not retrieve parent block for hash %v",block.ParentHash().Hex())
    }


	var (
		totalUsedGas = big.NewInt(0)
	)
	gaspool:=new(core.GasPool).AddGas(block.GasLimit())
	bconf:=blockchain.Config()
    if bconf.DAOForkSupport && bconf.DAOForkBlock != nil && bconf.DAOForkBlock.Cmp(block.Number()) == 0 {
        misc.ApplyDAOHardFork(statedb)
    }
	num_elts:=len(block.Transactions())
	log.Info("EthBot: processing","block_num",block.Number(),"transactions",num_elts)
	var receipts     types.Receipts
	var tx_value_transfers []*Value_transfer_t
	var total_VTs int=0
	per_block_value_transferred:=big.NewInt(0)

	var start_ts int64=0
	if debug_sql_exec_time {
		start_ts=time.Now().UnixNano() / int64(time.Millisecond)
	}

    for i, tx := range block_transactions {
		structLogger := vm.NewStructLogger(lcfg)
		vm_cfg.Tracer=structLogger
        statedb.Prepare(tx.Hash(), block.Hash(), i)
		var tx_err error
		var msg types.Message
		var receipt *types.Receipt
		var gas *big.Int
		var tx_invalid bool=false
		from:=common.Address{}
	    msg, tx_err = tx.AsMessage(types.MakeSigner(bconf, block.Number()))
	    if tx_err != nil {
			tx_invalid=true
		} else {
			from=msg.From()
		}
		var apply_err error
		core.Vmerr4ethbot=nil
		receipt, gas, apply_err = core.ApplyTransaction(bconf, blockchain, nil, gaspool, statedb, block.Header(), tx, totalUsedGas, *vm_cfg)
		 if apply_err != nil {
			log.Error(fmt.Sprintf("EthBot: transaction executed with error, err=%v",apply_err))
		}
		tx_value_transfers=nil
		if (tx_err==nil) {
			if core.Vmerr4ethbot!=nil {	// Vmerr4ethbot is set in state_transition.go:TransitionDb()
				// ToDo: update transaction setting error code
			} else {
			}
			var process_log_start_ts int64=0
			if debug_sql_exec_time {
				process_log_start_ts=time.Now().UnixNano() / int64(time.Millisecond)
			}
			tx_value_transfers=bot.process_StructLogs(previous_state,statedb,block,block_id,block_num,from,&msg,tx,structLogger.StructLogs(),receipt)
			if debug_sql_exec_time {
				report_time(process_log_start_ts,"process_StructLogs")
			}
		}
		gas_used:=big.NewInt(0)
		if receipt!=nil {
			gas_used.Set(receipt.GasUsed)
		}
		// now we insert miner's fee to process this transaction
        miner_tx_fee:=&Value_transfer_t {
			depth: 0,
			block_id: block_id,
			block_num: block_num,
			src: &from,
			dst: &block.Header().Coinbase,
			value: new(big.Int).Mul(gas_used, tx.GasPrice()) ,
			kind: VALTRANSF_TX_FEE,
		}
		tx_value_transfers=append(tx_value_transfers,miner_tx_fee)
		// calculate statistics
		amount_transferred:=big.NewInt(0)
		var num_VTs int = 0
		for _,vt:= range tx_value_transfers {
			num_VTs++
			amount_transferred.Add(amount_transferred,vt.value)
		}
		transaction_id,_:=sql_insert_transaction(tx_invalid,&from,tx,receipt,tx_err,block_id,block_num,i,int(block.Time().Int64()),num_VTs,amount_transferred)
		for _,vt:= range tx_value_transfers {
			_,val_err:=sql_insert_value_transfer(vt,transaction_id)
			if val_err!=nil {
				log.Error(fmt.Sprintf("EthBot: insertion of value_transfer from %v to % for amount=%v failed. block_num=%v",hex.EncodeToString(vt.src.Bytes()),hex.EncodeToString(vt.dst.Bytes()),vt.value.String(),block_num))
			}
		}
		per_block_value_transferred.Add(per_block_value_transferred,amount_transferred)
		total_VTs=total_VTs+num_VTs
		_=i
		_=gas
		if (receipt!=nil) {
			receipts = append(receipts, receipt)
		}
    }
	if debug_sql_exec_time {
		report_time(start_ts,"transaction loop")
	}
	// is the following call Finalize() really needed
	reward_vts,reward_amount:=add_rewards(*bc_cfg,statedb,block,block_id);
	per_block_value_transferred.Add(per_block_value_transferred,reward_amount)
	total_VTs=total_VTs+reward_vts
	sql_update_block_stats(block_id,per_block_value_transferred,reward_amount,len(block_transactions),total_VTs)

	return nil
}
func dump_vtransfers(items []*Value_transfer_t) {
	for i,item := range items {
		log.Info(fmt.Sprintf("%v : value=%v, from %v -> to %v",i,item.value.String(),item.src.Hex(),item.dst.Hex()))
	}
}
func (bot *EthBot_t) process_StructLogs(prev_statedb *state.StateDB,statedb *state.StateDB, block *types.Block, block_id Block_id_t,block_num Block_num_t, src common.Address,msg *types.Message,	tx	*types.Transaction,logs [](vm.StructLog),receipt *types.Receipt) []*Value_transfer_t {
	var (
		stack           []*Stack_frame_t
		execution_err	error=nil
	)
	kind:=VALTRANSF_TRANSACTION
	var toaddr *common.Address = &common.Address{}
	from:=msg.From();
	null_addr:=common.Address{}
	if !bytes.Equal(receipt.ContractAddress.Bytes(),null_addr.Bytes()) {
		toaddr.Set(receipt.ContractAddress)
		kind=VALTRANSF_CONTRACT_CREATION
	} else {
		toaddr.SetBytes(tx.To().Bytes())
	}

	trsf:=&Value_transfer_t {
			depth: 0,
			block_id: block_id,
			block_num: block_num,
			src: &from,
			dst: toaddr,
			value: tx.Value(),
			kind: kind,
	}
	transfers_slice:=make([]*Value_transfer_t,0)
	transfers_slice=append(transfers_slice,trsf)

	sframe:=&Stack_frame_t {
		transfers: transfers_slice,
	}
	sframe.acct_addr=*toaddr
	stack=make([]*Stack_frame_t,0)
	stack=append(stack,sframe)
	if len(logs)>0 {
//		log.Info(fmt.Sprintf("EthBot: contract with %v instructions is being processed",len(logs)))
	}
	var i int = 0
	var ilen int =len(logs)
	for i<ilen {
		instruction:=logs[i]
		_=i
		if instruction.Depth == len(stack) - 1 {
			returnFrame := stack[len(stack) - 1]
			stack = stack[:len(stack) - 1]
			topFrame := stack[len(stack) - 1]
			// Our call succeded, so add any transfers that happened to the current stack frame
			topFrame.transfers = append(topFrame.transfers, returnFrame.transfers...)

        } else if instruction.Depth < len(stack) {
			panic(fmt.Sprintf("EthBot: unexpected stack transition: was %v, now %v", len(stack), instruction.Depth))
		}
		var instruction_error error
		if instruction.Err!=nil {
			instruction_error=instruction.Err
		} else {
			if (instruction.EthBotErr!=nil) {
				instruction_error=instruction.EthBotErr
			}
		}
	    // If we just returned from a call
		switch instruction.Op {
		    case vm.CREATE:
			    // CREATE adds a frame to the stack, but we don't know their address yet - we'll fill it in
				// when the call returns.
		        value := instruction.Stack[len(instruction.Stack) - 1]
		        src := stack[len(stack) - 1].acct_addr
				newsrc:=common.Address{}
				newsrc.Set(src)
				var err_str string=""
				var instruction_error error
				if instruction.Err!=nil {
					instruction_error=instruction.Err
				} else {
					if (instruction.EthBotErr!=nil) {
						instruction_error=instruction.EthBotErr
					}
				}
				if (instruction_error!=nil) {
					err_str=fmt.Sprintf(`{"opcode":"%v","vm_err":"%v","gas_limit":%v,"src":"%v","dst":"0","value":%v}`,instruction.Op.String(),instruction_error.Error(),instruction.GasLimit,hex.EncodeToString(src.Bytes()),value.String())
					value.Set(big.NewInt(0))	// we clear the value since there is an error. Real value will go in error variable
				}
				dst_addr:=common.Address{}
				dst_addr.Set(instruction.CreateAddr)
				trsf:=&Value_transfer_t {
							depth: len(stack),
							block_id: block_id,
							block_num: block_num,
							src: &newsrc,
							dst: &dst_addr,
							value: value,
							kind: VALTRANSF_CONTRACT_CREATION ,
							err_str: err_str,
				}
				if instruction.Err==nil {	// CALL is not erroneous, so add a stackframe
					var transfers []*Value_transfer_t
					transfers = append(transfers,trsf)
			        frame := &Stack_frame_t {
			            op: instruction.Op,
				        acct_addr: common.Address{},
				        transfers: transfers,
					}
			        stack = append(stack, frame)
				} else {	// CALL is erronous, no stack increase, just add the unsuccessful transfer attempt
					transfers:=stack[len(stack)-1].transfers
					transfers=append(transfers,trsf)
					stack[len(stack)-1].transfers=transfers
				}

		    case vm.CALL:
				// CALL adds a frame to the stack with the target address and value
		        value := instruction.Stack[len(instruction.Stack) - 3]
				dest := common.BigToAddress(instruction.Stack[len(instruction.Stack) - 2])
				src := stack[len(stack) - 1].acct_addr
				var err_str string=""
				var instruction_error error
				if instruction.Err!=nil {
					instruction_error=instruction.Err
				} else {
					if (instruction.EthBotErr!=nil) {
						instruction_error=instruction.EthBotErr
					}
				}
				if (instruction_error!=nil) {
					err_str=fmt.Sprintf(`{"opcode":"%v","vm_err":"%v","gas_limit":%v,"src":"%v","dst":"%v","value":%v}`,instruction.Op.String(),instruction_error.Error(),instruction.GasLimit,hex.EncodeToString(src.Bytes()),hex.EncodeToString(dest.Bytes()),value.String())
					value.Set(big.NewInt(0))	// we clear the value since there is an error. Real value will go in error variable
				}
				trsf:=&Value_transfer_t {
							depth: len(stack),
							block_id: block_id,
							block_num: block_num,
							src: &src,
							dst: &dest,
							value: value,
							kind: VALTRANSF_CONTRACT_TRANSACTION,
							err_str: err_str,
				}
				if instruction.Err==nil {	// CALL is not erroneous, so add a stackframe
					var transfers []*Value_transfer_t
					transfers = append(transfers,trsf)
			        frame := &Stack_frame_t{
			            op: instruction.Op,
			            acct_addr: dest,
				        transfers: transfers,
					}
			        stack = append(stack, frame)
				} else {	// CALL is erronous, no stack increase, just add the unsuccessful transfer attempt
					transfers:=stack[len(stack)-1].transfers
					transfers=append(transfers,trsf)
					stack[len(stack)-1].transfers=transfers
				}

		    case vm.CALLCODE: fallthrough
		    case vm.DELEGATECALL:
		        // CALLCODE and DELEGATECALL don't transfer value or change the from address, but do create
		        // a separate failure domain.
			  if (instruction.Err==nil) {
				frame := &Stack_frame_t{
		            op: instruction.Op,
					acct_addr: stack[len(stack) - 1].acct_addr,
		        }
				stack = append(stack, frame)
			  }
		    case vm.SELFDESTRUCT:
		        // SELFDESTRUCT results in a transfer back to the calling address.
		        frame := stack[len(stack) - 1]
				value:=big.NewInt(0) // SQL procedure will find out what is the balance to return to caller
				newsrc:=common.Address{}
				newsrc.Set(frame.acct_addr)
				var err_str string=""
				var instruction_error error
				if instruction.Err!=nil {
					instruction_error=instruction.Err
				} else {
					if (instruction.EthBotErr!=nil) {
						instruction_error=instruction.EthBotErr
					}
				}
				dest:=common.Address{}
				destination:=common.Address{}
				if (len(instruction.Stack)>0) {
					destination = common.BigToAddress(instruction.Stack[len(instruction.Stack) - 1])
					dest.Set(destination)
				}
				suicide_balance:=big.NewInt(0)
				var exists bool
				tmp_account_id:=lookup_account(&frame.acct_addr)
				if (tmp_account_id==0) {
				} else {
					suicide_balance,exists=modified_balances[tmp_account_id]
					if !exists {
						suicide_balance= prev_statedb.GetBalance(frame.acct_addr)
					} else {
					}
				}
				if (instruction_error!=nil) {
					err_str=fmt.Sprintf(`{"opcode":"%v","vm_err":"%v","gas_limit":%v,"src":"%v","dst":"%v","value":%v}`,instruction.Op.String(),instruction_error.Error(),instruction.GasLimit,hex.EncodeToString(src.Bytes()),hex.EncodeToString(dest.Bytes()),value.String())
					value.Set(big.NewInt(0))	// we clear the value since there is an error. Real value will go in error variable
				} else {
				}
				trsf:=&Value_transfer_t {
						depth: len(stack),
						block_id: block_id,
						block_num: block_num,
						src: &newsrc,
						dst: &dest,
						value: suicide_balance,
						kind: VALTRANSF_CONTRACT_SELFDESTRUCT,
						err_str: err_str,
				}
		        frame.transfers = append(frame.transfers, trsf)
		} //switch
		if (instruction_error!=nil) {
			ttt:=stack[len(stack) - 1].transfers
			zero:=big.NewInt(0)
			for j:=0;j<len(ttt);j++ {	// mark all transfers as CANCELLED because of this error (only transfers of the current depth)`
				t:=ttt[j]
				var dst_str string
				if t.dst!=nil {
					dst_str=hex.EncodeToString(t.dst.Bytes())
				}
				efmt:=fmt.Sprintf(`{"opcode":"%v","pc":%v,"error":"%v","gas_limit":%v,"gas":%v,"depth":%v,"src":"%v","dst":"%v","value":%v}`,instruction.Pc,instruction.Op.String(),instruction_error.Error(),instruction.GasLimit,instruction.Gas,instruction.Depth,hex.EncodeToString(t.src.Bytes()),dst_str,t.value.String())
				ttt[j].err_str=efmt
				ttt[j].value.Set(zero)
			}
		}
		i++
	} // for

	if len(stack)>1 {
		utils.Fatalf("EthBot: transaction wasn't completed. bug in code or bad blockchain data, stack len=%v",len(stack));
	} else if (len(stack)==1) {
	}
	_=execution_err
	return stack[0].transfers
}
func process_genesis_block(chain *core.BlockChain,block *types.Block, gsd state.Dump) error {
	i := 0
	log.Info("EthBot: loading accounts from the Genesis block","num_accounts",len(gsd.Accounts))

	block_id,err:=block2sql(chain,block,0)
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error inserting genesis block: %v",err))
		return err
	}
	var null_addr []byte=make([]byte,common.AddressLength);
    for address, account := range gsd.Accounts {
        balance, ok := new(big.Int).SetString(account.Balance, 10)
        if !ok {
			panic("EthBot: could not decode balance of genesis account")
        }
		destination_addr:=common.HexToAddress(address)
        transfer := &Value_transfer_t {
            depth: 0,
			block_id: block_id,
			block_num: 0,
            value: balance,
            kind: VALTRANSF_GENESIS,
        }
		transfer.src=nil	// by seting to `nil` we assign source account_id to -1 (NONEXISTENT account)
		transfer.dst=&common.Address{}
		if (transfer.src!=nil) {
			transfer.src.SetBytes(null_addr)
		}
		if (transfer.dst!=nil) {
			transfer.dst.Set(destination_addr)
		}
		_,val_err:=sql_insert_value_transfer(transfer,-1)
		if (val_err!=nil) {
			log.Error(fmt.Sprintf("EthBot: error inserting value transfer for address %v",hex.EncodeToString(transfer.dst.Bytes())))
			os.Exit(2)
		}
		i += 1
    }
	return err
}
