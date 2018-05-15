-- Example queries
-- Select account balances ordering from higher to lower

SELECT
	a.address,
	last_balance/1000000000000000000 AS eth_balance,
FROM account AS a
ORDER BY last_balance DESC
LIMIT 100;

