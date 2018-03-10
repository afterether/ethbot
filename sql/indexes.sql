CREATE INDEX block_num_idx 		ON 	block 		USING 	btree 	("block_num");
CREATE INDEX block_ts_idx 		ON 	block 		USING 	btree 	("block_ts");
CREATE INDEX block_miner_id_idx ON 	block		USING  	btree 	("miner_id");
CREATE INDEX block_hash_idx 	ON 	block 		USING 	hash	("block_hash");
CREATE INDEX tx_hash_idx		ON	transaction	USING	hash	("tx_hash");
CREATE INDEX tx_ts_idx			ON	transaction	USING	btree	("tx_ts");
CREATE INDEX tx_from_idx		ON	transactoin USING	btree	("from_id");
CREATE INDEX tx_to_idx			ON	transaction USING	btree	("to_id");
CREATE INDEX tx_block_id		ON	transaction USINg	btree	("block_id");
CREATE INDEX account_addr_idx 	ON 	account 	USING 	hash	("address");

