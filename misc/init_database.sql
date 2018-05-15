CREATE TABLE account (
	account_id			SERIAL	 		PRIMARY KEY,
	owner_id			INT			NOT NULL DEFAULT 0,
	last_balance		NUMERIC			DEFAULT 0,
	num_tx				BIGINT			DEFAULT 0,
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
	num_tx				int				NOT NULL DEFAULT 0,
	num_vt				int				NOT NULL DEFAULT 0,
	num_uncles			int				NOT NULL,
	size				int				NOT NULL,
	val_transferred		NUMERIC			NOT NULL DEFAULT 0,
	miner_reward		NUMERIC			NOT NULL DEFAULT 0,
	rcpt_hash			TEXT			NOT NULL,
	mix_hash			TEXT			NOT NULL,
	bloom				bytea			NOT NULL,
	extra				bytea			NOT NULL
);
CREATE TABLE uncle (
	uncle_id			SERIAL 			PRIMARY KEY,
	block_id			INT				NOT NULL REFERENCES block(block_id) ON DELETE CASCADE ON UPDATE CASCADE,
	parent_id			INT				NOT NULL,
	block_num 			INT		 		NOT NULL,
	parent_num			INT				NOT NULL,
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
	gas_limit			INT 			NOT NULL,
	gas_used			INT				NOT NULL,
	tx_value			NUMERIC 		NOT NULL,
	gas_price			NUMERIC 		NOT NULL,
	nonce 				INT				NOT NULL,
	block_id			INT 			NOT NULL REFERENCES block(block_id) ON DELETE CASCADE ON UPDATE CASCADE,
	block_num			INT				NOT NULL,
	tx_index			INT 			NOT NULL,
	tx_ts				INT 			NOT NULL,
	num_vt				INT				NOT NULL DEFAULT 0,
	val_transferred		NUMERIC			NOT NULL,
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
	kind				CHAR			NOT NULL,
	error				TEXT			NOT NULL
);
CREATE TABLE mainstats (
	hash_rate			DECIMAL,
	block_time			DECIMAL,
	tx_per_block		DECIMAL,
	gas_price			DECIMAL,
	tx_cost				DECIMAL,
	supply				DECIMAL,
	difficulty			DECIMAL,
	last_block			int
);
CREATE TABLE last_block (
	block_num			INT				NOT NULL
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
CREATE INDEX account_owner_idx 	ON 	account 		USING 	btree	("owner_id");
CREATE INDEX vt_tx_from_idx		ON	value_transfer	USING	btree	("tx_id");
CREATE INDEX vt_block_num_idx	ON	value_transfer	USING	btree	("block_num");
CREATE INDEX vt_from_id_idx		ON	value_transfer	USING	btree	("from_id");
CREATE INDEX vt_to_id_idx		ON	value_transfer	USING	btree	("to_id");
INSERT INTO account VALUES (-3,0,0,0,'*');
INSERT INTO account VALUES (-2,0,0,0,'');
INSERT INTO account VALUES (-1,0,0,0,'0');
INSERT INTO account VALUES (1,0,0,0,'0000000000000000000000000000000000000000');
INSERT INTO block VALUES(-1,0,-1,0,-1,0,0,0,0,0,'','','','',0,0,0,0,0,0,'','',''::bytea,'');
INSERT INTO transaction VALUES(-1,-1,-1,0,0,0,0,0,-1,-1,0,0,0,0,0,0,0,0,'','',''::bytea);
INSERT INTO mainstats VALUES(0,0,0,0,0,0,0,0);
INSERT INTO last_block VALUES(-1);
CREATE OR REPLACE FUNCTION update_last_balance_insert() RETURNS trigger AS  $$
BEGIN
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
BEGIN
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
CREATE TRIGGER update_last_balance_insert AFTER INSERT ON value_transfer FOR EACH ROW EXECUTE PROCEDURE update_last_balance_insert();
CREATE TRIGGER update_last_balance_delete AFTER DELETE ON value_transfer FOR EACH ROW EXECUTE PROCEDURE update_last_balance_delete();
