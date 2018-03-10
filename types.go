package main
import (
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
	"encoding/hex"
)
const (
	VALTRANSF_UNKNOWN = iota
	VALTRANSF_GENESIS
	VALTRANSF_TRANSACTION
	VALTRANSF_TX_FEE
	VALTRANSF_BLOCK_REWARD
	VALTRANSF_CONTRACT_CREATION
	VALTRANSF_CONTRACT_TRANSACTION
	VALTRANSF_CONTRACT_SELFDESTRUCT
)
const (
	VERIFICATION_LEVELDB = iota
	VERIFICATION_SQL
)

const VERIFICATION_LOT_SIZE int = 50	// how many accounts go as parameter in IN() function of the WHERE clause when checking account balances
const VERIFICATION_NUM_THREADS int = 60	// how many parallel (concurrent) SQL queries to Postgres are going to issued to check for account balances

type Block_id_t int
type Block_num_t int
type Account_id_t int

// pre-INSERTed accounts in SQL database
const CONTRACT_CREATION_ACCOUNT_ID Account_id_t =	-2			// this is the account meaning a contract is created
const NONEXISTENT_ADDRESS_ACCOUNT_ID Account_id_t =	-1		// this is the default account_id for unexistent address (not 0x0000000000000000000000000000000000000000 address, but completely unexistent address, which means money deposited from this address is created from nothing) NONEXISTENT account is different from ZEROed (all zeros) account because ZEROed account can receive money but NONEXISTENT can not
const ZERO_ADDRESS_ACCOUNT_ID Account_id_t =		 1				// this is the account with all zeros

type  EthBot_t struct {
	ethereum				*eth.Ethereum				// eth.Ethereum object which holds the APIs
	ethbot_db				ethdb.Database				// database to store EthBot's variables used in the process of importing blocks
	blocks					chan *types.Block
	head_ch					chan<- core.ChainHeadEvent
	head_sub				event.Subscription
	server					*p2p.Server
	eb_accounts				map[common.Address]Account_id_t
	verification			Verification_t
	export					Export_t
}
type Value_transfer_t struct {
    depth           int
	block_id		Block_id_t
	block_num		Block_num_t
    src             *common.Address
    dst             *common.Address
    src_balance     *big.Int
    dst_balance     *big.Int
    value           *big.Int
	kind			int
}
type Acct_history_t struct{
	block_id		Block_id_t
	block_num		Block_num_t
	valtr_id		int64
	from_id			Account_id_t
	to_id			Account_id_t
	from_balance	string
	to_balance		string
	value			string
}
type Verification_t struct {
	Mode				int
	Starting_block		Block_num_t
	Ending_block		Block_num_t
	Direction			int
	In_progress			bool
	Current_block_num	Block_num_t
	Failed				bool
	Failing_valtr_id	int64
	Error_str			string
	User_cancelled		bool
	Threads_counter		int
	Finished_threads_counter int
	Num_accounts		int
	Num_processed		int
}
type Export_t struct {
	Starting_block	Block_num_t
	Ending_block	Block_num_t
	Cur_block_num	Block_num_t	// current block counter
	Exported_block	Block_num_t // block number that has been already successfuly exported
	Direction		int			// are we increasing or decreasing the current block counter
	Range_export	bool		// true if range export function call has been made
	User_cancelled	bool		// true if the user has issued a cancel request for this export
	In_progress		bool
	Listening_mode	bool		// true if we have to enter into listening mode (after all the blocks in local DB has been exported) , to export incoming blocks as they arrive
	Verify			bool		// true if we have to verify blockchain balances with SQL after processing each block
}
type EthBotAPI struct {
	bot			*EthBot_t
}
type Stack_frame_t struct {
    op				vm.OpCode
    acct_addr		common.Address
    transfers       []*Value_transfer_t
}
func get_hex_addr(addr *common.Address) string {
	address_string:=hex.EncodeToString(addr.Bytes());
	return address_string
}
