import asyncio
import sqlite3

# A: K2 erratum, positive integer stored-value domain, not all input syntax.
with sqlite3.connect(':memory:') as db:
    db.execute('CREATE TABLE loose(q INTEGER NOT NULL CHECK(q>0))')
    db.execute("INSERT INTO loose VALUES ('abc'),(1.5)")
    assert db.execute('SELECT q,typeof(q) FROM loose ORDER BY rowid').fetchall() == [('abc','text'),(1.5,'real')]
    db.execute('CREATE TABLE strict_q(q INTEGER NOT NULL CHECK(q>0)) STRICT')
    for value in ('abc', 1.5, 0, -1, None):
        try:
            db.execute('INSERT INTO strict_q VALUES (?)', (value,))
        except sqlite3.IntegrityError:
            pass
        else:
            raise AssertionError(value)
    db.execute('INSERT INTO strict_q VALUES (?)', (1,))
    db.execute('INSERT INTO strict_q VALUES (?)', ('2',))  # lossless coercion
    assert db.execute('SELECT q,typeof(q) FROM strict_q ORDER BY q').fetchall() == [(1,'integer'),(2,'integer')]
    print('A loose accepts text/real; STRICT rejects selected invalid values; coerces numeric text')

# B: waiting boundary and TaskGroup-owned child cleanup.
async def lifecycle():
    gate = asyncio.Event()
    async def work():
        await gate.wait()
        return 'effect'
    task = asyncio.create_task(work())
    done, pending = await asyncio.wait({task}, timeout=0)
    assert not done and pending == {task} and not task.cancelled()
    gate.set()
    assert await task == 'effect'
    started, cleaned = asyncio.Event(), asyncio.Event()
    async def sibling():
        try:
            started.set()
            await asyncio.Event().wait()
        finally:
            cleaned.set()
    async def fail():
        await started.wait()
        raise ValueError('failure')
    try:
        async with asyncio.TaskGroup() as group:
            child = group.create_task(sibling())
            group.create_task(fail())
    except* ValueError as group_error:
        assert len(group_error.exceptions) == 1
    else:
        raise AssertionError('missing group failure')
    assert child.cancelled() and cleaned.is_set()
    print('B wait timed out without cancellation; later result; group sibling cancelled and cleaned')

asyncio.run(lifecycle())

# C: local transaction only; stable event identity and canonical teaching body.
with sqlite3.connect(':memory:') as db:
    db.execute('CREATE TABLE inbox(id TEXT PRIMARY KEY NOT NULL, body TEXT NOT NULL)')
    db.execute('CREATE TABLE effect(id TEXT PRIMARY KEY NOT NULL, body TEXT NOT NULL)')
    def apply(event, body, fail=False):
        with db:
            old = db.execute('SELECT body FROM inbox WHERE id=?', (event,)).fetchone()
            if old is not None:
                return 'duplicate' if old == (body,) else 'conflict'
            db.execute('INSERT INTO inbox VALUES (?,?)', (event,body))
            db.execute('INSERT INTO effect VALUES (?,?)', (event,body))
            if fail:
                raise ValueError('before commit')
        return 'applied'
    try:
        apply('E8','v8',True)
    except ValueError:
        pass
    else:
        raise AssertionError('missing failure')
    for table in ('inbox','effect'):
        assert db.execute(f'SELECT * FROM {table}').fetchall() == []
    assert apply('E8','v8') == 'applied'
    assert apply('E8','v8') == 'duplicate'
    assert apply('E8','other') == 'conflict'
    for table in ('inbox','effect'):
        assert db.execute(f'SELECT * FROM {table}').fetchall() == [('E8','v8')]
    print('C precommit exception rolls both back; repeat same identity no new effect; conflicting body rejected')
