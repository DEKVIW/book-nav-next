-- Rebuild the share tables so databases that already applied 000002 gain
-- nullable creators and guest-sharing metadata without losing share items.
DROP INDEX IF EXISTS idx_search_shares_status;
DROP INDEX IF EXISTS idx_search_shares_created_by;
DROP INDEX IF EXISTS idx_search_shares_expires_at;
DROP INDEX IF EXISTS idx_search_shares_created_at;
DROP INDEX IF EXISTS idx_search_share_items_share_order;

ALTER TABLE search_share_items RENAME TO search_share_items_legacy;
ALTER TABLE search_shares RENAME TO search_shares_legacy;

CREATE TABLE search_shares (
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

INSERT INTO search_shares(
    id, token, created_by, creator_type, title, query, mode, summary, status,
    include_private, created_ip_hash, expires_at, view_count, created_at, updated_at
)
SELECT id, token, created_by, 'admin', title, query, mode, summary, status,
       0, '', expires_at, view_count, created_at, updated_at
FROM search_shares_legacy;

CREATE TABLE search_share_items (
    share_id INTEGER NOT NULL REFERENCES search_shares(id) ON DELETE CASCADE,
    website_id INTEGER NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL,
    PRIMARY KEY (share_id, website_id),
    UNIQUE (share_id, sort_order)
);

INSERT INTO search_share_items(share_id, website_id, sort_order)
SELECT share_id, website_id, sort_order
FROM search_share_items_legacy;

DROP TABLE search_share_items_legacy;
DROP TABLE search_shares_legacy;

CREATE INDEX idx_search_shares_status ON search_shares(status);
CREATE INDEX idx_search_shares_created_by ON search_shares(created_by);
CREATE INDEX idx_search_shares_expires_at ON search_shares(expires_at);
CREATE INDEX idx_search_shares_created_at ON search_shares(created_at DESC);
CREATE INDEX idx_search_shares_guest_ip ON search_shares(creator_type, created_ip_hash, created_at);
CREATE INDEX idx_search_share_items_share_order ON search_share_items(share_id, sort_order);
