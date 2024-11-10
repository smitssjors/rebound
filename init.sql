PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA busy_timeout=5000;
PRAGMA cache_size=2000;
PRAGMA mmap_size=134217728; -- 128MB
PRAGMA temp_store=MEMORY;

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
	queue TEXT NOT NULL,
    priority INTEGER NOT NULL,
	locked_until INTEGER NOT NULL,
	ttr INTEGER NOT NULL,
	body TEXT NOT NULL
) STRICT;
CREATE INDEX IF NOT EXISTS messages_queue_locked_until ON messages (queue, locked_until); 
