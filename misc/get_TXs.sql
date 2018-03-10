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
