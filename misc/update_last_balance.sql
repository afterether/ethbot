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
