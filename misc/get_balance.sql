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
