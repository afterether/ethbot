CREATE OR REPLACE FUNCTION update_last_balance_insert() RETURNS trigger AS  $$
BEGIN
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
BEGIN
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
