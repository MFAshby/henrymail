PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
                                     id integer primary key not null,
                                     username text not null,
                                     passwordBytes blob not null,
                                     admin bool not null
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (
                                                               username
);

CREATE TABLE IF NOT EXISTS mailboxes (
                                         id integer primary key not null,
                                         userid integer not null,
                                         name text not null,
                                         uidnext integer default 1 not null,
                                         uidvalidity integer default 1 not null,
                                         subscribed bool default true not null,
                                         FOREIGN KEY(userid) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS ids_mailboxes_userid on mailboxes (
                                                              userid
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mailboxes_userid_name on mailboxes (
                                                              userid,
                                                              name
);

CREATE TABLE IF NOT EXISTS messages (
                                        id integer primary key not null,
                                        mailboxid integer not null,
                                        content blob not null,
                                        uid integer not null,
                                        ts timestamp not null,
                                        flagsjson blob not null,
                                        FOREIGN KEY (mailboxid) REFERENCES mailboxes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_mailboxid ON messages (
                                                              mailboxid
);

CREATE TABLE IF NOT EXISTS keys (
    id integer primary key not null,
    name text not null,
    key blob not null
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_keys_name on keys (
    name
);