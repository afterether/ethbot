package main

import (
	"fmt"
	"encoding/hex"
	"errors"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/robertkrimen/otto"
	"gopkg.in/urfave/cli.v1"

)
var (
	ethbot_instance *EthBot_t
)
func NewEthBot(ctx *node.ServiceContext) (node.Service, error) {
    var ethereum *eth.Ethereum
    if err := ctx.Service(&ethereum); err != nil {
        return nil, err
    }
    ethbot_instance=&EthBot_t {
        ethereum:				ethereum,
		head_ch:				nil,
        head_sub:				nil,
		blocks:					make(chan *types.Block),
		verification:			Verification_t{},
		export:					Export_t{},
    }
	return ethbot_instance, nil
}
func (bot *EthBot_t) adjust_starting_block_num(starting_block Block_num_t) Block_num_t {
	if (starting_block==-1) {
		starting_block = sql_get_last_block_num()
		if (starting_block==-1) {
			return 0;		// SQL database is emtpy, return genesis block (block_num=0)
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
	// exporting process uses 2 functions:
	// export_blocks() which feeds the blocks through the channel bot->blocks
	// process_blocks() which does the actual export to SQL server
	// 
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
			log.Info("EthBot: null block received from the blockchain, aborting")
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
	bot.export.In_progress=true
	bot.export.Range_export=false
	bot.export.Starting_block=starting_block
	bot.export.Ending_block=ending_block
	bot.export.Listening_mode=false
	bot.export.Direction=1
	bot.export.non_existent_balance=nil
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
	go bot.export_blocks()

	return nil
}
func (bot *EthBot_t) blockchain_export_stop() bool {

	if (!bot.export.In_progress) {
		log.Error("EthBot: export process is not running")
		return false
	}
	if (bot.head_ch!=nil) {			// if listening routine is running
		nil_event:=core.ChainHeadEvent{
			Block: nil,
		}
		bot.head_ch <- nil_event// tell listening routine to exit
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

	var block *types.Block
    block = <-bot.blocks
	if (block==nil) { // nil means Exit
		log.Error("EthBot: null block received")
		goto again;
	}
	block_num:=Block_num_t(block.Number().Uint64())
	err := bot.trace_block(bot.ethereum, block)
    if err != nil {
		if err==ErrAncestorNotFound {	// probably a chain split occurred, fix it
			bot.repair_chain_split(block)
		} else {
	       log.Error("EthBot: Unable to trace transactions","block", block.Number().Uint64(),"err",err)
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
	chain:=bot.ethereum.BlockChain()
	block_num:=Block_num_t(block.NumberU64())
	last_block_num:=block_num
	for(block_num>0) {
		parent_id,parent_block_num:=lookup_block_by_hash(hex.EncodeToString(block.Header().Hash().Bytes()));
		if (parent_id!=-1) { // we found our first valid parent
			log.Info(fmt.Sprintf("EthBot: fixing chain split from block %v",parent_block_num))
			bot.repair_chain_split_insert_blocks(parent_block_num,last_block_num)
			return
		}
		block_num--
		block=chain.GetBlockByNumber(uint64(block_num))
		if (block==nil) {
			utils.Fatalf("correct block wasnt found , block_num=",block_num)
		}
	}
}
func (bot *EthBot_t) repair_chain_split_insert_blocks(from_num Block_num_t,to_num Block_num_t) {
	chain:=bot.ethereum.BlockChain()
	i:=from_num
	for ;i<=to_num;i++ {
		block:=chain.GetBlockByNumber(uint64(i))
		if (block!=nil) {
			err:=bot.trace_block(bot.ethereum,block)
			if (err!=nil) {
				log.Error(fmt.Sprintf("EthBot: repairing chain split: error found inserting block %v :  %v",i,err))
			}
		} else {
			log.Error(fmt.Sprintf("EthBot: can't get block number %v",i))
		}
	}
}
func (bot *EthBot_t) Start(server *p2p.Server) error {
    log.Info("EthBot: Starting EthBot.")
    bot.server = server
	init_postgres()
    return nil
}
func (bot *EthBot_t) Stop() error {
    log.Info("EthBot: Stopping EthBot.")
	if (bot.head_sub!=nil) {
	    bot.head_sub.Unsubscribe()
	}
    return nil
}
func (bot *EthBot_t) APIs() []rpc.API {
    return []rpc.API{{
			Namespace: "ethbot",
			Version:	"1.0",
			Service:	&EthBotAPI {
				bot: bot,
			},
			},
	}
}
func (bot *EthBot_t) Protocols() ([]p2p.Protocol) {
    return []p2p.Protocol{}
}
func (bot *EthBot_t) get_account_value_transfers(acct_addr string) string {
	return rpc_get_account_value_transfers(acct_addr);
}
func (bot *EthBot_t) check_export_on_startup(ctx *cli.Context) {
    if ctx.GlobalIsSet(utils.NoExportFlag.Name) {
		// --noexport flag used, we skip exporting to SQL at startup
		log.Info("EthBot: export not launched as requested by user");
	} else {
		go bot.blockchain_export_start(-1,-1) // start from the last_block previously recorded and after importing everything enter into listening mode
	}
}
func (bot *EthBot_t) blockchain_export_status(exp *Export_t) *otto.Object {
	jsre:=console_obj.JSRE()
	vm:=jsre.VM()
	obj_str:=fmt.Sprintf(`({"starting_block":%d,"current_block":%d,"ending_block":%d,"direction":%d,"listening_mode":%v})`,exp.Starting_block,exp.Cur_block_num,exp.Ending_block,exp.Direction,exp.Listening_mode);
	object, err := vm.Object(obj_str)
	if (err!=nil) {
		utils.Fatalf("Failed to create object in Javascript VM for blockchain export status: %v",err)
	}
	return object
}
func (bot *EthBot_t) empty_object() *otto.Object {
	jsre:=console_obj.JSRE()
	vm:=jsre.VM()
	object, err := vm.Object("({})")
	if (err!=nil) {
		utils.Fatalf("Failed to create object in Javascript VM for blockchain export status: %v",err)
	}
	return object
}
func js_local_blockchain_export_status() *otto.Object {
	return ethbot_instance.blockchain_export_status(&ethbot_instance.export)
}
func js_remote_blockchain_export_status() *otto.Object {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return ethbot_instance.empty_object()
	}
	var result Export_t
	err:=remote_EthBot.Call(&result,"ethbot_blockchainexportstatus");
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_blockchainexportstatus","error",err)
		return ethbot_instance.empty_object()
	} else {
		return ethbot_instance.blockchain_export_status(&result)
	}
}
func js_local_blockchain_export_stop() otto.Value {
	if ethbot_instance.blockchain_export_stop() {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_blockchain_export_stop() otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err:=remote_EthBot.Call(&result,"ethbot_blockchainexportstop");
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_blockchainexportstop","error",err)
		return otto.FalseValue()
	} else {
		if result==true {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_blockchain_export_start(arg_starting_block otto.Value,arg_ending_block otto.Value) otto.Value {

	p_starting_block,p_ending_block,err:=check_input_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	err=ethbot_instance.blockchain_export_start(p_starting_block,p_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	return otto.TrueValue();
}
func js_remote_blockchain_export_start(arg_starting_block otto.Value,arg_ending_block otto.Value) otto.Value {

	p_starting_block,p_ending_block,err:=check_input_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result error
	err=remote_EthBot.Call(&result,"ethbot_blockchainexportstart",p_starting_block,p_ending_block);
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_blockchainexportstart","error",err)
		return otto.FalseValue()
	} else {
		if result==nil {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func (bot *EthBot_t) export_block_range(starting_block Block_num_t,ending_block Block_num_t) bool {
	ethereum:=bot.ethereum;

	log.Info("EthBot: this function has not been finished yet. Wait for beta version of EthBot.");
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
		bot.trace_block(ethereum, bptr);
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
func js_local_export_block_range(arg_starting_block otto.Value,arg_ending_block otto.Value) otto.Value {

	starting_block,ending_block,err:=check_input_export_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	ret:=ethbot_instance.export_block_range(starting_block,ending_block)
	if ret {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_export_block_range(arg_starting_block otto.Value,arg_ending_block otto.Value) otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	starting_block,ending_block,err:=check_input_export_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	var result bool
	err=remote_EthBot.Call(&result,"ethbot_exportblockrange",starting_block,ending_block);
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_exportblockrange","error",err)
		return otto.FalseValue()
	} else {
		if result {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_update_main_stats(p_block_num otto.Value) otto.Value {
	var err error
	block_num,err:=p_block_num.ToInteger()
	if (err!=nil) {
		err_text:="EthBot: invalid input value for `block_num` parameter: positive integer values or -1 are allowed"
		log.Error(err_text)
		return otto.FalseValue()
	}
	if ethbot_instance.update_main_stats(int(block_num)) {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_update_main_stats(p_block_num otto.Value) otto.Value {
	var err error
	block_num,err:=p_block_num.ToInteger()
	if (err!=nil) {
		err_text:="EthBot: invalid input value for `block_num` parameter: positive integer values or -1 are allowed"
		log.Error(err_text)
		return otto.FalseValue()
	}
	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}

	var result bool
	err=remote_EthBot.Call(&result,"ethbot_updatemainstats",block_num);
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method updatemainstats","error",err)
		return otto.FalseValue()
	} else {
		if result {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
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
func (bot *EthBot_t) fix_last_balances() bool {
	last_block_num:=sql_get_last_block_num();
	if (last_block_num<0) {
		log.Error("EthBot: last_block is < 0 (invalid value)")
		return false;
	}
	chain:=bot.ethereum.BlockChain()
	block:=chain.GetBlockByNumber(uint64(last_block_num))
	statedb, err := chain.StateAt(block.Root())
    if err != nil {
		log.Error(fmt.Sprintf("EthBot: StateAt(%v) failed",last_block_num))
		return false
	}
	accounts:=statedb.EthBotDump()
	return sql_fix_last_balances(&accounts);
}
func js_local_fix_last_balances() otto.Value {
	if ethbot_instance.fix_last_balances() {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_fix_last_balances() otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err:=remote_EthBot.Call(&result,"ethbot_fixlastbalances");
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_blockchainexportstop","error",err)
		return otto.FalseValue()
	} else {
		if result==true {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func (bot *EthBot_t) verify_last_balances() bool {
	last_block_num:=sql_get_last_block_num();
	if (last_block_num<0) {
		log.Error("EthBot: last_block is < 0 (invalid value)")
		return false;
	}
	chain:=bot.ethereum.BlockChain()
	block:=chain.GetBlockByNumber(uint64(last_block_num))
	statedb, err := chain.StateAt(block.Root())
    if err != nil {
		log.Error(fmt.Sprintf("EthBot: StateAt(%v) failed",last_block_num))
		return false
	}
	accounts:=statedb.EthBotDump()
	return sql_verify_last_balances(&accounts);
}
func js_local_verify_last_balances() otto.Value {
	if ethbot_instance.verify_last_balances() {
		return otto.TrueValue()
	} else {
		return otto.FalseValue()
	}
}
func js_remote_verify_last_balances() otto.Value {

	if (remote_EthBot==nil) {
		log.Error("EthBot: remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err:=remote_EthBot.Call(&result,"ethbot_verifylastbalances");
	if (err!=nil) {
		log.Error("EthBot: error calling RPC method ethbot_verifylastbalances","error",err)
		return otto.FalseValue()
	} else {
		if result==true {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
