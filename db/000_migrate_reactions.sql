CREATE TABLE reactions_copy
(
    id                  INTEGER PRIMARY KEY                 NOT NULL,
    when_replying_to    VARCHAR(160) DEFAULT 'stub'         NOT NULL,
    regex_str           VARCHAR(4096)                       NOT NULL,
    reply_str           VARCHAR(4096)                       NOT NULL,
    added_by            VARCHAR(160) DEFAULT 'system'       NOT NULL,
    deleted_by          VARCHAR(160) DEFAULT 'system',
    created_at          DATETIME DEFAULT current_timestamp  NOT NULL,
    updated_at          DATETIME DEFAULT current_timestamp  NOT NULL,
    deleted_at          DATETIME DEFAULT NULL,
    hits                INTEGER DEFAULT 0                   NOT NULL
);

-- Migrate data into new table
INSERT INTO reactions_copy (when_replying_to, regex_str, reply_str, added_by)
    SELECT when_replying_to, regex_str, reply_str, added_by FROM reactions;

DROP TABLE reactions;

ALTER TABLE reactions_copy RENAME TO reactions;