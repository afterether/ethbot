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
	"math/big"
	"fmt"
	"encoding/hex"
	"errors"
	"time"
	"io/ioutil"
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
	"github.com/robertkrimen/otto"
	"gopkg.in/urfave/cli.v1"
)
func (bot *EthBot_t) verify_genesis_block_hashes() {
	block:=bot.ethereum.BlockChain().GetBlockByNumber(0)
	if block==nil {
		utils.Fatalf("Genesis block doesn't exist")
	}
	blockchain_hash:=hex.EncodeToString(block.Hash().Bytes())
	_,found:=lookup_block_by_hash(blockchain_hash)
	if !found {
		block_num:=lookup_block_by_num(0)
		if block_num==0 {
			utils.Fatalf("Wrong blockchains, genesis block of SQL and LevelDB do not match")
		}
	}
}
func (bot *EthBot_t) get_cached_balance(block_num Block_num_t,account_id Account_id_t) *big.Int {
	// this func only caches BLOCKCHAIN account, other accounts aren't cached, their balances are coming from the EVM
	var balance *big.Int=nil
	if account_id==NONEXISTENT_ADDRESS_ACCOUNT_ID {
		if bot.export.blockchain_balance!=nil {
			return bot.export.blockchain_balance
		}
	}

	balance=get_previous_balance_SQL(block_num,account_id)

	if account_id==NONEXISTENT_ADDRESS_ACCOUNT_ID {
		bot.export.blockchain_balance=balance
	}
	return balance
}
func (bot *EthBot_t) add_rewards(cfg params.ChainConfig,statedb *state.StateDB, block *types.Block) (int,*big.Int) {
	var (
		big8  = big.NewInt(8)
		big32 = big.NewInt(32)
	)
	total_reward:=big.NewInt(0)
	var block_num Block_num_t=Block_num_t(block.NumberU64())
	timestamp:=int(block.Header().Time.Int64())
	blockReward := frontierBlockReward
	var num_VTs=0
	if (cfg.ChainID.Cmp(big.NewInt(1))==0) { // Ethereum Main Net
		if cfg.IsByzantium(block.Header().Number) {
			blockReward = byzantiumBlockReward
		}
	}
	if (cfg.ChainID.Cmp(big.NewInt(233))==0) { // AfterEther Main Net
		if (block.NumberU64()>154865) {
			blockReward.Set(big.NewInt(2e+18))
		} else {
			blockReward.Set(big.NewInt(5e+18))
		}
	}
	miner_addr:=block.Header().Coinbase
	miner_balance:=statedb.GetBalance(miner_addr)
	reward := new(big.Int).Set(blockReward)
	first_uncle_addr:=common.Address{}
	first_uncle_balance:=big.NewInt(0)
	first_uncle_value:=big.NewInt(0)
	for i, uncle := range block.Uncles() {
		r := new(big.Int)
		r.Add(uncle.Number, big8)
		header:=block.Header();
		r.Sub(r, header.Number)
		r.Mul(r, blockReward)
		r.Div(r, big8)
		uncle_reward:=big.NewInt(0).Set(r)
		transfer:=&vm.Ethbot_EVM_VT_t{
			To: uncle.Coinbase,
			Kind: VALTRANSF_BLOCK_REWARD,
		}
		transfer.Value.Set(uncle_reward)
		transfer.To_balance.Set(statedb.GetBalance(uncle.Coinbase))
		transfer.To_balance.Add(&transfer.To_balance,uncle_reward)
		if i==1 {
			if first_uncle_addr==uncle.Coinbase {
				transfer.To_balance.Set(first_uncle_balance)
				transfer.To_balance.Add(&transfer.To_balance,uncle_reward)
			}
		} else {
			first_uncle_value.Set(&transfer.Value)
			first_uncle_balance.Set(&transfer.To_balance)
			first_uncle_addr.SetBytes(uncle.Coinbase.Bytes())
		}
		sql_insert_value_transfer(transfer,-1,nil,timestamp,block_num)
		total_reward.Add(total_reward,uncle_reward)
		num_VTs++
		r.Div(blockReward, big32)
		reward.Add(reward, r)
		if miner_addr==uncle.Coinbase {		// a miner can mine 3 blocks, main block + both uncles, so we accumulate the values he is mining
			miner_balance.Add(miner_balance,&transfer.Value)
		}
	}

	block_reward:=big.NewInt(0).Set(reward)
	transfer:=&vm.Ethbot_EVM_VT_t{
		To: block.Header().Coinbase,
		Kind: VALTRANSF_BLOCK_REWARD,
	}
	transfer.Value.Set(block_reward)
	transfer.To_balance.Set(miner_balance)
	transfer.To_balance.Add(&transfer.To_balance,block_reward)
	_,_,_,val_err:=sql_insert_value_transfer(transfer,-1,nil,timestamp,block_num)
	if (val_err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error inserting block reward for miner %v",hex.EncodeToString(transfer.To.Bytes())))
		os.Exit(2)
	}
	total_reward.Add(total_reward,block_reward)
	num_VTs++

	miner_balance=statedb.GetBalance(miner_addr)
	return num_VTs,total_reward
}
func (bot *EthBot_t) export_block_data(ethereum *eth.Ethereum, block *types.Block) error {

	export_start_ts:=time.Now().UnixNano() / int64(time.Millisecond)
	per_block_VT_counter=0
	blockchain := ethereum.BlockChain()
	bc_cfg:=blockchain.Config()
	block_num:=Block_num_t(block.NumberU64())
	timestamp:=int(block.Header().Time.Int64())
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
	err=block2sql(blockchain,block,len(block_transactions));
	if (err!=nil) {
		return err;
	}
	vm_cfg:=&vm.Config{
        Debug: false,
	}

    parent := blockchain.GetBlockByHash(block.ParentHash())
    if parent == nil {
		utils.Fatalf("EthBot: could not retrieve parent block for hash %v",block.ParentHash().Hex())
    }

	var (
		totalUsedGas = new(uint64)
	)
	gaspool:=new(core.GasPool).AddGas(block.GasLimit())
	bconf:=blockchain.Config()

	var total_VTs int=0
	per_block_value_transferred:=big.NewInt(0)

    if bconf.DAOForkSupport && bconf.DAOForkBlock != nil && bconf.DAOForkBlock.Cmp(block.Number()) == 0 {
		var num_VTs int = 0
		dao_balance:=statedb.GetBalance(params.DAORefundContract)
		amount_transferred:=big.NewInt(0)
		var fork_value_transfers []*vm.Ethbot_EVM_VT_t=make([]*vm.Ethbot_EVM_VT_t,0,64)
		for _, addr := range params.DAODrainList() {
			trsf:=&vm.Ethbot_EVM_VT_t{
				Kind:			VALTRANSF_FORK,
			}
			trsf.Value.Set(statedb.GetBalance(addr))
			trsf.From.SetBytes(addr.Bytes())
			trsf.From_balance.Set(big.NewInt(0))
			trsf.To.SetBytes(params.DAORefundContract.Bytes())
			dao_balance.Add(dao_balance,&trsf.Value)
			trsf.To_balance.Set(dao_balance)
			if vt_debug {
				vt_dump(trsf)
			}
			fork_value_transfers=append(fork_value_transfers,trsf)
		}
		for _,vt:= range fork_value_transfers {
			_,_,_,val_err:=sql_insert_value_transfer(vt,-1,nil,timestamp,block_num)
			if val_err!=nil {
				utils.Fatalf(fmt.Sprintf("EthBot: error in insertion of (FORK) value_transfer from %v to % for amount=%v failed. block_num=%v",hex.EncodeToString(vt.From.Bytes()),hex.EncodeToString(vt.To.Bytes()),vt.Value.String(),block_num))
			}
			num_VTs++
			amount_transferred.Add(amount_transferred,&vt.Value)
		}
		per_block_value_transferred.Add(per_block_value_transferred,amount_transferred)
		total_VTs=total_VTs+num_VTs
        misc.ApplyDAOHardFork(statedb)
    }
	num_transactions:=len(block_transactions)
	var geth_time int64 =0
	var vt_time int64 =0
	var del_time int64 =0
	var extras_time int64 =0
	var tx_time int64 =0
	modified_accts:=make(map[common.Address]VT_insert_info_t)
    for i, tx := range block_transactions {
		var num_VTs int = 0
		var tx_err error
		var receipt *types.Receipt
		var apply_err error

		vm_err4ethbot:=new(error)
		vm_VTs4ethbot:=make([]*vm.Ethbot_EVM_VT_t,0,64)
		deleted_addresses:=make([]common.Address,0,8)
		amount_transferred:=big.NewInt(0)

        statedb.Prepare(tx.Hash(), block.Hash(), i)
		vm_start_ts:=time.Now().UnixNano() / int64(time.Millisecond)
		receipt, _, apply_err = core.ApplyTransaction(bconf, blockchain, nil, gaspool, statedb, block.Header(), tx, totalUsedGas, *vm_cfg, &vm_VTs4ethbot,&deleted_addresses,vm_err4ethbot)
		vm_end_ts:=time.Now().UnixNano() / int64(time.Millisecond)
		geth_time=geth_time + (vm_end_ts-vm_start_ts)
		tx_status:=1;
		if apply_err != nil {
			log.Error(fmt.Sprintf("EthBot: transaction executed with error, err=%v",apply_err))
			tx_status=0;
		}
		tx_start_ts:=time.Now().UnixNano() / int64(time.Millisecond)
		transaction_id,_:=sql_insert_transaction(tx_status,&vm_VTs4ethbot[0].From,tx,receipt,tx_err,*vm_err4ethbot,block_num,i,int(block.Time().Int64()),num_VTs,amount_transferred)
		tx_end_ts:=time.Now().UnixNano() / int64(time.Millisecond)
		tx_time=tx_time + (tx_end_ts-tx_start_ts)
		for i,evm_vt:= range vm_VTs4ethbot {
			num_VTs++
			if evm_vt.Err==nil {
				amount_transferred.Add(amount_transferred,&evm_vt.Value)
			}
			vt_start_ts:=time.Now().UnixNano() / int64(time.Millisecond)
			valtr_id,_,dst_id,_:=sql_insert_value_transfer(evm_vt, transaction_id,tx,timestamp,block_num)
			vt_end_ts:=time.Now().UnixNano() / int64(time.Millisecond)
			vt_time=vt_time + (vt_end_ts-vt_start_ts)
			if (len(vm_VTs4ethbot[i].Input)>0) || (len(vm_VTs4ethbot[i].Logs)>0) || (len(vm_VTs4ethbot[i].Output)>0) { // we have some extra data
				extras_start_ts:=time.Now().UnixNano() / int64(time.Millisecond)
				sql_insert_value_transfer_extras(vm_VTs4ethbot[i],valtr_id,dst_id,transaction_id)
				extras_end_ts:=time.Now().UnixNano() / int64(time.Millisecond)
				extras_time=del_time + (extras_end_ts-extras_start_ts)
			}
			if len(deleted_addresses)>0 {
				del_start_ts:=time.Now().UnixNano() / int64(time.Millisecond)
				for _,dead_body:=range deleted_addresses {
					sql_mark_account_as_deleted(&dead_body,block_num)
				}
				del_end_ts:=time.Now().UnixNano() / int64(time.Millisecond)
				del_time=del_time + (del_end_ts-del_start_ts)
			}
		}
		per_block_value_transferred.Add(per_block_value_transferred,amount_transferred)
		total_VTs=total_VTs+num_VTs
    }
	reward_vts,reward_amount:=bot.add_rewards(*bc_cfg,statedb,block);
	per_block_value_transferred.Add(per_block_value_transferred,reward_amount)
	total_VTs=total_VTs+reward_vts
	sql_update_block_stats(block_num,per_block_value_transferred,reward_amount,len(block_transactions),total_VTs)
	ETHERS:=big.NewInt(0)
	ETHERS.Div(per_block_value_transferred,big.NewInt(1000000000000000000))  // convert weis to ethers
	if ethbot_instance.export.Alarms_on && (block_num>0) {
		if ethbot_instance.export.Last_block_num<block_num {
			check_alarms(&modified_accts,block_num,statedb)
			update_alarm_status(block_num)
		}
	}
	var export_end_ts int64
	export_end_ts=time.Now().UnixNano() / int64(time.Millisecond)
	processing_time:=export_end_ts-export_start_ts
	log.Info(fmt.Sprintf("EthBot: processed block %v",block.NumberU64()),"TXs",num_transactions,"VTs",total_VTs,"$",ETHERS,"t(ms)",processing_time,"EVM(ms)",geth_time,"VT(ms)",vt_time,"Del(ms)",del_time,"CIO(ms)",extras_time,"TX(ms)",tx_time)
	return nil
}
func check_alarms(modified_accts *map[common.Address]VT_insert_info_t,block_num Block_num_t,statedb *state.StateDB) {
	if ethbot_instance.export.Last_block_num<block_num {
		for addr,info:= range *modified_accts {
				verif_balance_SQL:=get_previous_balance_SQL(block_num,info.account_id)
				verif_balance_BC:=statedb.GetBalance(addr)
				if verif_balance_BC.Cmp(verif_balance_SQL)!=0 {
					send_bad_balance_alarm(block_num,&addr,&info,verif_balance_BC,verif_balance_SQL)
				}
				verif_nonce_SQL:=sql_get_account_nonce(info.account_id)
				verif_nonce_BC:=statedb.GetNonce(addr)
				if verif_nonce_SQL!=verif_nonce_BC {
					send_bad_nonce_alarm(block_num,&addr,&info,verif_nonce_BC,verif_nonce_SQL)
				}
		}
	}
}
func process_genesis_block(chain *core.BlockChain,block *types.Block, gsd state.Dump) error {
	i := 0
	log.Info("EthBot: loading accounts from the Genesis block","num_accounts",len(gsd.Accounts))

	err:=block2sql(chain,block,0)
	if (err!=nil) {
		log.Error(fmt.Sprintf("EthBot: error inserting genesis block: %v",err))
		return err
	}
	timestamp:=int(block.Header().Time.Int64())
    for address, account := range gsd.Accounts {
        balance, ok := new(big.Int).SetString(account.Balance, 10)
        if !ok {
			panic("EthBot: could not decode balance of genesis account")
        }
        transfer := &vm.Ethbot_EVM_VT_t {
            Kind: VALTRANSF_GENESIS,
        }
		addr:=common.HexToAddress(address)
		transfer.Value.Set(balance)
		transfer.To.SetBytes(addr.Bytes())
		transfer.To_balance.Set(balance)
		_,_,_,val_err:=sql_insert_value_transfer(transfer,-1,nil,timestamp,0)
		if (val_err!=nil) {
			log.Error(fmt.Sprintf("EthBot: error inserting value transfer for address %v",hex.EncodeToString(transfer.To.Bytes())))
			os.Exit(2)
		}
		i += 1
    }
	sql_set_block_as_processed(0)
	return err
}
func (bot *EthBot_t) adjust_starting_block_num(starting_block Block_num_t) Block_num_t {
	if (starting_block==-1) {
		starting_block = sql_get_last_block_num()
		if (starting_block==-1) {
			return 0;		// SQL database is emtpy, return genesis block (block_num=0)
		} else {
			starting_block++
		}
	}
	return starting_block
}
func (bot *EthBot_t) adjust_ending_block_num(ending_block Block_num_t) Block_num_t {
	if (ending_block==-1) {
		chain:=bot.ethereum.BlockChain()
		last_block_num:=Block_num_t(chain.CurrentBlock().NumberU64())
		ending_block=last_block_num
	}

	return ending_block
}
func (bot *EthBot_t) export_blocks() {
    chain := bot.ethereum.BlockChain()
    go bot.process_blocks()
	bot.export.Cur_block_num=bot.export.Starting_block
	for true {
		if !bot.export.In_progress { // the process was probably aborted by user
			break;
		}
		block:=chain.GetBlockByNumber(uint64(bot.export.Cur_block_num))
		if block!=nil {
			bot.blocks<-block
		} else {
			break;
		}
		tmp_num:=Block_num_t(chain.CurrentBlock().Number().Uint64())
		if (bot.export.Cur_block_num==tmp_num) {	// we reached the end of the blockchain, stored in the DB
			break
		}
		if (bot.export.Direction>0) {
			bot.export.Cur_block_num++
			if (bot.export.Cur_block_num>bot.export.Ending_block) {
				break
			}
		} else {
			if bot.export.Direction<0 {
				bot.export.Cur_block_num--
				if (bot.export.Cur_block_num<bot.export.Ending_block) {
					break
				}
			} else {
				break;		// if Direction is 0 it means we are processing a single block, so we exit loop
			}
		}
	}
	// at this point we processed all the blocks up to the ending_block
	// so, if user specified listening mode, enter into it and continue exporting incoming blocks
	if (bot.export.In_progress) {	// the user can interrupt export , so we check again
		if (bot.export.Listening_mode) {
		    bot.head_ch= make(chan core.ChainHeadEvent)
			bot.head_sub=chain.SubscribeChainHeadEvent(bot.head_ch)
			go bot.listen4blocks();
		} else {
			bot.export.In_progress=false
		}
	}
}
func (bot *EthBot_t) listen4blocks() {

	if (bot.listening_started) {
		return
	}
	bot.listening_started=true
    chain := bot.ethereum.BlockChain()
	log.Info("EthBot: entering listening mode, to export blocks as they arrive")
  again:
	evt:= <-bot.head_ch
	if (evt.Block==nil) {	// this routine exits on receiving null block
		bot.listening_started=false
		return;
	}
	new_block_num:=Block_num_t(evt.Block.NumberU64())
	last_block_num:=sql_get_last_block_num()
	var block *types.Block
	for block_num:=(last_block_num+1);block_num<=new_block_num;block_num++ {
		block=chain.GetBlockByNumber(uint64(block_num))
		bot.blocks<-block
	}
	if (block!=nil) {
		ethbot_instance.update_main_stats(int(block.NumberU64()))
	}
	if (bot.export.In_progress==false) {
		bot.listening_started=false
		return
	}
	goto again;
}
func (bot *EthBot_t) blockchain_export_start(starting_block Block_num_t,ending_block Block_num_t) error {

	if (bot.export.In_progress) {
		log.Error("EthBot: export already in progress");
		return errors.New("Export already in progress");
	}
	starting_block=bot.adjust_starting_block_num(starting_block)

	// global variables , per export process
	bot.clear_export_state_variables()
	// end of global variables, per export process

	// export control variables
	bot.export.In_progress=true
	bot.export.Range_export=false
	bot.export.Starting_block=starting_block
	bot.export.Ending_block=ending_block
	bot.export.Last_block_num=sql_get_last_block_num()
	bot.export.Listening_mode=false
	bot.export.Direction=1
	if bot.export.Ending_block!=-1 {
		if (bot.export.Starting_block<bot.export.Ending_block) {
			bot.export.Direction=1
		} else {
			if (bot.export.Starting_block<bot.export.Ending_block) {
				bot.export.Direction=-1
			} else {
				bot.export.Direction=0
			}
		}
	} else {
		bot.export.Listening_mode=true
		bot.export.Ending_block=bot.adjust_ending_block_num(bot.export.Ending_block)
		log.Info(fmt.Sprintf("EthBot: will export blocks from %v to %v, after that, will enter listening mode",bot.export.Starting_block,bot.export.Ending_block))
	}
	// end of export control variables

	go bot.export_blocks()

	return nil
}
func (bot *EthBot_t) blockchain_export_stop() bool {

	if (!bot.export.In_progress) {
		log.Error("EthBot: export process is not running")
		return false
	}
	if (bot.head_ch!=nil) {			// if listening routine is running
		if (bot.head_sub!=nil) {
			bot.head_sub.Unsubscribe()
		}
	}
	bot.export.In_progress=false;
	return true
}
func (bot *EthBot_t) process_blocks() {

	if bot.process_started {	// additional security check, to do not launch this twice
		return
	}
	bot.process_started=true
  again:

	if !bot.export.In_progress {
		log.Info("EthBot: block data export to SQL is over")
	}
	var block *types.Block
    block = <-bot.blocks
	if (block==nil) {
		log.Error("EthBot: null block received")
		goto again;
	}
	block_num:=Block_num_t(block.Number().Uint64())
	err := bot.export_block_data(bot.ethereum, block)
    if err != nil {
		log.Error(fmt.Sprintf("Data export of block %v failed: %v",block_num,err))
		if err==ErrAncestorNotFound {	// probably a chain split occurred, fix it
			bot.repair_chain_split(block)
		} else {
	       log.Error("EthBot: Unable to export block data","block", block.Number().Uint64(),"err",err)
		}
        goto again;
    }

	if (err == nil) {
		if (bot.export.Range_export) {
			// range export does not update the `last_block` counter, since it is an "out of order" export process
		} else {
			sql_set_last_block_num(block_num)
			bot.export.Exported_block=block_num
		}
	}
	goto again
}
func (bot *EthBot_t) repair_chain_split(block *types.Block) {
	// scan back until we find the last valid parent and then re-insert all the blocks from that number
	log.Info(fmt.Sprintf("Chain split detected at block %v",block.NumberU64()))
	chain:=bot.ethereum.BlockChain()
	block_num:=Block_num_t(block.NumberU64())
	last_block_num:=block_num
	for(block_num>0) {
		_,block_found:=lookup_block_by_hash(hex.EncodeToString(block.Header().ParentHash.Bytes()));
		if block_found { // we found our first valid parent
			log.Info(fmt.Sprintf("EthBot: fixing chain split from block %v",block_num))
			bot.repair_chain_split_insert_blocks(block_num,last_block_num)
			return
		}
		block_num--
		block=chain.GetBlockByNumber(uint64(block_num))
		if (block==nil) {
			utils.Fatalf("correct block wasnt found , block_num=",block_num)
		}
	}
}
func (bot *EthBot_t) clear_export_state_variables() {
	// these variables are kept per export process, once the export finishes/starts, they are cleared
	bot.eb_accounts=make(map[common.Address]Account_id_t)
	bot.export.blockchain_balance=nil
}
func (bot *EthBot_t) repair_chain_split_insert_blocks(from_num Block_num_t,to_num Block_num_t) {
	chain:=bot.ethereum.BlockChain()
	bot.clear_export_state_variables()
	i:=from_num
	for ;i<=to_num;i++ {
		block:=chain.GetBlockByNumber(uint64(i))
		if (block!=nil) {
			err:=bot.export_block_data(bot.ethereum,block)
			if (err!=nil) {
				log.Error(fmt.Sprintf("EthBot: repairing chain split: error found inserting block %v :  %v",i,err))
			}
		} else {
			log.Error(fmt.Sprintf("EthBot: can't get block number %v",i))
		}
	}
}
func (bot *EthBot_t) check_export_on_startup(ctx *cli.Context) {
    if ctx.GlobalIsSet(utils.NoExportFlag.Name) {
		// --noexport flag used, we skip exporting to SQL at startup
		log.Info("EthBot: export not launched as requested by user");
	} else {
		go bot.blockchain_export_start(-1,-1) // start from the last_block previously recorded and after importing everything enter into listening mode
	}
    if ctx.GlobalIsSet(utils.PTXOutFlag.Name) {
		// Export pending transactions to SQL
		log.Info("EthBot: Exporting TX Pool contents is enabled");
		go bot.init_pending_tx_gathering()
	} else {
	}
}
func (bot *EthBot_t) blockchain_export_status(exp *Export_t) *otto.Object {
	jsre:=console_obj.JSRE()
	vm:=jsre.VM()
	obj_str:=fmt.Sprintf(`({"starting_block":%d,"current_block":%d,"ending_block":%d,"direction":%d,"listening_mode":%v,"user_cancelled":%v,"in_progress":%v,"alarms_on":%v})`,exp.Starting_block,exp.Cur_block_num,exp.Ending_block,exp.Direction,exp.Listening_mode,exp.User_cancelled,exp.In_progress,exp.Alarms_on);
	object, err := vm.Object(obj_str)
	if (err!=nil) {
		utils.Fatalf("Failed to create object in Javascript VM for blockchain export status: %v",err)
	}
	return object
}
func (bot *EthBot_t) export_block_range(starting_block Block_num_t,ending_block Block_num_t) bool {
	ethereum:=bot.ethereum;

	log.Info("EthBot: this function is under development and has not been finished yet.");
	return false;
	if (bot.export.In_progress) {
		log.Info("EthBot: Export already in progress")
		return false
	}
	bot.export.In_progress=true
	bot.export.Range_export=true
	bot.export.User_cancelled=false
	blockchain := ethereum.BlockChain()
	starting_block=ethbot_instance.adjust_starting_block_num(starting_block)
	current:=starting_block
	for (current<=ending_block) {
		bptr:=blockchain.GetBlockByNumber(uint64(current))
		bot.export_block_data(ethereum, bptr);
		current++
		if (bot.export.User_cancelled) {
			log.Info("EthBot: range export cancelled by user")
			break
		}
	}
	bot.export.In_progress=false
	return true
}
func check_input_export_block_range(arg_starting_block otto.Value,arg_ending_block otto.Value) (Block_num_t,Block_num_t,error) {

	p_starting_block,p_ending_block,err:=check_input_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return 0,0,err
	}
	return p_starting_block,p_ending_block,nil
}
func (bot *EthBot_t) update_main_stats(p_block_num int) bool {
	var block_num Block_num_t
	if p_block_num==-1 {
		block_num=sql_get_last_block_num()
		if (block_num<0) {
			return false
		}
	} else {
		block_num=Block_num_t(p_block_num)
	}
	return sql_update_main_stats(block_num)
}
func send_bad_balance_alarm(block_num Block_num_t,addr *common.Address,info *VT_insert_info_t,bad_balance,good_balance *big.Int) {
	var address_str string
	address_str=hex.EncodeToString(addr.Bytes())
	diff:=big.NewInt(0)
	diff.Set(good_balance)
	diff.Sub(good_balance,bad_balance)
	message:=fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v",block_num,address_str,bad_balance.String(),good_balance.String(),diff.String(),"balances do not match")
	msg_bytes := []byte(message)
	filename:=fmt.Sprintf("%v/balance-%v-%v",alarms_dir,block_num,info.valtr_id)
	err := ioutil.WriteFile(filename, msg_bytes, 0644)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Cant write alarm file for bad balance: %v",err))
	}
}
func send_bad_nonce_alarm(block_num Block_num_t,addr *common.Address,info *VT_insert_info_t,bad_nonce,good_nonce uint64) {
	var address_str string
	address_str=hex.EncodeToString(addr.Bytes())
	message:=fmt.Sprintf("%v\t%v\t%v\t%v\t%v",block_num,address_str,bad_nonce,good_nonce,"nonces do not match")
	msg_bytes := []byte(message)
	filename:=fmt.Sprintf("%v/nonce-%v-%v",alarms_dir,block_num,info.valtr_id)
	err := ioutil.WriteFile(filename, msg_bytes, 0644)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Cant write alarm file for bad nonce: %v",err))
	}
}
func update_alarm_status(block_num Block_num_t) {

	message:=fmt.Sprintf("%v",block_num)
	msg_bytes:=[]byte(message)
	filename:=fmt.Sprintf("%v/status",alarms_dir)
	err := ioutil.WriteFile(filename, msg_bytes, 0644)
	if err!=nil {
		utils.Fatalf(fmt.Sprintf("Cant write status alarm file: %v",err))
	}

}
func (bot *EthBot_t) alarms_on() bool {

	if (bot.export.Alarms_on) {
		log.Error("EthBot: alarms are already turned on")
		return false
	}
	bot.export.Alarms_on=true;
	return true
}
func (bot *EthBot_t) alarms_off() bool {

	if (!bot.export.Alarms_on) {
		log.Error("EthBot: alarms are already turned off")
		return false
	}
	bot.export.Alarms_on=false;
	return true
}
