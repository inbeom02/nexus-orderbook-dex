CREATE TABLE IF NOT EXISTS orders (
    id              TEXT PRIMARY KEY,
    maker           TEXT NOT NULL,
    token_sell      TEXT NOT NULL,
    token_buy       TEXT NOT NULL,
    amount_sell     NUMERIC(78,0) NOT NULL,
    amount_buy      NUMERIC(78,0) NOT NULL,
    expiry          BIGINT NOT NULL,
    nonce           BIGINT NOT NULL,
    salt            NUMERIC(78,0) NOT NULL,
    signature       TEXT NOT NULL,
    side            TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    status          TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'partially_filled', 'filled', 'cancelled')),
    filled_base     NUMERIC(78,0) NOT NULL DEFAULT 0,
    pair            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_maker ON orders(maker);
CREATE INDEX idx_orders_pair_status ON orders(pair, status);
CREATE INDEX idx_orders_pair_side_status ON orders(pair, side, status);

CREATE TABLE IF NOT EXISTS trades (
    id              TEXT PRIMARY KEY,
    buy_order_id    TEXT NOT NULL REFERENCES orders(id),
    sell_order_id   TEXT NOT NULL REFERENCES orders(id),
    buyer           TEXT NOT NULL,
    seller          TEXT NOT NULL,
    pair            TEXT NOT NULL,
    base_amount     NUMERIC(78,0) NOT NULL,
    quote_amount    NUMERIC(78,0) NOT NULL,
    price           DOUBLE PRECISION NOT NULL,
    tx_hash         TEXT NOT NULL DEFAULT '',
    settled_on_chain BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trades_pair ON trades(pair);
CREATE INDEX idx_trades_created_at ON trades(created_at DESC);
