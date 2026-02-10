-- Clear all data from the chessgame database
-- WARNING: This will DELETE ALL DATA from all tables!
DELETE FROM games;

DELETE FROM users;

SELECT 'users' as table_name, COUNT(*) as row_count FROM users
UNION ALL
SELECT 'games' as table_name, COUNT(*) as row_count FROM games;
