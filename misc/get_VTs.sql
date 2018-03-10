CREATE OR REPLACE FUNCTION get_VTs(p_account_id integer,p_block_num integer)
RETURNS TABLE(valtr_id bigint,block_num int,from_id int,from_balance text,to_id int,to_balance text,value text) AS $$
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
		v.value::text
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
