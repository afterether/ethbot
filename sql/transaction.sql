CREATE TABLE transaction (
	tx_id				BIGSERIAL 		PRIMARY KEY,
	nonce 				INT				,
	block_id			INT 			NOT NULL REFERENCES block(block_id) ON DELETE CASCADE ON UPDATE CASCADE,
	tx_index			INT 			NOT NULL,
	from_id				INT 			NOT NULL REFERENCES account(account_id) ON DELETE CASCADE ON UPDATE CASCADE,
	to_id				INT 			NOT NULL REFERENCES account(account_id) ON DELETE CASCADE ON UPDATE CASCADE,
	tx_value			NUMERIC 		NOT NULL,
	gas 				BIGINT 			NOT NULL,
	gasPrice			NUMERIC 		NOT NULL,
	tx_ts				INT 			NOT NULL,
	tx_hash 			TEXT
);
