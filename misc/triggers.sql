-- blockchain
CREATE TRIGGER update_last_balance_insert AFTER INSERT ON value_transfer FOR EACH ROW EXECUTE PROCEDURE update_last_balance_insert();
CREATE TRIGGER update_last_balance_delete AFTER DELETE ON value_transfer FOR EACH ROW EXECUTE PROCEDURE update_last_balance_delete();
CREATE TRIGGER update_account_nonce_insert AFTER INSERT ON transaction FOR EACH ROW EXECUTE PROCEDURE update_account_nonce_insert();
CREATE TRIGGER update_account_nonce_delete AFTER DELETE ON transaction FOR EACH ROW EXECUTE PROCEDURE update_account_nonce_delete();
-- tokens
CREATE TRIGGER update_holdings_tokop_insert AFTER INSERT ON tokop FOR EACH ROW EXECUTE PROCEDURE update_holdings_tokop_insert();
CREATE TRIGGER update_holdings_tokop_delete AFTER DELETE ON tokop FOR EACH ROW EXECUTE PROCEDURE update_holdings_tokop_delete();
CREATE TRIGGER update_holdings_approval_insert AFTER INSERT ON approval FOR EACH ROW EXECUTE PROCEDURE update_holdings_approval_insert();
CREATE TRIGGER update_holdings_approval_delete AFTER DELETE ON approval FOR EACH ROW EXECUTE PROCEDURE update_holdings_approval_delete();
CREATE TRIGGER tkapr_insert AFTER INSERT ON tokop_approval FOR EACH ROW EXECUTE PROCEDURE tkapr_insert();
CREATE TRIGGER tkapr_delete BEFORE DELETE ON tokop_approval FOR EACH ROW EXECUTE PROCEDURE tkapr_delete();
