package main
import (
	"github.com/ethereum/go-ethereum/eth"
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
const (
	JSON_OBJ_TYPE_UNDEFINED = iota
	JSON_OBJ_TYPE_BLOCK
	JSON_OBJ_TYPE_ACCOUNT
	JSON_OBJ_TYPE_TRANSACTION
	JSON_OBJ_TYPE_VALUE_TRANSFER
	JSON_OBJ_TYPE_UNCLE
)
const VERIFICATION_LOT_SIZE int = 50	// how many accounts go as parameter in IN() function of the WHERE clause when checking account balances
const VERIFICATION_NUM_THREADS int = 60	// how many parallel (concurrent) SQL queries to Postgres are going to issued to check for account balances

type Block_id_t int
type Block_num_t int
type Account_id_t int

// pre-INSERTed accounts in SQL database
const WRONG_TRANSACTION_SENDER_ACCOUNT_ID = -3			// if a transaction can't be verified, this address is used in `From` field
const CONTRACT_CREATION_ACCOUNT_ID Account_id_t =	-2			// this is the account meaning a contract is created
const NONEXISTENT_ADDRESS_ACCOUNT_ID Account_id_t =	-1		// this is the default account_id for unexistent address (not 0x0000000000000000000000000000000000000000 address, but completely unexistent address, which means money deposited from this address is created from nothing) NONEXISTENT account is different from ZEROed (all zeros) account because ZEROed account can receive money but NONEXISTENT can not
const ZERO_ADDRESS_ACCOUNT_ID Account_id_t =		 1				// this is the account with all zeros

type  EthBot_t struct {
	ethereum				*eth.Ethereum				// eth.Ethereum object which holds the APIs
	blocks					chan *types.Block
	head_ch					chan core.ChainHeadEvent
	process_started			bool						// true if process_blocks() go routine is already running
	listening_started		bool						// true if listen4blocks() go routine is already running
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
	err_str			string
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
	Starting_block			Block_num_t	// range of blocks to export from
	Ending_block			Block_num_t
	Cur_block_num			Block_num_t	// current block counter
	Exported_block			Block_num_t // block number that has been already successfuly exported
	Direction				int			// are we increasing or decreasing the current block counter
	Range_export			bool		// true if range export function call has been made
	User_cancelled			bool		// true if the user has issued a cancel request for this export
	In_progress				bool
	Listening_mode			bool		// true if we have to enter into listening mode (after all the blocks in local DB has been exported) , to export incoming blocks as they arrive
	non_existent_balance	*big.Int	// balance for NONEXISTENT_ADDRESS_ACCOUNT_ID

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
type Json_value_transfer_t struct {
	Valtr_id		int64	`json: "valtr_id"		gencodec:"required"`
	Block_num		int		`json: "block_num"		gencodec:"required"`
	Block_id		int64	`json: "block_id"		gencodec:"required"`
	From_id			int64	`json: "from_id"		gencodec:"required"`
	To_id			int64	`json: "to_id"			gencodec:"required"`
	From_addr		string	`json: "from_addr"		gencodec:"required"`
	To_addr			string	`json: "to_addr"		gencodec:"required"`
	From_balance	string	`json: "from_balance"	gencodec:"required"`
	To_balance		string	`json: "to_balance"		gencodec:"required"`
	Value			string	`json: "value"			gencodec:"required"`
	Kind			int		`json: "kind"			gencodec:"rquired"`
	Tx_id			int64	`json: "tx_id"			gencodec:"required"`
	Tx_hash			string	`json: "tx_hash"		gencodec:"required"`
	Direction		int		`jsoin:"direction"		gencodec:"required"`
	Error			string	`json: "error"			gencodec:"required"`
}
type Json_transaction_t  struct {
	Tx_id			int64
	From_id			int		`json: "from_id"		gencodec:"required"`
	To_id			int		`json: "to_id"			gencodec:"required"`
	From_addr		string	`json: "from_addr"		gencodec:"required"`
	To_addr			string	`json: "to_addr"		gencodec:"required"`
	Value			string	`json: "value"			gencodec:"required"`
	Tx_hash			string	`json: "tx_hash"		gencodec:"required"`
	Gas_limit		string	`json: "gas_limit"		gencodec:"required"`
	Gas_used		string	`json: "gas_used"		gencodec:"required"`
	Gas_price		string	`json: "gas_price"		gencodec:"required"`
	Cost			string	`json: "cost"			gencodec:"required"`
	Nonce			int		`json: "nonce"			gencodec:"required"`
	Block_id		int		`json: "block_id"		gencodec:"required"`
	Block_num		int		`json: "block_num"		gencodec:"required"`
	Tx_index		int		`json: "tx_index"		gencodec:"required"`
	Tx_status		int		`json: "tx_status"		gencodec:"required"`
	Confirmations	int		`json: "confirmations"	gencodec:"required"`
	V				string	`json: "v"				gencodec:"required"`
	R				string	`json: "r"				gencodec:"required"`
	S				string	`json: "s"				gencodec:"required"`
	Tx_error		string	`json: "tx_error"		gencodec:"required"`
}
type Json_block_t struct {
	Number				uint64	`json: "number"`
	Parent_num			uint64	`json: "number"`
	Hash				string	`json: "hash"`
	Confirmations		uint64	`json: "confirmations"`
	Timestamp			uint64	`json: "timestamp"`
	Miner				string	`jsoin: "miner"`
	Num_transactions	int		`json: "transactions"`
	Difficulty			string	`json: "difficulty"`
	Total_difficulty	string	`json: "total_difficulty"`
	Gas_used			uint64	`json: "gas_used"`
	Gas_limit			uint64	`json: "gas_limit"`
	Size				float64	`json: "block_size"`
	Nonce				uint64	`json: "nonce"`
	Parent_hash			string	`json: "parent_hash"`
	Sha3uncles			string	`json: "sha3uncles"`
	Extra_data			string	`json: "extra_data"`
}
type Json_uncles_t struct {
	Block_num			int					`json: "block_num"`
	Num_uncles			int					`json: "num_uncles"`
	Uncle1				Json_block_t		`json: "uncle1"`
	Uncle2				Json_block_t		`json: "uncle2"`
}
type Json_main_stats_t struct {
	Hash_rate			string	`json: "hash_rate"`
	Block_time			string	`json: "block_time"`
	Tx_per_block		string	`json: "tx_per_block"`
	Gas_price			string	`json: "gas_price"`
	Tx_cost				string	`json: "tx_cost"`
	Supply				string	`json: "supply"`
	Difficulty			string	`json: "difficulty"`
	Last_block			int		`json: "last_block"`
}
type Json_search_result_t struct {
	Object_type		int
	Block			Json_block_t
	Transaction		Json_transaction_t
	Value_transfers	[]Json_value_transfer_t
	Search_text		string
	Account_balance	string
	Account_id		int
	Vt_set			Json_vt_set_t
}
type Json_vt_set_t struct {
	Account_address	string
	Account_balance	string
	Account_id		int
	Offset			int
	Limit			int
	Value_transfers	[]Json_value_transfer_t
}
type Json_tx_set_t struct {
	Account_address	string
	Account_balance	string
	Account_id		int
	Offset			int
	Limit			int
	Transactions	[]Json_transaction_t
}
type Last_block_info_t struct {
	Block_number		uint64	`json: "block_number"`
	Num_transactions	int		`json: "num_transactions"`
}

