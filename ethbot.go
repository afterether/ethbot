package main

import (
	"fmt"
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/eth"
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

    db, err := ctx.OpenDatabase("ethbot_db", 16, 16)
    if err != nil {
        return nil, err
    }
    ethbot_instance=&EthBot_t {
        ethereum:				ethereum,
        ethbot_db:				db,
		head_ch:				nil,
        head_sub:				nil,
		eb_accounts:			make(map[common.Address]Account_id_t),
		blocks:					make(chan *types.Block),
		verification:			Verification_t{},
		export:					Export_t{},
    }
	return ethbot_instance, nil
}
func (bot *EthBot_t) getInt(key string) (uint64, error) {
    data, err := bot.ethbot_db.Get([]byte(key))
    if err != nil {
        return 0, err
    }

    var value uint64
    err = binary.Read(bytes.NewReader(data), binary.LittleEndian, &value)
    if err != nil {
        return 0, err
    }

    return value, nil
}
func (bot *EthBot_t) putInt(key string, value uint64) error {
    buf := new(bytes.Buffer)
    err := binary.Write(buf, binary.LittleEndian, value)
    if err != nil {
        return err
    }
    return bot.ethbot_db.Put([]byte(key), buf.Bytes())
}
func (bot *EthBot_t) get_last_block() uint64 {
    last_block, err := bot.getInt("last_block")
    if err != nil {
        return 0
    }
    return last_block
}
func (bot *EthBot_t) put_last_block(block uint64) {
    bot.putInt("last_block", block)
}
func (bot *EthBot_t) adjust_starting_block_num(starting_block Block_num_t) Block_num_t {
	if (starting_block==-1) {	// -1 here means we have to get last block processed in previous session
		exists,err:=bot.ethbot_db.Has([]byte("last_block"));
		if (err!=nil) {
			utils.Fatalf("Error at Has() querying ethbot_db")
		}
		if (exists) {
			starting_block = Block_num_t(bot.get_last_block())
		} else {
			starting_block=0;
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
		block:=chain.GetBlockByNumber(uint64(bot.export.Cur_block_num))
		if block!=nil {
			bot.blocks<-block
		} else {
			utils.Fatalf("null block received in export_blocks()")
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
	log.Info("export_blocks() over")
	// at this point we processed all the blocks up to the ending_block
	// so, if user specified listening mode, enter into it and continue exporting incoming blocks
	if (bot.export.Listening_mode) {
		log.Info("EthBot: entering listening mode, to export blocks as they arrive")
		return						// listening mode was not set, exit
	}
	bot.blocks<-nil				// tell process_blocks() go routine to exit
}
func (bot *EthBot_t) blockchain_export_start(starting_block Block_num_t,ending_block Block_num_t,verify bool) error {

	if (bot.export.In_progress) {
		log.Error("Export already in progress");
		return errors.New("Export already in progress");
	}
	starting_block=bot.adjust_starting_block_num(starting_block)

	bot.export.In_progress=true
	bot.export.Range_export=false
	bot.export.Starting_block=starting_block
	bot.export.Ending_block=ending_block
	bot.export.Listening_mode=false
	bot.export.Verify=verify
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
	go bot.export_blocks()

	return nil
}
func (bot *EthBot_t) blockchain_export_stop() bool {

	if (!bot.export.In_progress) {
		log.Error("Export process is not running")
		return false
	}
	bot.blocks<-nil
	return true
}
func (bot *EthBot_t) process_blocks() {

  again:

	var block *types.Block
    block = <-bot.blocks
	if (block==nil) { // nil means Exit
		bot.export.In_progress=false
		return
	}
	block_num:=Block_num_t(block.Number().Uint64())
	err := bot.trace_block(bot.ethereum, block)
    if err != nil {
       log.Info("EthBot: Unable to trace transactions","block", block.Number().Uint64(),"err",err)
       goto again;
    }

	if (err == nil) {
		if (bot.export.Range_export) {
			// range export does not update the `last_block` counter, since it is an "out of order" export process
		} else {
			bot.put_last_block(uint64(block_num))
			bot.export.Exported_block=block_num
		}
	}
	if (bot.export.Verify) {
		log.Info(fmt.Sprintf("Ethbot: verifying block number %v",block_num))
		vres:=verify_SQL_data(0,block_num,block_num)
		if (!vres) {
			log.Error("EthBot: verification failed, exiting")
			bot.blockchain_export_stop()
		}
	}
	goto again
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
func (ebapi *EthBotAPI) Verificationstatus() Verification_t {
	return ethbot_instance.verification
}
func (ebapi *EthBotAPI) Blockchainexportstatus() Export_t {
	log.Info("Sending export struct");
	return ethbot_instance.export
}
func (ebapi *EthBotAPI) Verifysqldata(mode int,starting_block Block_num_t, ending_block Block_num_t) bool {
	return verify_SQL_data(mode,starting_block,ending_block)
}
func (ebapi *EthBotAPI) Stopverification() {
	stop_verification()
}
func (ebapi *EthBotAPI) Verifyaccount(account_addr_str string,block_num Block_num_t) bool {
	return verify_single_account(account_addr_str,block_num)
}
func (ebapi *EthBotAPI) Verifyallaccounts(block_num Block_num_t) bool {
	return verify_all_accounts(block_num)
}
func (ebapi *EthBotAPI) Exportblockrange(starting_block Block_num_t,ending_block Block_num_t) bool {
	return ethbot_instance.export_block_range(starting_block,ending_block)
}
func (ebapi *EthBotAPI) Blockchainexportstart(starting_block Block_num_t,ending_block Block_num_t,verify bool) error {
	return ethbot_instance.blockchain_export_start(starting_block,ending_block,verify)
}
func (ebapi *EthBotAPI) Blockchainexportstop() bool {
	return ethbot_instance.blockchain_export_stop()
}
func (bot *EthBot_t) APIs() []rpc.API {
    return []rpc.API{{
		Namespace: "ethbot",
		Version:	"1.0",
		Service:	&EthBotAPI {
			bot: bot,
		},
	}}
}
func (bot *EthBot_t) Protocols() ([]p2p.Protocol) {
    return []p2p.Protocol{}
}
func (bot *EthBot_t) check_export_on_startup(ctx *cli.Context) {
    if ctx.GlobalIsSet(utils.NoExportFlag.Name) {
		// --noexport flag used, we skip exporting to SQL at startup
		log.Info("EthBot: export not launched as requested by user");
	} else {
		go bot.blockchain_export_start(-1,-1,false) // start from the last_block previously recorded and after importing everything enter into listening mode
	}
}
func (bot *EthBot_t) blockchain_export_status(exp *Export_t) *otto.Object {
	jsre:=console_obj.JSRE()
	vm:=jsre.VM()
	obj_str:=fmt.Sprintf(`({"starting_block":%d,"current_block":%d,"ending_block":%d,"direction":%d,"listening_mode":%v,"verify":%v})`,exp.Starting_block,exp.Cur_block_num,exp.Ending_block,exp.Direction,exp.Listening_mode,exp.Verify);
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
		log.Error("remote end for RPC not initalized")
		return ethbot_instance.empty_object()
	}
	var result Export_t
	err:=remote_EthBot.Call(&result,"ethbot_blockchainexportstatus");
	if (err!=nil) {
		log.Error("Error calling RPC method ethbot_blockchainexportstatus","error",err)
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
		log.Error("remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result bool
	err:=remote_EthBot.Call(&result,"ethbot_blockchainexportstop");
	if (err!=nil) {
		log.Error("Error calling RPC method ethbot_blockchainexportstop","error",err)
		return otto.FalseValue()
	} else {
		if result==true {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
func js_local_blockchain_export_start(arg_starting_block otto.Value,arg_ending_block otto.Value,arg_verify otto.Value) otto.Value {

	p_starting_block,p_ending_block,err:=check_input_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	var verify bool
	verify,err=arg_verify.ToBoolean()
	if (err!=nil) {
		log.Error("Invalid input for 'verify' parameter'")
		return otto.FalseValue()
	}
	err=ethbot_instance.blockchain_export_start(p_starting_block,p_ending_block,verify)
	if (err!=nil) {
		return otto.FalseValue()
	}
	return otto.TrueValue();
}
func js_remote_blockchain_export_start(arg_starting_block otto.Value,arg_ending_block otto.Value,arg_verify otto.Value) otto.Value {

	p_starting_block,p_ending_block,err:=check_input_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	var verify bool
	verify,err=arg_verify.ToBoolean()
	if (err!=nil) {
		log.Error("Invalid input for 'verify' parameter'")
		return otto.FalseValue()
	}
	if (remote_EthBot==nil) {
		log.Error("remote end for RPC not initalized")
		return otto.FalseValue()
	}
	var result error
	err=remote_EthBot.Call(&result,"ethbot_blockchainexportstart",p_starting_block,p_ending_block,verify);
	if (err!=nil) {
		log.Error("Error calling RPC method ethbot_blockchainexportstart","error",err)
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

	if (bot.export.In_progress) {
		log.Info("Export already in progress");
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
			log.Info("Range export cancelled by user")
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
		log.Error("remote end for RPC not initalized")
		return otto.FalseValue()
	}
	starting_block,ending_block,err:=check_input_export_block_range(arg_starting_block,arg_ending_block)
	if (err!=nil) {
		return otto.FalseValue()
	}
	var result bool
	err=remote_EthBot.Call(&result,"ethbot_exportblockrange",starting_block,ending_block);
	if (err!=nil) {
		log.Error("Error calling RPC method ethbot_exportblockrange","error",err)
		return otto.FalseValue()
	} else {
		if result {
			return otto.TrueValue()
		} else {
			return otto.FalseValue()
		}
	}
}
