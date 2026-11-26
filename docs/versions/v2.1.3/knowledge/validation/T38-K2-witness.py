import sqlite3

with sqlite3.connect(':memory:') as db:
    print('SQLite', sqlite3.sqlite_version)
    db.execute('PRAGMA foreign_keys=ON')
    assert db.execute('PRAGMA foreign_keys').fetchone() == (1,)
    db.executescript('''
        CREATE TABLE project(id TEXT PRIMARY KEY NOT NULL);
        CREATE TABLE device(id INTEGER PRIMARY KEY, vendor INTEGER NOT NULL,
                            serial TEXT NOT NULL, UNIQUE(vendor, serial));
        CREATE TABLE link(p TEXT NOT NULL REFERENCES project(id),
                          d INTEGER NOT NULL REFERENCES device(id),
                          role TEXT NOT NULL, PRIMARY KEY(p,d));
        CREATE TABLE quantity(q INTEGER CHECK(q>0));
        CREATE TABLE required_quantity(q INTEGER NOT NULL CHECK(q>0));
        CREATE TABLE optional_key(k INTEGER UNIQUE);
        INSERT INTO project VALUES ('A'),('B');
        INSERT INTO device VALUES (1,1,'s'),(2,2,'s');
        INSERT INTO link VALUES ('A',1,'reader'),('A',2,'operator');
    ''')

    def rejected(sql):
        try:
            db.execute(sql)
        except sqlite3.IntegrityError:
            return True
        return False

    assert rejected("INSERT INTO device VALUES (3,1,'s')")
    assert rejected("INSERT INTO link VALUES ('C',1,'reader')")
    assert rejected("INSERT INTO link VALUES ('A',NULL,'reader')")
    db.execute('INSERT INTO quantity VALUES (NULL)')
    assert rejected('INSERT INTO required_quantity VALUES (NULL)')
    assert rejected('INSERT INTO quantity VALUES (0)')
    db.execute('INSERT INTO optional_key VALUES (NULL),(NULL)')
    assert db.execute('SELECT COUNT(*) FROM optional_key').fetchone() == (2,)
    print('A scoped uniqueness, FK, CHECK NULL, NOT NULL, UNIQUE NULL observed')

    join = ' FROM project p LEFT JOIN link l ON p.id=l.p'
    assert db.execute('SELECT p.id,l.d'+join+" AND l.role='reader' ORDER BY p.id").fetchall() == [('A',1),('B',None)]
    assert db.execute('SELECT p.id,l.d'+join+" WHERE l.role='reader' ORDER BY p.id").fetchall() == [('A',1)]
    assert db.execute('SELECT p.id,COUNT(*),COUNT(l.d)'+join+" AND l.role='reader' GROUP BY p.id ORDER BY p.id").fetchall() == [('A',1,1),('B',1,0)]
    assert db.execute('SELECT NOT (NULL=3), 1 NOT IN (2,NULL)').fetchone() == (None,None)
    print('B ON retains B; WHERE removes B; B counts 1/0; predicates UNKNOWN')

    db.execute('CREATE TABLE stock(q INTEGER NOT NULL CHECK(q>=0))')
    db.execute('INSERT INTO stock VALUES (5)')
    take = 'UPDATE stock SET q=q-? WHERE q>=?'
    assert db.execute(take,(3,3)).rowcount == 1
    assert db.execute(take,(4,4)).rowcount == 0
    assert db.execute('SELECT q FROM stock').fetchone() == (2,)
    db.commit()
    print('C sequential conditional debit 3/4 from 5: affected 1/0; final 2')
