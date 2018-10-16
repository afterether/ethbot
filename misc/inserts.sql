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
