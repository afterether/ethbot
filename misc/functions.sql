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
		DROP TABLE tmp_hashrate;
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
