PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
                                     id integer primary key,
                                     username text,
                                     passwordBytes blob,
                                     admin bool
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (
                                                               username
);

CREATE TABLE IF NOT EXISTS mailboxes (
                                         id integer primary key,
                                         userid integer,
                                         name text,
                                         uidnext integer default 1,
                                         uidvalidity integer default 1,
                                         subscribed bool default true,
                                         FOREIGN KEY(userid) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS messages (
                                        id integer primary key,
                                        mailboxid integer,
                                        content blob,
                                        uid integer,
                                        ts timestamp,
                                        FOREIGN KEY (mailboxid) REFERENCES mailboxes(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS messageflags (
                                            id integer primary key,
                                            messageid integer not null,
                                            flag text,
                                            FOREIGN KEY (messageid) REFERENCES messages(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS queue (
                                     id integer primary key,
                                     msgfrom text,
                                     msgto text,
                                     ts timestamp,
                                     retries integer default 0,
                                     content blob
);