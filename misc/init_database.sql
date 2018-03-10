CREATE TABLE account (
	account_id			SERIAL	 		PRIMARY KEY,
	address				TEXT 			NOT NULL
);
CREATE TABLE block (
	block_id			SERIAL 			PRIMARY KEY,
	parent_id			INT				NOT NULL,
	block_num 			INT		 		NOT NULL,
	block_ts			INT				NOT NULL,
	miner_id			INT 			NOT NULL REFERENCES account(account_id) ON DELETE CASCADE ON UPDATE CASCADE,
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
CREATE TABLE uncle (
	uncle_id			SERIAL 			PRIMARY KEY,
	parent_id			INT				NOT NULL,
	block_num 			INT		 		NOT NULL,
	block_ts			INT				NOT NULL,
	miner_id			INT 			NOT NULL REFERENCES account(account_id) ON DELETE CASCADE ON UPDATE CASCADE,
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
	from_id				INT 			NOT NULL REFERENCES account(account_id) ON DELETE CASCADE ON UPDATE CASCADE,
	to_id				INT 			NOT NULL REFERENCES account(account_id) ON DELETE CASCADE ON UPDATE CASCADE,
	gas 				INT 			NOT NULL,
	tx_value			NUMERIC 		NOT NULL,
	gas_price			NUMERIC 		NOT NULL,
	nonce 				INT				NOT NULL,
	block_id			INT 			NOT NULL REFERENCES block(block_id) ON DELETE CASCADE ON UPDATE CASCADE,
	block_num			INT				NOT NULL,
	tx_index			INT 			NOT NULL,
	tx_ts				INT 			NOT NULL,
	v					NUMERIC			NOT NULL,
	r					NUMERIC			NOT NULL,
	s					NUMERIC			NOT NULL,
	tx_status			INT				NOT NULL,
	tx_hash 			TEXT			NOT NULL,
	tx_error			TEXT			NOT NULL,
	payload				bytea			NOT NULL
);
CREATE TABLE value_transfer (
	valtr_id			BIGSERIAL		PRIMARY KEY,
	tx_id				BIGINT			REFERENCES transaction(tx_id) ON DELETE CASCADE ON UPDATE CASCADE,
	block_id			INT				REFERENCES block(block_id) ON DELETE CASCADE ON UPDATE CASCADE,
	block_num			INT				NOT NULL,
	from_id				INT				NOT NULL REFERENCES account(account_id) ON DELETE CASCADE ON UPDATE CASCADE,
	to_id				INT				NOT NULL REFERENCES account(account_id) ON DELETE CASCADE ON UPDATE CASCADE,
	value				NUMERIC			DEFAULT 0,
	from_balance		NUMERIC			DEFAULT 0,
	to_balance			NUMERIC			DEFAULT 0,
	kind				CHAR			NOT NULL
);
CREATE UNIQUE INDEX block_num_idx 		ON 	block 			USING 	btree 	("block_num");
CREATE INDEX block_ts_idx 		ON 	block 			USING 	btree 	("block_ts");
CREATE INDEX block_miner_id_idx ON 	block			USING  	btree 	("miner_id");
CREATE INDEX block_hash_idx 	ON 	block 			USING 	hash	("block_hash");
CREATE INDEX uncle_hash_idx		ON 	uncle			USING	hash	("block_hash");
CREATE INDEX uncle_miner_id_idx ON 	uncle			USING  	btree 	("miner_id");
CREATE UNIQUE INDEX tx_hash_idx	ON	transaction		USING	btree	("tx_hash");
CREATE INDEX tx_ts_idx			ON	transaction		USING	btree	("tx_ts");
CREATE INDEX tx_from_idx		ON	transaction 	USING	btree	("from_id");
CREATE INDEX tx_to_idx			ON	transaction 	USING	btree	("to_id");
CREATE INDEX tx_block_id		ON	transaction 	USING	btree	("block_id");
CREATE INDEX account_addr_idx 	ON 	account 		USING 	hash	("address");
CREATE INDEX vt_tx_from_idx		ON	value_transfer	USING	btree	("tx_id");
CREATE INDEX vt_block_num_idx	ON	value_transfer	USING	btree	("block_num");
CREATE INDEX vt_from_id_idx		ON	value_transfer	USING	btree	("from_id");
CREATE INDEX vt_to_id_idx		ON	value_transfer	USING	btree	("to_id");
INSERT INTO account VALUES (-2,'');
INSERT INTO account VALUES (-1,'0');
INSERT INTO account VALUES (1,'0000000000000000000000000000000000000000');
INSERT INTO block VALUES(-1,0,-1,0,-1,0,0,0,0,0,'','','','','','',''::bytea,'');
INSERT INTO transaction VALUES(-1,-1,-1,0,0,0,0,-1,-1,0,0,0,0,0,'',''::bytea);

