CREATE TABLE IF NOT EXISTS users (
    username VARCHAR(50) PRIMARY KEY,
    password_hash VARCHAR(255) NOT NULL,
    session_token VARCHAR(255),
    csrf_token VARCHAR(255),
    elo INT DEFAULT 500,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS games (
    id VARCHAR(50) PRIMARY KEY,
    player_white VARCHAR(50) REFERENCES users(username),
    player_black VARCHAR(50) REFERENCES users(username),
    fen TEXT NOT NULL,
    state VARCHAR(20) DEFAULT 'waiting', -- waiting, in_progress, finished
    winner VARCHAR(50), -- null, white, black, draw
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);