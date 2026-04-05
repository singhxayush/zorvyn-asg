CREATE TABLE IF NOT EXISTS financial_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    amount REAL NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('income', 'expense')),
    category TEXT NOT NULL,
    date DATETIME NOT NULL,
    description TEXT,
    created_by INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    FOREIGN KEY(created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_records_type ON financial_records(type);
CREATE INDEX IF NOT EXISTS idx_records_category ON financial_records(category);
CREATE INDEX IF NOT EXISTS idx_records_date ON financial_records(date);
CREATE INDEX IF NOT EXISTS idx_records_deleted_at ON financial_records(deleted_at);