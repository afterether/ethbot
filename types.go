package main
import (
	"sync"
	"errors"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
	"database/sql"
)
const (
	VALTRANSF_UNKNOWN = iota
	VALTRANSF_GENESIS							//	1
	VALTRANSF_TRANSACTION						//	2
	VALTRANSF_TX_FEE							//	3
	VALTRANSF_BLOCK_REWARD						//	4
	VALTRANSF_CONTRACT_CREATION					//	5
	VALTRANSF_CONTRACT_TRANSACTION				//	6
	VALTRANSF_CONTRACT_SELFDESTRUCT				//	7
	VALTRANSF_FORK								//  8
)
const (
	TOKENOP_UNKNOWN = iota
	TOKENOP_TRANSFER							//  1
	TOKENOP_APPROVAL							//  2
	TOKENOP_TRANSFER_FROM						//	3
	TOKENOP_BURN								//  4
	TOKENOP_MINT								//  4
	TOKENOP_FREEZE								//  5
	TOKENOP_UNFREEZE							//  7
)
const (
	TOKIFACE_UNKNOWN = iota
	TOKIFACE_ERC20
	TOKIFACE_ERC223
	TOKIFACE_ERC677
	TOKIFACE_ERC721
	TOKIFACE_ERC777
	TOKIFACE_ERC827
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
const VERIFICATION_LOT_SIZE int = 64// how many accounts a single goroutine should verify, this is supposed to run in parallel, but not yet
const VERIFICATION_NUM_THREADS int = 256// how many parallel (concurrent) SQL queries to Postgres are going to issued to check for account balances. each thread spawns a separate goroutine

//type Block_id_t int
type Block_num_t int
type Account_id_t int

// pre-INSERTed accounts in SQL database
const REFUNDS_ACCOUNT_ID= -3									// Refunds returned to the sender if he releases storage from the state
const CONTRACT_CREATION_ACCOUNT_ID Account_id_t =	-2			// this is the account meaning a contract is created
const NONEXISTENT_ADDRESS_ACCOUNT_ID Account_id_t =	-1		// this is the default account_id for unexistent address (not 0x0000000000000000000000000000000000000000 address, but completely unexistent address, which means money deposited from this address is created from nothing) NONEXISTENT account is different from ZEROed (all zeros) account because ZEROed account can receive money but NONEXISTENT can not
const ZERO_ADDRESS_ACCOUNT_ID Account_id_t =		 1				// this is the account with all zeros

type  EthBot_t struct {
	ethereum				*eth.Ethereum				// eth.Ethereum object which holds the APIs
	blocks					chan *types.Block
	head_ch					chan core.ChainHeadEvent
	process_started			bool						// true if process_blocks() go routine is already running
	token_process_started	bool						// true if process_blocks() go routine is already running
	listening_started		bool						// true if listen4blocks() go routine is already running
	head_sub				event.Subscription
	server					*p2p.Server
	eb_accounts				map[common.Address]Account_id_t
	verification			Verification_t
	export					Export_t
	token_export			TokenExport_t
	newTXs_evt_sub			event.Subscription
	newTXs_ch				chan core.NewTxsEvent
}
type Export_t struct {
	Starting_block			Block_num_t	// range of blocks to export from
	Ending_block			Block_num_t
	Cur_block_num			Block_num_t	// current block counter
	Exported_block			Block_num_t // block number that has been already successfuly exported
	Last_block_num			Block_num_t	// current latest block in SQL db
	Direction				int			// are we increasing or decreasing the current block counter
	Range_export			bool		// true if range export function call has been made
	User_cancelled			bool		// true if the user has issued a cancel request for this export
	In_progress				bool
	Listening_mode			bool		// true if we have to enter into listening mode (after all the blocks in local DB has been exported) , to export incoming blocks as they arrive
	Alarms_on				bool		// verify balance and nonces of accounts involved in processed block (comparing SQL vs LevelDB) and alarm on mismatch
	blockchain_balance		*big.Int	// balance for NONEXISTENT_ADDRESS_ACCOUNT_ID (all the Ethers created so far)
}
type TokenExport_t struct {
	blocks					chan *types.Block
	head_ch					chan core.ChainHeadEvent
	head_sub				event.Subscription
	Starting_block			Block_num_t	// range of blocks to export from
	Ending_block			Block_num_t
	Cur_block_num			Block_num_t	// current block counter
	Exported_block			Block_num_t // block number that has been already successfuly exported
	Direction				int			// are we increasing or decreasing the current block counter
	Range_export			bool		// true if range export function call has been made
	User_cancelled			bool		// true if the user has issued a cancel request for this export
	In_progress				bool
	Listening_mode			bool		// true if we have to enter into listening mode (after all the blocks in local DB has been exported) , to export incoming blocks as they arrive
	listening_started		bool		// true if listen4token_blocks() go routine is already running
}
type Acct_history_t struct{
	block_num		Block_num_t
	kind			int
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
type Acache_t struct {		// account cache entry
	account_id		Account_id_t
	deleted			int16
}
type EthBotAPI struct {
	bot			*EthBot_t
}
type SimAPI struct {
	bot			*EthBot_t
}
type Simulation_result_t struct {
	Status					int
	From_address			string
	To_address				string
	From_old_balance		string
	To_old_balance			string
	From_new_balance		string
	To_new_balance			string
	From_diff				string
	To_diff					string
	Value					string
	Tok_from_address		string
	Tok_to_address			string
	Tok_from_old_balance	string
	Tok_from_new_balance	string
	Tok_from_diff			string
	Tok_to_old_balance		string
	Tok_to_new_balance		string
	Tok_to_diff				string
	Tok_value				string
	Tok_ret_val				bool
	Signed_tx				string
	Gas_used				string
	Gas_limit				string
	Gas_price				string
	Gas_price_threshold		string
	Error					string
	VMError					string
	Token_transfer			bool		// true if token transfer was detected
	New_acct				bool		// flag if the destination account doesn't exist (flag usefull for detecting wrong address or first transfer to the account)
}
type Token_transf_t struct {
	From					common.Address
	To						common.Address
	Value					*big.Int
	Kind					int
	Non_compliant			bool
	Non_compliance_str		string
}
type Token_t struct {
	contract_addr			*common.Address
	total_supply			*big.Int
	tx_id					int64
	decimals				int32
	non_fungible			bool
	name					string
	symbol					string
}
type Token_info_t struct {
	Contract_id						Account_id_t
	block_created					Block_num_t
	fully_discovered				bool			// true if during contract creation we were able to discover all the methods of the token , false if we have to re-discover again, when first event arrives
	i_ERC20							bool
	i_burnable						bool
	i_mintable						bool
	i_ERC223						bool
	i_ERC677						bool
	i_ERC721						bool
	i_ERC777						bool
	i_ERC827						bool
	nc_ERC20						bool
	nc_ERC223						bool
	nc_ERC677						bool
	nc_ERC721						bool
	nc_ERC777						bool
	nc_ERC827						bool
	m_ERC20_name					bool
	m_ERC20_symbol					bool
	m_ERC20_decimals				bool
	m_ERC20_total_supply			bool
	m_ERC20_balance_of				bool
	m_ERC20_allowance				bool
	m_ERC20_transfer				bool
	m_ERC20_approve					bool
	m_ERC20_transfer_from			bool
	m_ERC20_ext_burn				bool
	m_ERC20_ext_mint				bool
	m_ERC20_ext_freeze				bool
	m_ERC20_ext_unfreeze			bool
	m_ERC721_owner_of				bool
	m_ERC721_take_ownership			bool
	m_ERC721_token_by_index			bool
	m_ERC721_token_metadata			bool
	m_ERC777_default_operators		bool
	m_ERC777_is_operator_for		bool
	m_ERC777_authorize_operator		bool
	m_ERC777_revoke_operator		bool
	m_ERC777_send					bool
	m_ERC777_operator_send			bool
	m_ERC777_burn					bool
	m_ERC777_operator_burn			bool
	e_ERC20							string
	e_ERC223						string
	e_ERC677						string
	e_ERC721						string
	e_ERC777						string
	e_ERC827						string
	name							string
	symbol							string
}
type ERC20_transfer_method_params_t struct {
	to						common.Address
	value					*big.Int
}
type Last_block_info_t struct {
	Block_number		uint64	`json: "block_number"`
	Num_transactions	int		`json: "num_transactions"`
}
type Event_t struct {
	Event_id		int64
	Topics			[]common.Hash
	Data			[]byte
}
type Topics_t struct {
	Topics			[]common.Hash
}
type VT_insert_info_t struct {
	valtr_id		int64
	account_id		Account_id_t
}
type Verif_acct_param_t struct {
	balance			big.Int
	account_id		Account_id_t
//	kind			int	
	addr			common.Address
}

const alarms_dir string =`/var/tmp/ethbot/`
const debug_sql_exec_time = false
const debug_exec_time = false
const vt_debug = false
const dump_VTs_to_file_flag bool = false
const dump_VTs_dir string = "/var/tmp/vt_dump"
var (// Global vars
    frontierBlockReward  *big.Int = big.NewInt(5e+18)
    byzantiumBlockReward *big.Int = big.NewInt(3e+18)
    maxUncles                     = 2
	previous_state		*state.StateDB
	accounts_cache		map[common.Address]Account_id_t=make(map[common.Address]Account_id_t)
	ErrAncestorNotFound		error = errors.New("Lookup of parent block by hash failed")
	new_tokens					int32 = 0
	per_block_VT_counter		int = 0
	ac_lock						= sync.RWMutex{}
)
var erc_20_token_abi_json_str string = `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"tokens","type":"uint256"}],"name":"approve","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"tokens","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"tokenOwner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"tokens","type":"uint256"}],"name":"transfer","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"tokenOwner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"remaining","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"tokens","type":"uint256"}],"name":"Transfer","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"tokenOwner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"tokens","type":"uint256"}],"name":"Approval","type":"event"}]`
var erc20_approval_event_signature []byte
var erc20_transfer_event_signature []byte
var erc20_transfer_method_signature []byte
var erc20_approve_method_signature []byte
var erc20_transfer_from_method_signature []byte
var erc20_methods_abi_json_str string = `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"tokens","type":"uint256"}],"name":"approve","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"tokens","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"tokenOwner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"tokens","type":"uint256"}],"name":"transfer","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"tokenOwner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"remaining","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`
var erc20_token_abi_str string =`[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transfer","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"},{"name":"","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"}]`
var (
	ethbot_instance *EthBot_t
	erc20_token_abi	abi.ABI
	erc20_events_abi	abi.ABI
	erc20_methods_abi	abi.ABI
)
var db *sql.DB
var zero *big.Int=big.NewInt(0)
var wg sync.WaitGroup
