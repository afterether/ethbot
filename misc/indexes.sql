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
