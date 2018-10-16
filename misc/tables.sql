-- blockchain
CREATE TABLE account (
	account_id			SERIAL	 		PRIMARY KEY,
	owner_id			INT				NOT NULL DEFAULT 0,
	last_balance		NUMERIC			DEFAULT 0,
	num_tx				BIGINT			DEFAULT 0,
	num_vt				BIGINT			DEFAULT 0,
	ts_created			INT				DEFAULT 0,
	block_created		INT				DEFAULT -1,
	deleted				SMALLINT		DEFAULT 0,
	block_sd			INT				DEFAULT -1,
	block_del			INT			DEFAULT -1,
	address				TEXT 			NOT NULL UNIQUE
);
CREATE TABLE block (
	parent_id			INT				NOT NULL,
	block_num 			INT		 	PRIMARY KEY,
	block_ts			INT				NOT NULL,
	miner_id			INT 			NOT NULL REFERENCES account(account_id) ON DELETE CASCADE,
	difficulty			NUMERIC			NOT NULL,
	total_dif			NUMERIC			NOT NULL,
	gas_limit			NUMERIC			NOT NULL,
	gas_used			NUMERIC			NOT NULL,
	nonce				NUMERIC			NOT NULL,
	block_hash 			TEXT			NOT NULL UNIQUE,
	uncle_hash			TEXT			NOT NULL,
	state_root			TEXT			NOT NULL,
	tx_hash				TEXT			NOT NULL,
	num_tx				int				NOT NULL DEFAULT 0,
	num_vt				int				NOT NULL DEFAULT 0,
	num_uncles			int				NOT NULL,
	size				int				NOT NULL,
	done				boolean			DEFAULT false,
	val_transferred		NUMERIC			NOT NULL DEFAULT 0,
	miner_reward		NUMERIC			NOT NULL DEFAULT 0,
	rcpt_hash			TEXT			NOT NULL,
	mix_hash			TEXT			NOT NULL,
	bloom				bytea			,
	extra				bytea
);
CREATE TABLE uncle (
	uncle_id			SERIAL 			PRIMARY KEY,
	block_num 			INT		 		NOT NULL,
	parent_num			INT			NOT NULL REFERENCES block(block_num) ON DELETE CASCADE,
	block_ts			INT				NOT NULL,
	miner_id			INT 			NOT NULL REFERENCES account(account_id) ON DELETE CASCADE,
	uncle_pos			SMALLINT 		NOT NULL,
	difficulty			NUMERIC			NOT NULL,
	total_dif			NUMERIC			NOT NULL,
	gas_limit			NUMERIC			NOT NULL,
	gas_used			NUMERIC			NOT NULL,
	nonce				NUMERIC			NOT NULL,
	block_hash 			TEXT			NOT NULL UNIQUE,
	uncle_hash			TEXT			NOT NULL,
	state_root			TEXT			NOT NULL,
	tx_hash				TEXT			NOT NULL,
	rcpt_hash			TEXT			NOT NULL,
	mix_hash			TEXT			NOT NULL,
	bloom				bytea			NOT NULL,
	extra				bytea			NOT NULL
);
CREATE TABLE transaction (
	tx_id				BIGSERIAL 		PRIMARY KEY,
	bnumtx				BIGINT			NOT NULL,
	from_id				INT 			NOT NULL REFERENCES account(account_id) ON DELETE CASCADE,
	to_id				INT 			NOT NULL REFERENCES account(account_id) ON DELETE CASCADE,
	gas_limit			INT 			NOT NULL,
	gas_used			INT				NOT NULL,
	tx_value			NUMERIC 		NOT NULL,
	gas_price			NUMERIC 		NOT NULL,
	nonce 				INT				NOT NULL,
	block_num			INT			NOT NULL REFERENCES block(block_num) ON DELETE CASCADE,
	tx_index			INT 			NOT NULL,
	tx_ts				INT 			NOT NULL,
	num_vt				INT				NOT NULL DEFAULT 0,
	val_transferred		NUMERIC			NOT NULL,
	v				NUMERIC			,
	r				NUMERIC			,
	s				NUMERIC			,
	tx_status			INT				DEFAULT 0,
	tx_hash 			TEXT			NOT NULL UNIQUE,
	tx_error			TEXT			DEFAULT '',
	vm_error			TEXT			DEFAULT '',
	payload				bytea
);
CREATE TABLE value_transfer (
	valtr_id			BIGSERIAL		PRIMARY KEY,
	tx_id				BIGINT			REFERENCES transaction(tx_id) ON DELETE CASCADE,
	bnumvt				BIGINT				NOT NULL,
	block_num			INT			NOT NULL REFERENCES block(block_num) ON DELETE CASCADE,
	from_id				INT				NOT NULL,
	to_id				INT				NOT NULL,
	depth				INT				DEFAULT 0,
	kind				SMALLINT		NOT NULL,
	value				NUMERIC			DEFAULT 0,
	from_balance		NUMERIC			DEFAULT 0,
	to_balance			NUMERIC			DEFAULT 0,
	gas_refund			NUMERIC			DEFAULT 0,
	error				TEXT			NOT NULL
);
CREATE TABLE vt_extras (
	valtr_id			BIGINT			PRIMARY KEY REFERENCES value_transfer(valtr_id) ON DELETE CASCADE,
	input				bytea			DEFAULT NULL,
	output				bytea			DEFAULT NULL
);
CREATE TABLE event (
	event_id			BIGSERIAL		PRIMARY KEY,
	valtr_id			BIGINT			NOT NULL REFERENCES value_transfer(valtr_id) ON DELETE CASCADE,
	tx_id				BIGINT			NOT NULL,
	block_num			INT			NOT NULL,
	contract_id			INT			NOT NULL,
	log_index			INT			NOT NULL,
	topics				TEXT			NOT NULL,
	data				bytea
);
CREATE TABLE last_block (
	block_num			INT				NOT NULL
);
-- tokens
CREATE TABLE tokens_processed (
	block_num			INT			PRIMARY KEY NOT NULL REFERENCES block(block_num) ON DELETE CASCADE,
	done				boolean			DEFAULT FALSE
);
CREATE TABLE tokacct (
	tx_id				BIGINT			NOT NULL,
	account_id			SERIAL	 		PRIMARY KEY,
	state_acct_id			INT			NOT NULL DEFAULT 0,		-- references `account.account_id`
	ts_created			INT			DEFAULT 0,
	block_created		INT				DEFAULT 0,
	address				TEXT 			NOT NULL
);
CREATE TABLE token (
	num_transfers			BIGINT			DEFAULT 0,
	contract_id			INT			PRIMARY KEY,
	block_created			INT			NOT NULL REFERENCES tokens_processed(block_num) ON DELETE CASCADE,
	created_tx_id			INT			NOT NULL,
	decimals			INT			NOT NULL,
	non_fungible		BOOLEAN			DEFAULT FALSE,
	total_supply			NUMERIC			NOT NULL,
	granularity			NUMERIC			DEFAULT 0,
	symbol				TEXT			NOT NULL,
	name				TEXT			NOT NULL
);
CREATE TABLE token_info (
	contract_id			INT			PRIMARY KEY,
	block_created			INT			NOT NULL REFERENCES tokens_processed(block_num) ON DELETE CASCADE,
	fully_discovered		BOOLEAN			DEFAULT FALSE,
	i_ERC20				BOOLEAN			DEFAULT FALSE,-- supported interfaces
	i_burnable			BOOLEAN			DEFAULT FALSE,
	i_mintable			BOOLEAN			DEFAULT FALSE,
	i_ERC223			BOOLEAN			DEFAULT FALSE,
	i_ERC677			BOOLEAN			DEFAULT FALSE,
	i_ERC721			BOOLEAN			DEFAULT FALSE,
	i_ERC777			BOOLEAN			DEFAULT FALSE,
	i_ERC827			BOOLEAN			DEFAULT FALSE,
	nc_ERC20			BOOLEAN			DEFAULT FALSE,-- non-compliancy
	nc_ERC223			BOOLEAN			DEFAULT FALSE,
	nc_ERC677			BOOLEAN			DEFAULT FALSE,
	nc_ERC721			BOOLEAN			DEFAULT FALSE,
	nc_ERC777			BOOLEAN			DEFAULT FALSE,
	nc_ERC827			BOOLEAN			DEFAULT FALSE,
	m_ERC20_name			BOOLEAN			DEFAULT FALSE,-- methods
	m_ERC20_symbol			BOOLEAN			DEFAULT FALSE,
	m_ERC20_decimals		BOOLEAN			DEFAULT FALSE,
	m_ERC20_total_supply		BOOLEAN			DEFAULT FALSE,
	m_ERC20_balance_of		BOOLEAN			DEFAULT FALSE,
	m_ERC20_allowance		BOOLEAN			DEFAULT FALSE,
	m_ERC20_transfer		BOOLEAN			DEFAULT FALSE,
	m_ERC20_approve			BOOLEAN			DEFAULT FALSE,
	m_ERC20_transfer_from		BOOLEAN			DEFAULT FALSE,
	m_ERC20_ext_burn		BOOLEAN			DEFAULT FALSE,
	m_ERC20_ext_mint		BOOLEAN			DEFAULT FALSE,
	m_ERC20_ext_freeze		BOOLEAN			DEFAULT FALSE,
	m_ERC20_ext_unfreeze		BOOLEAN			DEFAULT FALSE,
	m_ERC721_owner_of		BOOLEAN			DEFAULT FALSE,
	m_ERC721_take_ownership		BOOLEAN			DEFAULT FALSE,
	m_ERC721_token_by_index		BOOLEAN			DEFAULT FALSE,
	m_ERC721_token_metadata		BOOLEAN			DEFAULT FALSE,
	m_ERC777_default_operators	BOOLEAN			DEFAULT FALSE,
	m_ERC777_is_operator_for	BOOLEAN			DEFAULT FALSE,
	m_ERC777_authorize_operator	BOOLEAN			DEFAULT FALSE,
	m_ERC777_revoke_operator	BOOLEAN			DEFAULT FALSE,
	m_ERC777_send			BOOLEAN			DEFAULT FALSE,
	m_ERC777_operator_send		BOOLEAN			DEFAULT FALSE,
	m_ERC777_burn			BOOLEAN			DEFAULT FALSE,
	m_ERC777_operator_burn		BOOLEAN			DEFAULT FALSE,
	e_ERC20				text			DEFAULT '',   -- non-compliancy error message
	e_ERC223			text			DEFAULT '',
	e_ERC677			text			DEFAULT '',
	e_ERC721			text			DEFAULT '',
	e_ERC777			text			DEFAULT '',
	e_ERC827			text			DEFAULT '',
	name				text			DEFAULT '',
	symbol				text			DEFAULT ''
);
CREATE TABLE tokop (
	tokop_id			BIGSERIAL		PRIMARY KEY,
	tx_id				BIGINT			NOT NULL,
	approval_tx_id		BIGINT			DEFAULT 0,
	contract_id			INT			NOT NULL,
	block_num			INT			NOT NULL REFERENCES tokens_processed(block_num) ON DELETE CASCADE,
	block_ts			INT			NOT NULL,
	from_id				INT			NOT NULL,
	to_id				INT			NOT NULL,
	value				NUMERIC			DEFAULT 0,
	from_balance		NUMERIC			DEFAULT 0,
	to_balance			NUMERIC			DEFAULT 0,
	kind				SMALLINT		DEFAULT 0,
	non_fungible			BOOLEAN			DEFAULT FALSE,
	non_compliant			BOOLEAN			DEFAULT FALSE,
	non_compliance_err		TEXT			DEFAULT ''
);
CREATE TABLE approval (
	approval_id			BIGSERIAL		PRIMARY KEY,
	tx_id				BIGINT			NOT NULL,
	contract_id			INT			NOT NULL,
	block_num			INT			NOT NULL REFERENCES tokens_processed(block_num) ON DELETE CASCADE,
	block_ts			INT			NOT NULL,
	from_id				INT			NOT NULL,
	to_id				INT			NOT NULL,
	value				NUMERIC			NOT NULL,
	value_consumed			NUMERIC			DEFAULT 0,
	expired				BOOLEAN			DEFAULT FALSE,
	non_compliant			BOOLEAN			DEFAULT FALSE,
	non_compliance_err		TEXT			DEFAULT ''
);
CREATE TABLE tokop_approval (
	tokop_id			BIGINT			NOT NULL REFERENCES tokop(tokop_id) ON DELETE CASCADE,
	approval_id			BIGINT			NOT NULL REFERENCES approval(approval_id) ON DELETE CASCADE,
	-- the following fields are a copy of tokop.* because the trigger ON DELETE is executed after parent records are deleted and we have no way to know them
	contract_id			INT				DEFAULT 0,
	from_id				INT				DEFAULT 0,
	to_id				INT				DEFAULT 0,
	value				NUMERIC			DEFAULT 0
);
CREATE TABLE event_tokop (
	event_id			BIGINT			REFERENCES event(event_id) ON DELETE CASCADE,
	tokop_id			BIGINT			REFERENCES tokop(tokop_id) ON DELETE CASCADE
);
CREATE TABLE event_approval (
	event_id			BIGINT			REFERENCES event(event_id) ON DELETE CASCADE,
	approval_id			BIGINT			REFERENCES approval(approval_id) ON DELETE CASCADE
);
CREATE TABLE ft_hold (
	contract_id			INT			NOT NULL,
	tokacct_id			INT			NOT NULL,
	amount				DECIMAL			NOT NULL
);
CREATE TABLE ft_approve (
	contract_id			INT			NOT NULL,
	tokacct_id			INT			NOT NULL,
	amount				NUMERIC			NOT NULL
);
CREATE TABLE nft_hold (
	contract_id			INT			NOT NULL,
	tokacct_id			INT			NOT NULL,
	token_id			DECIMAL		NOT NULL
);
CREATE TABLE last_token_block (
	block_num			INT				NOT NULL
);
-- txpool
CREATE TABLE pending_tx (
	ptx_id				BIGSERIAL			PRIMARY KEY,
	nonce 				INT				NOT NULL,
	gas_limit			INT 			NOT NULL,
	inserted_ts			INT			DEFAULT 0,
	ptx_status			SMALLINT		DEFAULT 0,	-- core.TxStatus enum
	validated			BOOL			DEFAULT FALSE,
	tx_value			NUMERIC 		NOT NULL,
	gas_price			NUMERIC 		NOT NULL,
	v					NUMERIC			NOT NULL,
	r					NUMERIC			NOT NULL,
	s					NUMERIC			NOT NULL,
	from_address			TEXT			NOT NULL,
	to_address			TEXT			NOT NULL,
	tx_hash 			TEXT			NOT NULL UNIQUE,
	payload				bytea			DEFAULT NULL
);
-- statistics
CREATE TABLE mainstats (
	hash_rate			DECIMAL,
	block_time			DECIMAL,
	tx_per_block		DECIMAL,
	tx_per_sec			DECIMAL,
	gas_price			DECIMAL,
	tx_cost				DECIMAL,
	supply				DECIMAL,
	difficulty			DECIMAL,
	volume				DECIMAL,
	activity			DECIMAL,
	last_block			int
);

