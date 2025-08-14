-- Initial schema for MeshGo database

CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chats (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    encryption INTEGER NOT NULL DEFAULT 0,
    last_message_ts INTEGER NOT NULL DEFAULT 0,
    unread_count INTEGER NOT NULL DEFAULT 0,
    is_channel BOOLEAN NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id TEXT NOT NULL,
    sender_id TEXT NOT NULL,
    portnum INTEGER NOT NULL,
    text TEXT NOT NULL,
    rx_snr REAL,
    rx_rssi INTEGER,
    timestamp INTEGER NOT NULL,
    is_unread BOOLEAN NOT NULL DEFAULT 1,
    FOREIGN KEY (chat_id) REFERENCES chats(id)
);

CREATE INDEX IF NOT EXISTS idx_messages_chat_timestamp ON messages(chat_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_messages_unread ON messages(is_unread) WHERE is_unread = 1;

CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    short_name TEXT,
    long_name TEXT,
    favorite BOOLEAN NOT NULL DEFAULT 0,
    ignored BOOLEAN NOT NULL DEFAULT 0,
    unencrypted BOOLEAN NOT NULL DEFAULT 0,
    enc_default_key BOOLEAN NOT NULL DEFAULT 0,
    enc_custom_key BOOLEAN NOT NULL DEFAULT 0,
    rssi INTEGER DEFAULT 0,
    snr REAL DEFAULT 0.0,
    signal_quality INTEGER DEFAULT 0,
    battery_level INTEGER,
    is_charging BOOLEAN,
    last_heard INTEGER NOT NULL DEFAULT 0,
    position_data TEXT,
    device_metrics_data TEXT
);

CREATE INDEX IF NOT EXISTS idx_nodes_favorite ON nodes(favorite) WHERE favorite = 1;
CREATE INDEX IF NOT EXISTS idx_nodes_last_heard ON nodes(last_heard DESC);

CREATE TABLE IF NOT EXISTS channels (
    name TEXT PRIMARY KEY,
    psk_class INTEGER NOT NULL DEFAULT 0,
    psk_data BLOB,
    channel_id INTEGER
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT OR IGNORE INTO schema_version (version) VALUES (1);