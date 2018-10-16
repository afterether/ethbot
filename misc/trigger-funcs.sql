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

