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

-- block
CREATE INDEX block_ts_idx 		ON 	block 		USING 	btree 	(block_ts);
CREATE INDEX block_miner_id_idx ON 	block			USING  	btree 	(miner_id);

-- uncle
CREATE INDEX uncle_parent_num_idx	ON uncle		USING	btree	(parent_num);
CREATE INDEX uncle_miner_id_idx ON 	uncle			USING  	btree 	(miner_id);

-- transaction
CREATE INDEX tx_from_idx		ON	transaction 	USING	btree	(from_id);
CREATE INDEX tx_to_idx			ON	transaction 	USING	btree	(to_id);
CREATE INDEX tx_block_num_idx		ON	transaction 	USING	btree	(block_num);
CREATE UNIQUE INDEX bnumtx_idx		ON	transaction	USING	btree	(bnumtx);

-- pending_tx
CREATE INDEX ptx_from_addr_idx		ON	pending_tx	USING	btree	(from_address);
CREATE INDEX ptx_to_addr_idx		ON	pending_tx	USING	btree	(to_address);

-- account
CREATE INDEX account_owner_idx 	ON 	account 		USING 	btree	(owner_id);
CREATE INDEX account_deleted_idx ON	account			USING	btree	(deleted);

-- value_transfer
CREATE INDEX vt_tx_idx			ON	value_transfer	USING	btree	(tx_id);
CREATE INDEX vt_block_num_idx	ON	value_transfer		USING	btree	(block_num);
CREATE INDEX vt_from_id_idx		ON	value_transfer	USING	btree	(from_id);
CREATE INDEX vt_to_id_idx		ON	value_transfer	USING	btree	(to_id);
CREATE INDEX bnum_valtr_idx 		ON	value_transfer	USING	btree	(block_num,valtr_id);
CREATE UNIQUE INDEX bnumvt_idx		ON	value_transfer	USING	btree	(bnumvt);
CREATE INDEX vt_from_bnum_idx		ON	value_transfer	USING	btree	(from_id,bnumvt DESC);
CREATE INDEX vt_to_bnum_idx		ON	value_transfer	USING	btree	(to_id,bnumvt DESC);

-- event
CREATE INDEX evt_block_num_idx		ON	event		USING	btree	(block_num);
CREATE INDEX evt_contract_id_idx	ON	event		USING	btree	(contract_id);
CREATE INDEX evt_valtr_id_idx		ON	event		USING	btree	(valtr_id);

-- token
CREATE INDEX tok_tx_id_idx		ON	token		USING	btree	(created_tx_id);
CREATE INDEX tok_block_cr_idx	ON	token		USING	btree	(block_created);

-- token_info
CREATE INDEX toki_block_cr_idx	ON	token_info	USING	btree	(block_created);

-- tokop
CREATE INDEX tkop_tx_idx		ON	tokop		USING	btree	(tx_id);
CREATE INDEX tkop_appr_tx_idx		ON	tokop		USING	btree	(approval_tx_id);
CREATE INDEX tkop_contract_id_idx	ON	tokop		USING	btree	(contract_id);
CREATE INDEX tkop_block_num_idx		ON	tokop		USING	btree	(block_num);
CREATE INDEX tkop_from_id_idx		ON	tokop		USING	btree	(from_id);
CREATE INDEX tkop_to_id_idx		ON	tokop		USING	btree	(to_id);
CREATE INDEX tkop_compound1_idx		ON	tokop		USING	btree	(to_id,block_num DESC);
CREATE INDEX tkop_compound2_idx		ON	tokop		USING	btree	(from_id,block_num DESC);

-- approval
CREATE INDEX approval_uniq2	ON	approval	USING	btree	(contract_id,to_id) WHERE expired IS FALSE;
CREATE INDEX approval_full_idx		ON	approval	USING	btree	(contract_id,to_id,block_num) WHERE expired IS FALSE;
CREATE INDEX appr_triple_idx		ON	approval	USING	btree	(contract_id,to_id,block_num);
CREATE INDEX appr_block_num_idx		ON	approval	USING	btree	(block_num);
CREATE INDEX appr_contract_id_idx	ON	tokop		USING	btree	(contract_id);
CREATE INDEX appr_from_id_idx		ON	approval	USING	btree	(from_id);
CREATE INDEX appr_to_id_idx			ON	approval	USING	btree	(to_id);
CREATE INDEX appr_compound1_idx		ON	tokop		USING	btree	(to_id,block_num DESC);
CREATE INDEX appr_compound2_idx		ON	tokop		USING	btree	(from_id,block_num DESC);

-- tokop_approval
CREATE UNIQUE INDEX tokapr_uniq			ON tokop_approval	USING	btree	(tokop_id,approval_id);

-- event_tokop
CREATE INDEX evt_tokop_idx1			ON event_tokop	USING	btree	(event_id);
CREATE INDEX evt_tokop_idx2			ON event_tokop 	USING	btree	(tokop_id);
CREATE INDEX evt_tokop_comp			ON event_tokop	USINg	btree	(event_id,tokop_id);

-- event_approval
CREATE INDEX evt_appr_idx1			ON	event_approval	USING	btree	(event_id);
CREATE INDEX evt_appr_idx2			ON	event_approval	USING	btree	(approval_id);
CREATE INDEX evt_appr_comp			ON	event_approval	USING	btree	(event_id,approval_id);

-- ft_hold
CREATE UNIQUE INDEX ft_hold_idx		ON	ft_hold		USING	btree	(contract_id,tokacct_id);
CREATE INDEX ft_contract_id_idx		ON	ft_hold		USING	btree	(contract_id);
CREATE INDEX ft_tokacct_id_idx		ON	ft_hold		USING	btree	(tokacct_id);

-- nft_hold
CREATE UNIQUE INDEX nft_hold_idx	ON	nft_hold	USING	btree	(contract_id,tokacct_id,token_id);
CREATE INDEX nft_contract_id_idx	ON	nft_hold	USING	btree	(contract_id);
CREATE INDEX nft_tokacct_id_idx		ON	nft_hold	USING	btree	(tokacct_id);

-- tokacct
CREATE UNIQUE INDEX tokacct_uniq_idx	ON	tokacct		USING	btree	(address);


-- ft_approve
CREATE UNIQUE INDEX ftap_uniq_idx	ON	ft_approve	USING	btree	(contract_id,tokacct_id);
INSERT INTO account(account_id,block_created,block_sd,block_del,address) VALUES(-3,-1,-1,-1,'REFUNDS');
INSERT INTO account(account_id,block_created,block_sd,block_del,address) VALUES(-2,-1,-1,-1,'CONTRACT');
INSERT INTO account(account_id,block_created,block_sd,block_del,address) VALUES(-1,-1,-1,-1,'BLOCKCHAIN');
INSERT INTO account(block_created,block_sd,block_del,address) VALUES(-1,-1,-1,'0000000000000000000000000000000000000000'); -- preallocate zero address ourselves to assign it ID=1
INSERT INTO block VALUES(-1,-1,0,-1,0,0,0,0,0,'','','','',0,0,0,0,TRUE,0,0,'','',NULL,NULL);
INSERT INTO transaction VALUES(-1,-1,-1,-1,0,0,0,0,0,-1,0,0,0,0,NULL,NULL,NULL,0,'','','',NULL);
INSERT INTO mainstats VALUES(0,0,0,0,0,0,0,0,0,0,0);
INSERT INTO last_block VALUES(-1);
INSERT INTO last_token_block VALUES(-1);
INSERT INTO tokacct(tx_id,account_id,state_acct_id,address) VALUES (-1,-1,-1,'NONEXISTENT');
CREATE OR REPLACE FUNCTION update_last_balance_insert() RETURNS trigger AS  $$
DECLARE
	slen	int;
BEGIN
		slen:=octet_length(NEW.error);
		IF slen=0 THEN
			UPDATE account SET num_vt=num_vt+1 WHERE account_id=NEW.from_id;
			UPDATE account SET num_vt=num_vt+1 WHERE account_id=NEW.to_id;
		END IF;
		IF NEW.kind = '7' THEN
			UPDATE account SET deleted=1,block_sd=NEW.block_num WHERE account_id=NEW.from_id;
		END IF;
		IF NEW.from_id = NEW.to_id THEN
			RETURN NEW;
		END IF;
		IF NEW.value = 0 THEN
			RETURN NEW;
		END IF;
		UPDATE account SET last_balance=last_balance-NEW.value WHERE account_id=NEW.from_id;
		UPDATE account SET last_balance=last_balance+NEW.value WHERE account_id=NEW.to_id;
		RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION update_last_balance_delete() RETURNS trigger AS  $$
DECLARE
	slen	int;
BEGIN
		slen:=octet_length(OLD.error);
		IF slen=0 THEN
			UPDATE account SET num_vt=num_vt-1 WHERE account_id=OLD.from_id;
			UPDATE account SET num_vt=num_vt-1 WHERE account_id=OLD.to_id;
		END IF;
		IF OLD.kind = '7' THEN
			UPDATE account SET deleted=0,block_sd=0 WHERE account_id=OLD.from_id;
		END IF;
		IF OLD.from_id = OLD.to_id THEN
			RETURN OLD;
		END IF;
		IF OLD.value = 0 THEN
			RETURN OLD;
		END IF;
		UPDATE account SET last_balance=last_balance+OLD.value WHERE account_id=OLD.from_id;
		UPDATE account SET last_balance=last_balance-OLD.value WHERE account_id=OLD.to_id;
		RETURN OLD;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION update_account_nonce_insert() RETURNS trigger AS  $$
BEGIN
		IF NEW.tx_status = 0 THEN
			RETURN NEW;
		END IF;
		UPDATE account SET num_tx=num_tx+1 WHERE account_id=NEW.from_id;
		RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION update_account_nonce_delete() RETURNS trigger AS  $$
BEGIN
		IF OLD.tx_status = 0 THEN
			RETURN OLD;
		END IF;
		UPDATE account SET num_tx=num_tx-1 WHERE account_id=OLD.from_id;
		RETURN OLD;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION update_holdings_tokop_insert() RETURNS trigger AS  $$
DECLARE
	v_cnt numeric;
	val numeric;
	v_last_token_block_num integer;
	v_consumed boolean;
BEGIN
	IF NEW.kind = 1 THEN
		IF NEW.from_id = NEW.to_id THEN
			RETURN NEW;
		END IF;
		IF NEW.value = 0 THEN
			RETURN NEW;
		END IF;
		UPDATE token SET num_transfers=num_transfers+1 WHERE contract_id=NEW.contract_id;
		UPDATE ft_hold SET amount=amount+NEW.value WHERE contract_id=NEW.contract_id AND tokacct_id=NEW.to_id;
       		GET DIAGNOSTICS v_cnt = ROW_COUNT;
		IF v_cnt = 0 THEN
			--RAISE NOTICE 'transfer() block % ins in ft_hold after tokop_insert (%,%,%)',NEW.block_num,NEW.contract_id,NEW.to_id,NEW.value;
			INSERT INTO ft_hold(contract_id,tokacct_id,amount) VALUES (NEW.contract_id,NEW.to_id,NEW.value);
		END IF;
		UPDATE ft_hold SET amount=amount-NEW.value WHERE contract_id=NEW.contract_id AND tokacct_id=NEW.from_id RETURNING amount INTO val;
   		GET DIAGNOSTICS v_cnt = ROW_COUNT;
--		RAISE NOTICE 'UPDATE ROW COUNT IS % , tokacct_id=%, val=%',v_cnt,NEW.from_id,val;
		IF v_cnt=0 THEN
			--RAISE NOTICE 'transfer() block % ins in ft_hold that shouldnt happen due to null amount (%,%,%)',NEW.block_num,NEW.contract_id,NEW.from_id,-NEW.value;
			INSERT INTO ft_hold VALUES(NEW.contract_id,NEW.from_id,-NEW.value);
		ELSE
			IF val = 0 THEN
				DELETE FROM ft_hold WHERE contract_id=NEW.contract_id AND tokacct_id=NEW.from_id;
			END IF;
		END IF;
	END IF;
	IF NEW.kind = 3 THEN
		IF NEW.from_id = NEW.to_id THEN
			RETURN NEW;
		END IF;
		IF NEW.value = 0 THEN
			RETURN NEW;
		END IF;
		UPDATE token SET num_transfers=num_transfers+1 WHERE contract_id=NEW.contract_id;
		UPDATE ft_hold SET amount=amount+NEW.value WHERE contract_id=NEW.contract_id AND tokacct_id=NEW.to_id;
   		GET DIAGNOSTICS v_cnt = ROW_COUNT;
		IF v_cnt = 0 THEN
			--RAISE NOTICE 'transferFrom() block %, ins in ft_hold after tokop_insert (%,%,%)',NEW.block_num,NEW.contract_id,NEW.to_id,NEW.value;
			INSERT INTO ft_hold(contract_id,tokacct_id,amount) VALUES (NEW.contract_id,NEW.to_id,NEW.value);
		END IF;
		UPDATE ft_hold SET amount=amount-NEW.value WHERE contract_id=NEW.contract_id AND tokacct_id=NEW.from_id RETURNING amount INTO val;
       	GET DIAGNOSTICS v_cnt = ROW_COUNT;
--		RAISE NOTICE 'UPDATE ROW COUNT IS % , tokacct_id=%, val=%',v_cnt,NEW.from_id,val;
		IF v_cnt=0 THEN
			--RAISE NOTICE 'transferFrom() block % ins in ft_hold that shouldnt happen due to null amount (%,%,%)',NEW.block_num,NEW.contract_id,NEW.from_id,-NEW.value;
			INSERT INTO ft_hold(contract_id,tokacct_id,amount) VALUES (NEW.contract_id,NEW.from_id,-NEW.value);
		ELSE
			IF val = 0 THEN
				DELETE FROM ft_hold WHERE contract_id=NEW.contract_id AND tokacct_id=NEW.from_id;
			END IF;
		END IF;
	END IF;
	IF NEW.kind = 99 THEN
		INSERT INTO nft_hold(contract_id,tokacct_id,token_id) VALUES(NEW.contract_id,NEW.to_id,NEW.token_id);
		DELETE FROM nft_hold WHERE contract_id=NEW.contract_id AND tokacct_id=NEW.tokacct_id AND token_id=NEW.token_id;
	END IF;
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION update_holdings_tokop_delete() RETURNS trigger AS  $$
DECLARE
    v_cnt numeric;
	val numeric;
BEGIN
	IF OLD.kind = 1 THEN -- TOKENOP_TRANSFER
		IF OLD.from_id = OLD.to_id THEN
			RETURN OLD;
		END IF;
		IF OLD.value = 0 THEN
			RETURN OLD;
		END IF;
		UPDATE token SET num_transfers=num_transfers-1 WHERE contract_id=OLD.contract_id;
		v_cnt:=0;
		UPDATE ft_hold SET amount=amount-OLD.value WHERE contract_id=OLD.contract_id AND tokacct_id=OLD.to_id RETURNING amount into val;
		GET DIAGNOSTICS v_cnt = ROW_COUNT;
		IF v_cnt != 0 THEN
			IF val = 0 THEN
				DELETE FROM ft_hold WHERE contract_id=OLD.contract_id AND tokacct_id=OLD.to_id;
			END IF;
		END IF;
		UPDATE ft_hold SET amount=amount+OLD.value WHERE contract_id=OLD.contract_id AND tokacct_id=OLD.from_id RETURNING amount into val;
		GET DIAGNOSTICS v_cnt = ROW_COUNT;
		IF v_cnt = 0 THEN
			IF val!=0 THEN
				--RAISE NOTICE 'transfer() block %, ins in ft_hold after tokop_delete (%,%,%)',OLD.block_num,OLD.contract_id,OLD.from_id,OLD.value;
				INSERT INTO ft_hold(contract_id,tokacct_id,amount) VALUES (OLD.contract_id,OLD.from_id,OLD.value);
			END IF;
		ELSE
			--RAISE NOTICE 'transfer() block %, delete tokop val=%',OLD.block_num,val;
			IF val=0 THEN
				DELETE FROM ft_hold WHERE contract_id=OLD.contract_id AND tokacct_id=OLD.from_id;
			END IF;
		END IF;
	END IF;
	IF OLD.kind = 3 THEN -- TOKENOP_TRANSFER_FROM
		IF OLD.from_id = OLD.to_id THEN
			RETURN OLD;
		END IF;
		IF OLD.value = 0 THEN
			RETURN OLD;
		END IF;
		UPDATE token SET num_transfers=num_transfers-1 WHERE contract_id=OLD.contract_id;
		UPDATE ft_hold SET amount=amount-OLD.value WHERE contract_id=OLD.contract_id AND tokacct_id=OLD.to_id RETURNING amount into val;
		GET DIAGNOSTICS v_cnt = ROW_COUNT;
		IF v_cnt != 0 THEN
			IF val = 0 THEN
				DELETE FROM ft_hold WHERE contract_id=OLD.contract_id AND tokacct_id=OLD.to_id;
			END IF;
		END IF;
		UPDATE ft_hold SET amount=amount+OLD.value WHERE contract_id=OLD.contract_id AND tokacct_id=OLD.from_id RETURNING amount into val;
		GET DIAGNOSTICS v_cnt = ROW_COUNT;
		IF v_cnt = 0 THEN
			--RAISE NOTICE 'transferFrom() block %, ins in ft_hold after tokop_delete (%,%,%)',OLD.block_num,OLD.contract_id,OLD.from_id,OLD.value;
			INSERT INTO ft_hold(contract_id,tokacct_id,amount) VALUES (OLD.contract_id,OLD.from_id,OLD.value);
		ELSE
			--RAISE NOTICE 'transferFrom() block %, contract %v, account %v, delete tokop val=%',OLD.block_num,OLD.contract_id,OLD.to_id,val;
			IF val=0 THEN
				DELETE FROM ft_hold WHERE contract_id=OLD.contract_id AND tokacct_id=OLD.from_id;
			END IF;
		END IF;
	END IF;
	IF OLD.kind = 99 THEN
		DELETE FROM nft_hold WHERE contract_id=OLD.contract_id AND tokacct_id=OLD.tokacct_id AND token_id=OLD.token_id;
		INSERT INTO nft_hold(contract_id,tokacct_id,token_id) VALUES(OLD.contract_id,OLD.to_id,OLD.token_id);
	END IF;
	RETURN OLD;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION update_holdings_approval_insert() RETURNS trigger AS  $$ -- this function executes prior to the consumption of approval
DECLARE
	v_cnt numeric;
	val numeric;
BEGIN

	UPDATE approval SET expired=TRUE WHERE contract_id=NEW.contract_id AND to_id=NEW.to_id AND expired=FALSE AND block_num<NEW.block_num;
	IF NEW.from_id = NEW.to_id THEN
		RETURN NEW;
	END IF;
	IF NEW.value = 0 THEN
		RETURN NEW;
	END IF;
	v_cnt:=0;
	UPDATE ft_approve SET amount=amount+NEW.value WHERE contract_id=NEW.contract_id AND tokacct_id=NEW.to_id;
	GET DIAGNOSTICS v_cnt = ROW_COUNT;
	IF v_cnt = 0 THEN
		INSERT INTO ft_approve(contract_id,tokacct_id,amount) VALUES (NEW.contract_id,NEW.to_id,NEW.value);
	END IF;

	RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION update_holdings_approval_delete() RETURNS trigger AS  $$ --this function executes only if approve wasn't consumed 
DECLARE
	v_cnt numeric;
	val numeric;
	v_approval_id bigint;
BEGIN
	SELECT approval_id FROM approval WHERE contract_id=OLD.contract_id AND to_id=OLD.to_id AND expired=TRUE AND block_num<OLD.block_num ORDER BY block_num DESC LIMIT 1 INTO v_approval_id;
	GET DIAGNOSTICS v_cnt = ROW_COUNT;
	IF v_cnt = 1 THEN
		UPDATE approval SET expired=FALSE WHERE approval_id=v_approval_id;
	END IF;
	IF OLD.from_id = OLD.to_id THEN
		RETURN OLD;
	END IF;
	IF OLD.value = 0 THEN
		RETURN OLD;
	END IF;
	UPDATE ft_approve SET amount=amount-OLD.value WHERE contract_id=OLD.contract_id AND tokacct_id=OLD.to_id RETURNING amount into val;
	GET DIAGNOSTICS v_cnt = ROW_COUNT;
	IF v_cnt > 0 THEN
		IF val = 0 THEN
			DELETE FROM ft_approve WHERE contract_id=OLD.contract_id AND tokacct_id=OLD.to_id;
		END IF;
	END IF;

	RETURN OLD;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION tkapr_insert() RETURNS trigger AS  $$ -- this function only updates 'value_consumed' and 'expired' fields
DECLARE
	v_cnt numeric;
	val numeric;
	v_last_token_block_num integer;
	v_consumed boolean;
--	v_contract_id	integer;
--	v_from_id 	integer;
--	v_to_id		integer;
--	v_value		numeric;
BEGIN

--	SELECT contract_id,from_id,to_id,value FROM tokop WHERE tokop_id=NEW.tokop_id INTO v_contract_id,v_from_id,v_to_id,v_value;
	IF NEW.from_id = NEW.to_id THEN
		RETURN NEW;
	END IF;
	IF NEW.value = 0 THEN
		RETURN NEW;
	END IF;

	-- update value_consumed
	UPDATE approval SET value_consumed=value_consumed+NEW.value WHERE approval_id=NEW.approval_id RETURNING value_consumed INTO val;
	GET DIAGNOSTICS v_cnt = ROW_COUNT;
	IF v_cnt != 1 THEN
--			RAISE EXCEPTION 'Update value_consumed should update 1 row, but updated % row, contract_id=%, to_id=%, from_id=%',v_cnt,NEW.contract_id,NEW.to_id,NEW.from_id;
	END IF;
	UPDATE approval SET expired=TRUE WHERE approval_id=NEW.approval_id AND value=value_consumed;
	IF val < 0 THEN
--			RAISE EXCEPTION 'Consumed value of approval became negative, contract_id=%, to_id=%, from_id=%',NEW.contract_id,NEW.to_id,NEW.from_id;
	END IF;
	UPDATE ft_approve SET amount=amount-NEW.value WHERE contract_id=NEW.contract_id AND tokacct_id=NEW.to_id RETURNING amount into val;
	GET DIAGNOSTICS v_cnt = ROW_COUNT;
	IF v_cnt > 0 THEN
		IF val = 0 THEN
			DELETE FROM ft_approve WHERE contract_id=NEW.contract_id AND tokacct_id=NEW.to_id;
		END IF;
	END IF;

	RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION tkapr_delete() RETURNS trigger AS  $$
DECLARE
    v_cnt numeric;
	val numeric;
--	v_contract_id	integer;
--	v_from_id 	integer;
--	v_to_id		integer;
--	v_value		numeric;
BEGIN

--    SELECT contract_id,from_id,to_id,value FROM tokop WHERE tokop_id=OLD.tokop_id  INTO v_contract_id,v_from_id,v_to_id,v_value;
--	GET DIAGNOSTICS v_cnt = ROW_COUNT;
--	RAISE NOTICE 'contract_id=%, value=% tokop_id=%, v_cnt=%',v_contract_id,v_value,OLD.tokop_id,v_cnt;
	IF OLD.from_id = OLD.to_id THEN
		RETURN OLD;
	END IF;
	IF OLD.value = 0 THEN
		RETURN OLD;
	END IF;

	-- revert value consumed
	UPDATE approval SET value_consumed=value_consumed+OLD.value WHERE approval_id=OLD.approval_id;
	GET DIAGNOSTICS v_cnt = ROW_COUNT;
	IF v_cnt != 1 THEN
--			RAISE EXCEPTION 'Update value_consumed resulted in more than 1 row updated, contract_id=%, to_id=%, from_id=%',OLD.contract_id,OLD.to_id,OLD.from_id;
	END IF;
	UPDATE approval SET expired=FALSE WHERE approval_id=OLD.approval_id AND value!=value_consumed;

	UPDATE ft_approve SET amount=amount+OLD.value WHERE contract_id=OLD.contract_id AND tokacct_id=OLD.to_id RETURNING amount into val;
	GET DIAGNOSTICS v_cnt = ROW_COUNT;
	IF v_cnt = 0 THEN
		INSERT INTO ft_approve VALUES(OLD.contract_id,OLD.to_id,OLD.value);
	ELSE
		IF val = 0 THEN
			DELETE FROM ft_approve WHERE contract_id=v_contract_id AND tokacct_id=OLD.to_id;
		END IF;
	END IF;

	RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- blockchain
CREATE TRIGGER update_last_balance_insert AFTER INSERT ON value_transfer FOR EACH ROW EXECUTE PROCEDURE update_last_balance_insert();
CREATE TRIGGER update_last_balance_delete AFTER DELETE ON value_transfer FOR EACH ROW EXECUTE PROCEDURE update_last_balance_delete();
CREATE TRIGGER update_account_nonce_insert AFTER INSERT ON transaction FOR EACH ROW EXECUTE PROCEDURE update_account_nonce_insert();
CREATE TRIGGER update_account_nonce_delete AFTER DELETE ON transaction FOR EACH ROW EXECUTE PROCEDURE update_account_nonce_delete();
-- tokens
CREATE TRIGGER update_holdings_tokop_insert AFTER INSERT ON tokop FOR EACH ROW EXECUTE PROCEDURE update_holdings_tokop_insert();
CREATE TRIGGER update_holdings_tokop_delete AFTER DELETE ON tokop FOR EACH ROW EXECUTE PROCEDURE update_holdings_tokop_delete();
CREATE TRIGGER update_holdings_approval_insert AFTER INSERT ON approval FOR EACH ROW EXECUTE PROCEDURE update_holdings_approval_insert();
CREATE TRIGGER update_holdings_approval_delete AFTER DELETE ON approval FOR EACH ROW EXECUTE PROCEDURE update_holdings_approval_delete();
CREATE TRIGGER tkapr_insert AFTER INSERT ON tokop_approval FOR EACH ROW EXECUTE PROCEDURE tkapr_insert();
CREATE TRIGGER tkapr_delete BEFORE DELETE ON tokop_approval FOR EACH ROW EXECUTE PROCEDURE tkapr_delete();
CREATE OR REPLACE FUNCTION get_balance(p_account_id integer,p_block_num integer)
RETURNS text AS $$
DECLARE
	v_valtr_id bigint;
	v_block_num int;
	v_from_id int;
	v_to_id int;
	v_from_balance text;
	v_to_balance text;
BEGIN

	IF p_block_num = -1 THEN
		p_block_num=2147483647;
	END IF;

	SELECT
			v.valtr_id,
			v.block_num,
			v.from_id,
			v.to_id,
			v.from_balance::text,
			v.to_balance::text
	FROM value_transfer v
    LEFT JOIN transaction t ON v.tx_id=t.tx_id
	WHERE
		(v.block_num<=p_block_num) AND
		(
			(v.to_id=p_account_id) OR
			(v.from_id=p_account_id)
		)
	ORDER BY
			v.block_num DESC,v.valtr_id DESC
	LIMIT 1
	INTO v_valtr_id,v_block_num,v_from_id,v_to_id,v_from_balance,v_to_balance;

	IF NOT FOUND THEN
		RETURN '-1';
	END IF;


	IF p_account_id = v_to_id THEN
		RETURN v_to_balance;
	END IF;

	IF p_account_id = v_from_id THEN
		RETURN v_from_balance;
	END IF;

	RAISE EXCEPTION 'PSQL function to get balance failed';

END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION get_blocktime(p_block_num integer)
RETURNS numeric AS $$
DECLARE
	row 		record;
	i			int;
	avg_time	numeric;
	prev_row	record;
	block_time	int;
BEGIN

	IF p_block_num = -1 THEN
		p_block_num=2147483647;
	END IF;

	CREATE UNLOGGED TABLE tmp_blocktime(
		block_time	NUMERIC
	);

	i:=0;
	FOR row IN
		SELECT b.block_ts,b.block_num FROM block AS b WHERE block_num <= p_block_num ORDER BY b.block_num DESC LIMIT 1001
	LOOP
		IF i>1 THEN
			IF prev_row.block_ts < row.block_ts THEN
				RAISE NOTICE 'invalid timestamp for block %',row.block_num;
			ELSE
				block_time=prev_row.block_ts-row.block_ts;
				INSERT INTO tmp_blocktime VALUES (block_time);
			END IF;
		END IF;
		i:=i+1;
		prev_row:=row;
	END LOOP;

	IF NOT FOUND THEN
		RETURN '-1';
	END IF;

	SELECT avg(t.block_time) from tmp_blocktime AS t INTO avg_time;

	DROP TABLE tmp_blocktime;

	RETURN avg_time;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION get_hashrate(p_block_num integer)
RETURNS numeric AS $$
DECLARE
	row 		record;
	i			int;
	avg_time	numeric;
	avg_diff	numeric;
	hashrate	numeric;
	prev_row	record;
	block_time	int;
BEGIN

	IF p_block_num = -1 THEN
		p_block_num=2147483647;
	END IF;

	CREATE UNLOGGED TABLE tmp_hashrate(
		block_time	NUMERIC,
		difficulty  NUMERIC
	);

	i:=0;
	FOR row IN
		SELECT b.block_ts,b.difficulty,b.block_num FROM block AS b WHERE block_num <= p_block_num ORDER BY b.block_num DESC LIMIT 1001
	LOOP
		IF i>1 THEN
			IF prev_row.block_ts < row.block_ts THEN
				RAISE NOTICE 'invalid timestamp for block %',row.block_num;
			ELSE
				block_time=prev_row.block_ts-row.block_ts;
				INSERT INTO tmp_hashrate VALUES (block_time,prev_row.difficulty);
			END IF;
		END IF;
		i:=i+1;
		prev_row:=row;
	END LOOP;

	IF NOT FOUND THEN
		RETURN '-1';
	END IF;

	SELECT avg(h.block_time)+0.0001,avg(h.difficulty) from tmp_hashrate AS h INTO avg_time,avg_diff;

	DROP TABLE tmp_hashrate;

	hashrate:=avg_diff/avg_time;
	RETURN hashrate;
END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION get_TXs(p_account_id integer,p_block_num integer)
RETURNS TABLE(tx_id bigint,block_num int,from_id int,to_id int,tx_value text,tx_index int) AS $$
BEGIN

	IF p_block_num = -1 THEN
		p_block_num=2147483647;
	END IF;

	RETURN QUERY
	SELECT
		t.tx_id,
		t.block_num,
		t.from_id,
		t.to_id,
		t.tx_value::text,
		t.tx_index
	FROM transaction t
	WHERE
		t.block_num<=p_block_num AND
		(
			(t.from_id=p_account_id) OR
			(t.to_id=p_account_id)
		)
	ORDER BY
		t.tx_index;

END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION get_VTs(p_account_id integer,p_block_num integer)
RETURNS TABLE(valtr_id bigint,block_num int,from_id int,from_balance text,to_id int,to_balance text,value text,tx_id bigint,kind char) AS $$
BEGIN

	IF p_block_num = -1 THEN
		p_block_num=2147483647;
	END IF;

	RETURN QUERY
	SELECT
		v.valtr_id,
		v.block_num,
		v.from_id,
		v.from_balance::text,
		v.to_id,
		v.to_balance::text,
		v.value::text,
		v.tx_id,
		v.kind
	FROM value_transfer v
	WHERE
		v.block_num<=p_block_num AND
		(
			(v.from_id=p_account_id) OR
			(v.to_id=p_account_id)
		)
	ORDER BY
		v.block_num DESC,v.valtr_id DESC;

END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION update_last_balance() RETURNS void AS $$
DECLARE
	row 		record;
	balance		numeric;
BEGIN

	FOR row IN
		SELECT account_id FROM account order by account_id
	LOOP
		SELECT get_balance(row.account_id,-1) INTO balance;
		IF balance = '-1' THEN
			balance:=0;
		END IF;
		UPDATE account SET last_balance=balance WHERE account_id=row.account_id;
	END LOOP;

END;
$$ LANGUAGE plpgsql;
CREATE OR REPLACE FUNCTION get_many_balances(p_block_num integer,p_account_ids text)
RETURNS TABLE(aid int,valtr_id bigint,block_num int,from_id int,to_id int,from_balance text,to_balance text) AS $$
DECLARE
	v_account_id text;
	v_acct_int int;
BEGIN

	IF p_block_num = -1 THEN
		p_block_num=2147483647;
	END IF;
	
	FOREACH v_account_id IN array string_to_array(p_account_ids, ',')
	LOOP
		v_acct_int:=v_account_id::int;
		SELECT v_acct_int,s.valtr_id,s.block_num,s.from_id,s.to_id,s.from_balance,s.to_balance FROM
		(
			(
				SELECT v.valtr_id,v.block_num,v.from_id,v.to_id,v.from_balance,v.to_balance
				FROM value_transfer v
				WHERE (v.block_num<=p_block_num) AND (v.from_id = v_acct_int)
			) UNION ALL (
				SELECT v.valtr_id,v.block_num,v.from_id,v.to_id,v.from_balance,v.to_balance
				FROM value_transfer v
				WHERE (v.block_num<=p_block_num) AND (v.to_id = v_acct_int)
			)
		) AS s
		ORDER BY s.block_num DESC,s.valtr_id DESC
		LIMIT 1
		INTO aid,valtr_id,block_num,from_id,to_id,from_balance,to_balance;

		RETURN NEXT;
	END LOOP;
	RETURN;
END;
$$ LANGUAGE plpgsql;
