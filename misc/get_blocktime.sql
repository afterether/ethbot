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
