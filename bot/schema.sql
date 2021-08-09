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

create table reactions
(
    when_replying_to varchar(160) default 'stub'   not null,
    regex_str        varchar(4096)                 not null,
    reply_str        varchar(4096)                 not null,
    added_by         varchar(160) default 'system' not null
);

create table users
(
    identifier varchar(160) not null,
    perm_level int          not null
);

