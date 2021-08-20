-- user identifier and any kind of message source length: 160

create table irc_configs
(
    subident     varchar(32)               not null primary key,
    auto_join    varchar(4096) default ''  not null,
    address      varchar(160)              not null,
    tls          tinyint(1)    default 1   not null,
    nick_name    varchar(32)               not null,
    user_name    varchar(32)               not null,
    real_name    varchar(32)               not null,
    pass         varchar(64)   default ''  not null,
    ping_freq    int           default 60  not null,
    ping_timeout int           default 120 not null
);

CREATE TABLE reactions
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

create table users
(
    identifier varchar(160) not null,
    perm_level int          not null
);

