package main
import (
	"math/big"
	"fmt"
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
	modified_balances map[Account_id_t]*big.Int
)

func add_rewards(cfg params.ChainConfig,statedb *state.StateDB, block *types.Block,block_id Block_id_t, ) {
	var (
	    big8  = big.NewInt(8)
	    big32 = big.NewInt(32)
	)
	var block_num Block_num_t=Block_num_t(block.NumberU64())
    blockReward := frontierBlockReward
    if cfg.IsByzantium(block.Header().Number) {
        blockReward = byzantiumBlockReward
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
		sql_insert_value_transfer(transfer,-1)
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
	sql_insert_value_transfer(transfer,-1)
}
func (bot *EthBot_t) trace_block(ethereum *eth.Ethereum, block *types.Block) error {

	blockchain := ethereum.BlockChain()
	bc_cfg:=blockchain.Config()
	block_num:=Block_num_t(block.NumberU64())
	ethbot_instance.export.Cur_block_num=Block_num_t(block.NumberU64())
    if block_num == 0 {
		statedb, err := blockchain.StateAt(block.Root())
        if err != nil {
            log.Error("Can't get StateAt()","block_num",block.Number().Uint64())
            return  err
        }
        genesis_state_dump := statedb.RawDump()
        err=process_genesis_block(blockchain,block,genesis_state_dump)
		return err;
    }

    statedb, err := blockchain.StateAt(blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1).Root() )
    if err != nil {
		log.Error("Can't get StateAt()","block_num",block.NumberU64())
        return err
    }
	block_id,err:=block2sql(blockchain,block);
	if (err!=nil) {
		return err;
	}
	lcfg:=&vm.LogConfig{
            DisableMemory: false,
            DisableStack: false,
            DisableStorage: false,
            FullStorage: false,
    }
	vm_cfg:=&vm.Config{
        Debug: true,
        EnableJit: false,
        ForceJit: false,
	}

    parent := blockchain.GetBlockByHash(block.ParentHash())
    if parent == nil {
        utils.Fatalf("Could not retrieve parent block for hash %v",block.ParentHash().Hex())
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
    for i, tx := range block.Transactions() {
		structLogger := vm.NewStructLogger(lcfg)
		vm_cfg.Tracer=structLogger
        statedb.Prepare(tx.Hash(), block.Hash(), i)

	    msg, err := tx.AsMessage(types.MakeSigner(bconf, block.Number()))
	    if err != nil {
			log.Error("EthBot: Error in tx.AsMessage()")
			return err
		}

        receipt, gas, err := core.ApplyTransaction(bconf, blockchain, nil, gaspool, statedb, block.Header(), tx, totalUsedGas, *vm_cfg)
        if err != nil {
			log.Error("EthBot: Error in core.ApplyTransaction()")
            return err
		}
		transaction_id,err:=sql_insert_transaction(&msg,tx,block_id,block_num,i,int(block.Time().Int64()))

		from:=msg.From();
		if core.Vmerr4ethbot!=nil {	// Vmerr4ethbot is set in state_transition.go:TransitionDb()
			log.Info("caught error in vm","err",core.Vmerr4ethbot)
			// ToDo: update transaction setting error code
		} else {
			bot.process_StructLogs(statedb,block,block_id,block_num,from,&msg,tx,structLogger.StructLogs(),receipt,transaction_id)
		}
        miner_tx_fee:=&Value_transfer_t {
			depth: 0,
			block_id: block_id,
			block_num: block_num,
			src: &from,
			dst: &block.Header().Coinbase,
			value: new(big.Int).Mul(receipt.GasUsed, tx.GasPrice()) ,
			kind: VALTRANSF_TX_FEE,
		}
		sql_insert_value_transfer(miner_tx_fee,transaction_id)
		_=i
		_=gas
    }
	add_rewards(*bc_cfg,statedb,block,block_id);

	return nil
}
func dump_vtransfers(items []*Value_transfer_t) {
	for i,item := range items {
		log.Info(fmt.Sprintf("%v : value=%v, from %v -> to %v",i,item.value.String(),item.src.Hex(),item.dst.Hex()))
	}
}
func fix_creation_addresses(transfers []*Value_transfer_t, address common.Address) {
    for _, transfer := range transfers {
        if *transfer.src == (common.Address{}) {
            transfer.src.Set(address)
        } else if *transfer.dst == (common.Address{}) {
            transfer.dst.Set(address)
        }
    }
}
func (bot *EthBot_t) process_StructLogs(statedb *state.StateDB, block *types.Block, block_id Block_id_t,block_num Block_num_t, src common.Address,msg *types.Message,	tx	*types.Transaction,logs [](vm.StructLog),receipt *types.Receipt,transaction_id int64)  {
	var (
		stack           []*Stack_frame_t
		execution_err	error=nil
	)
	kind:=VALTRANSF_TRANSACTION
	var toaddr *common.Address = &common.Address{}
	if (tx.To()==nil) {
		kind=VALTRANSF_CONTRACT_CREATION
	} else {
		toaddr.SetBytes(tx.To().Bytes())
	}
	from:=msg.From();
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
		log.Info(fmt.Sprintf("Contract with %v instructions is being processed",len(logs)))
	}
	for i,instruction:=range logs {
		_=i
		if (instruction.Err!=nil) {
			if instruction.Op==vm.CALL {
				//
			} else {
				stack=stack[:len(stack) - 1]
		        if len(stack) == 0 {
					execution_err = instruction.Err
				}
				continue;
			}
		}
	    // If we just returned from a call
		if instruction.Depth == len(stack) - 1 {
			returnFrame := stack[len(stack) - 1]
			stack = stack[:len(stack) - 1]
			topFrame := stack[len(stack) - 1]
			if topFrame.op == vm.CREATE {
				// Now we know our new address, fill it in everywhere.
				topFrame.acct_addr = common.BigToAddress(instruction.Stack[len(instruction.Stack) - 1])
				fix_creation_addresses(returnFrame.transfers, topFrame.acct_addr)
			}
			// Our call succeded, so add any transfers that happened to the current stack frame
			topFrame.transfers = append(topFrame.transfers, returnFrame.transfers...)
        } else if instruction.Depth != len(stack) {
			panic(fmt.Sprintf("Unexpected stack transition: was %v, now %v", len(stack), instruction.Depth))
		}

		switch instruction.Op {
		    case vm.CREATE:
			    // CREATE adds a frame to the stack, but we don't know their address yet - we'll fill it in
				// when the call returns.
		        value := instruction.Stack[len(instruction.Stack) - 1]
		        src := stack[len(stack) - 1].acct_addr

		        var transfers []*Value_transfer_t
		        if value.Cmp(big.NewInt(0)) != 0 {
					newsrc:=common.Address{}
					newsrc.Set(src)
		            transfers = []*Value_transfer_t{
						&Value_transfer_t {
							depth: len(stack),
							block_id: block_id,
							block_num: block_num,
							src: &newsrc,
							dst: &common.Address{},
							value: value,
							kind: VALTRANSF_CONTRACT_CREATION ,
						},
					}
				}
		        frame := &Stack_frame_t {
		            op: instruction.Op,
			        acct_addr: common.Address{},
		            transfers: transfers,
				}
		        stack = append(stack, frame)

		    case vm.CALL:
				// CALL adds a frame to the stack with the target address and value
		        value := instruction.Stack[len(instruction.Stack) - 3]
				dest := common.BigToAddress(instruction.Stack[len(instruction.Stack) - 2])
			    var transfers []*Value_transfer_t
				if (instruction.Err==nil) {
			        if value.Cmp(big.NewInt(0)) != 0 {
						src := stack[len(stack) - 1].acct_addr
						trsf:=&Value_transfer_t {
							depth: len(stack),
							block_id: block_id,
							block_num: block_num,
							src: &src,
							dst: &dest,
							value: value,
							kind: VALTRANSF_CONTRACT_TRANSACTION,
						}
			            transfers = append(transfers,trsf)
					}
				}
		        frame := &Stack_frame_t{
		            op: instruction.Op,
		            acct_addr: dest,
		            transfers: transfers,
		        }
		        stack = append(stack, frame)

		    case vm.CALLCODE: fallthrough
		    case vm.DELEGATECALL:
		        // CALLCODE and DELEGATECALL don't transfer value or change the from address, but do create
		        // a separate failure domain.
				frame := &Stack_frame_t{
		            op: instruction.Op,
					acct_addr: stack[len(stack) - 1].acct_addr,
		        }
				stack = append(stack, frame)
		    case vm.SELFDESTRUCT:
		        // SELFDESTRUCT results in a transfer back to the calling address.
		        frame := stack[len(stack) - 1]
				bc:=bot.ethereum.BlockChain()
				prev_statedb,err:=bc.StateAt(bc.GetBlock(block.ParentHash(), block.NumberU64()-1).Root())
				if (err!=nil) {
					utils.Fatalf("Can't get state inside vm.SELFDESTRUCT case")
				}
		        value := prev_statedb.GetBalance(frame.acct_addr)

		        dest := src
		        if len(stack) > 1 {
		            dest = stack[len(stack) - 2].acct_addr
		        }
				newsrc:=common.Address{}
				newsrc.Set(frame.acct_addr)
		        if value.Cmp(big.NewInt(0)) != 0 {
					trsf:=&Value_transfer_t {
						depth: len(stack),
						block_id: block_id,
						block_num: block_num,
						src: &newsrc,
						dst: &dest,
						value: value,
						kind: VALTRANSF_CONTRACT_SELFDESTRUCT,
					}
		            frame.transfers = append(frame.transfers, trsf)
				}
		} //switch

	}
	if len(stack)>1 {
		utils.Fatalf("Transaction wasn't completed. bug in code or bad blockchain data, stack len=%v",len(stack));
	} else if (len(stack)==1) {
		fix_creation_addresses(stack[0].transfers,receipt.ContractAddress)
		for i:=0;i<len(stack[0].transfers);i++ {
			sql_insert_value_transfer(stack[0].transfers[i],transaction_id)
		}
	}
	_=execution_err
}
func process_genesis_block(chain *core.BlockChain,block *types.Block, gsd state.Dump) error {
	i := 0
	log.Info("EthBot: loading accounts from the Genesis block","num_accounts",len(gsd.Accounts))

	block_id,err:=block2sql(chain,block)
	var null_addr []byte=make([]byte,common.AddressLength);
    for address, account := range gsd.Accounts {
        balance, ok := new(big.Int).SetString(account.Balance, 10)
        if !ok {
            panic("Could not decode balance of genesis account")
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
		sql_insert_value_transfer(transfer,-1)
		i += 1
    }
	return err
}
