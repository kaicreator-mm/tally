from decimal import Decimal as D, Inexact, ROUND_HALF_EVEN, ROUND_DOWN, localcontext
from itertools import product
import sqlite3

# A: finite, known Boolean domain. Expected is the independent policy.
rows = [((0, None, None), 'DENY'), ((1, 0, None), 'DENY'),
        ((1, 1, 0), 'DENY'), ((1, 1, 1), 'ALLOW')]
def matches(table, x):
    return [out for pattern, out in table
            if all(p is None or p == v for p, v in zip(pattern, x))]
domain = list(product((0, 1), repeat=3))
for x in domain:
    assert matches(rows, x) == ['ALLOW' if all(x) else 'DENY']
assert matches(rows + [((None, None, 1), 'ALLOW')], (0, 1, 1)) == ['DENY', 'ALLOW']
assert matches(rows[:2] + rows[3:], (1, 1, 0)) == []
assert matches(rows[:3] + [((1, 1, 1), 'DENY')], (1, 1, 1)) == ['DENY']
assert {matches(rows, x)[0] for x in domain if x[0] == 0} == {'DENY'}
assert {matches(rows, x)[0] for x in domain if x[:2] == (1, 1)} == {'ALLOW', 'DENY'}
print('A eight known combinations; overlap, gap, wrong output; partial completions')

# B: explicit decimal arithmetic contract, not a monetary policy.
with localcontext() as ctx:
    ctx.prec = 28
    ctx.rounding = ROUND_HALF_EVEN
    cent = D('0.01')
    assert [D(s).quantize(cent) for s in ('2.675', '-2.675')] == [D('2.68'), D('-2.68')]
    assert [D(s).quantize(cent, rounding=ROUND_DOWN) for s in ('2.675', '-2.675')] == [D('2.67'), D('-2.67')]
    assert D(2.675).quantize(cent) == D('2.67')
    assert sum(D('0.335').quantize(cent) for _ in range(3)) == D('1.02')
    assert (3 * D('0.335')).quantize(cent) == D('1.00')
    ctx.traps[Inexact] = True
    assert D('1.230').quantize(cent) == D('1.23')
    try:
        D('1.231').quantize(cent)
    except Inexact:
        pass
    else:
        raise AssertionError('nonzero discarded digit')
    ctx.traps[Inexact] = False
    ctx.prec = 3
    assert (D(1000) + D(1)) - D(1000) == 0
assert (1000 + 1) - 1000 == 1
print('B decimal rounding point, float conversion, Inexact and intermediate precision')

# C: fixed local transaction contrast, initial state after O1 reservation.
with sqlite3.connect(':memory:') as db:
    db.execute('CREATE TABLE stock(s TEXT PRIMARY KEY, q INTEGER NOT NULL CHECK(q>=0)) STRICT')
    db.executemany('INSERT INTO stock VALUES (?,?)', [('X', 2), ('Y', 2)])
    db.commit()  # establish the baseline before the failing transaction
    def take(s, q):
        return db.execute('UPDATE stock SET q=q-? WHERE s=? AND q>=?', (q, s, q)).rowcount
    def current():
        return db.execute('SELECT s,q FROM stock ORDER BY s').fetchall()
    try:
        with db:
            assert take('X', 1) == 1
            assert take('Y', 3) == 0
            raise ValueError('whole order rejected')
    except ValueError:
        pass
    assert current() == [('X', 2), ('Y', 2)]
    with db:
        assert take('X', 1) == 1
        assert take('Y', 3) == 0  # deliberately incorrect: commit partial change
    assert current() == [('X', 1), ('Y', 2)]
print('C same-DB rollback preserves both quantities; partial commit violates all-or-nothing')
