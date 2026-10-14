CREATE TABLE IF NOT EXISTS search_shares (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token TEXT NOT NULL UNIQUE,
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    creator_type TEXT NOT NULL DEFAULT 'admin' CHECK (creator_type IN ('admin', 'guest')),
    title TEXT NOT NULL DEFAULT '',
    query TEXT NOT NULL,
    mode TEXT NOT NULL CHECK (mode IN ('keyword', 'ai')),
    summary TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked', 'deleted')),
    include_private INTEGER NOT NULL DEFAULT 0,
    created_ip_hash TEXT NOT NULL DEFAULT '',
    expires_at TEXT,
    view_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_search_shares_status ON search_shares(status);
CREATE INDEX IF NOT EXISTS idx_search_shares_created_by ON search_shares(created_by);
CREATE INDEX IF NOT EXISTS idx_search_shares_expires_at ON search_shares(expires_at);
CREATE INDEX IF NOT EXISTS idx_search_shares_created_at ON search_shares(created_at DESC);

CREATE TABLE IF NOT EXISTS search_share_items (
    share_id INTEGER NOT NULL REFERENCES search_shares(id) ON DELETE CASCADE,
    website_id INTEGER NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL,
    PRIMARY KEY (share_id, website_id),
    UNIQUE (share_id, sort_order)
);
CREATE INDEX IF NOT EXISTS idx_search_share_items_share_order ON search_share_items(share_id, sort_order);
