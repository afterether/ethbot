CREATE TABLE block (
	block_id			SERIAL 		PRIMARY KEY,
	block_num 			INT		 	NOT NULL,
	block_ts			INT			NOT NULL,
	miner_id			INT 		NOT NULL,
	uncle_pos			SMALLINT 	NOT NULL,
	difficulty			NUMERIC		NOT NULL,
	total_dif			NUMERIC		NOT NULL,
	block_hash 			TEXT		NOT NULL UNIQUE
);
