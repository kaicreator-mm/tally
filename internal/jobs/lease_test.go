package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// 租约合同验收(P0-10 / PRD §17.1 Job Lease / §37 P0 correctness):
//
//	AC1 认领写 owner 与租约;
//	AC2 心跳只续当前 owner 的活租约,命中 0 行(RowsAffected=0)必须中止;
//	AC3 过期的 running(含迁移前 lease_until 为 NULL 的存量卡死行)可被回收重认领;
//	AC4 活租约永不被回收/抢走;
//	AC5 取消不得标记成功完成。
//
// 全部走内存查询面(fakeQueries,谓词建模与 SQL 一致),不依赖真库。

// waitFor 有界等待通道闭合,超时即 Fatal(防执行路径卡死拖挂测试)。
func waitFor(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal(msg)
	}
}

// takeAsync 在独立 goroutine 里 takeOne,返回结果通道。
func takeAsync(q *Queue, ctx context.Context) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		took, err := q.takeOne(ctx)
		if err != nil {
			errCh <- err
			return
		}
		if !took {
			errCh <- errors.New("应有任务被认领")
			return
		}
		errCh <- nil
	}()
	return errCh
}

// AC1:认领写 owner 与租约 —— 行转 running、归属认领者、租约窗口从认领时刻起算、attempts 递增。
func TestClaimWritesOwnerAndLease(t *testing.T) {
	f := newFakeQueries()
	base := time.Unix(1700000000, 0)
	f.now = func() time.Time { return base } // 冻结时钟:租约数学可精确断言

	const lease = 30 * time.Second
	var inHandler fakeJob
	q := newQueue(f, map[Kind]Handler{
		KindDeploy: func(context.Context, json.RawMessage) error {
			inHandler = f.job(1)
			return nil
		},
	}, WithLease(lease))

	f.seedPending(string(KindDeploy), []byte(`{}`))
	if _, err := q.takeOne(context.Background()); err != nil {
		t.Fatalf("takeOne: %v", err)
	}

	if inHandler.status != statusRunning {
		t.Errorf("认领后 status = %q, want running", inHandler.status)
	}
	if !inHandler.workerID.Valid || inHandler.workerID.String != q.workerID {
		t.Errorf("认领后 worker_id = %+v, want 本队列 owner %q", inHandler.workerID, q.workerID)
	}
	if inHandler.attempts != 1 {
		t.Errorf("认领后 attempts = %d, want 1", inHandler.attempts)
	}
	if want := base.Add(lease); !inHandler.leaseUntil.Equal(want) {
		t.Errorf("认领后 lease_until = %v, want 认领时刻+租约窗口 %v", inHandler.leaseUntil, want)
	}
}

// AC2(正向):租约存活期间心跳持续续租,且每次续租都以「行 id + 本 owner + 租约窗口」为入参。
func TestHeartbeatRenewsWhileLeaseLive(t *testing.T) {
	f := newFakeQueries()
	const lease = 10 * time.Second
	q := newQueue(f, map[Kind]Handler{
		KindDiagnose: func(ctx context.Context, _ json.RawMessage) error {
			select { // 长任务:给心跳留出触发窗口
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(80 * time.Millisecond):
				return nil
			}
		},
	}, WithLease(lease), WithHeartbeatInterval(10*time.Millisecond))

	f.seedPending(string(KindDiagnose), []byte(`{}`))
	if _, err := q.takeOne(context.Background()); err != nil {
		t.Fatalf("takeOne: %v", err)
	}

	if len(f.heartbeats) == 0 {
		t.Fatal("长任务执行期间应发生心跳续租")
	}
	for i, hb := range f.heartbeats {
		if hb.ID != 1 {
			t.Errorf("心跳 #%d 的行 id = %d, want 1", i, hb.ID)
		}
		if hb.WorkerID.String != q.workerID || !hb.WorkerID.Valid {
			t.Errorf("心跳 #%d 的 owner = %+v, want 本队列 owner(只续自己的租约)", i, hb.WorkerID)
		}
		if hb.Column3.Microseconds != lease.Microseconds() {
			t.Errorf("心跳 #%d 的续租窗口 = %dµs, want %dµs", i, hb.Column3.Microseconds, lease.Microseconds())
		}
	}
	// 成功完成后仍落完成(settle 的闸门续租对活租约命中 1 行)。
	if len(f.doneIDs) != 1 || f.doneIDs[0] != 1 {
		t.Errorf("MarkJobDone 记录 = %v, want [1]", f.doneIDs)
	}
}

// AC2(fencing):心跳命中 0 行 = 租约已被接管/过期,执行必须被中止;
// 即使 handler 之后误报成功,也不得写任何终态。
func TestLostLeaseAbortsHandlerAndForbidsTerminalWrite(t *testing.T) {
	f := newFakeQueries()
	f.hbScript = []int64{0} // 第一次心跳即命中 0 行:租约丢了

	aborted := make(chan struct{}) // handler 收到中止信号
	q := newQueue(f, map[Kind]Handler{
		KindDiagnose: func(ctx context.Context, _ json.RawMessage) error {
			<-ctx.Done() // 只随执行上下文中止
			close(aborted)
			return nil // 即使误报成功
		},
	}, WithLease(time.Hour), WithHeartbeatInterval(5*time.Millisecond))

	f.seedPending(string(KindDiagnose), []byte(`{}`))
	errCh := takeAsync(q, context.Background())
	waitFor(t, aborted, "租约丢失后 handler 未被中止")

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("takeOne: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("takeOne 未随中止收尾")
	}

	if len(f.doneIDs) != 0 {
		t.Errorf("租约丢失后不得写完成: %v", f.doneIDs)
	}
	if len(f.failed) != 0 {
		t.Errorf("租约丢失后不得写失败: %+v", f.failed)
	}
	if got := f.job(1).status; got != statusRunning {
		t.Errorf("status = %q, want 保持 running 由接管方/回收路径处置", got)
	}
}

// AC3:过期的 running(worker 崩溃遗留)可被重新认领:换主、续新租约、attempts 在旧值上递增。
func TestExpiredRunningJobIsReclaimable(t *testing.T) {
	f := newFakeQueries()
	base := time.Unix(1700000000, 0)
	adv := base
	f.now = func() time.Time { return adv }

	var inHandler fakeJob
	q := newQueue(f, map[Kind]Handler{
		KindDeploy: func(context.Context, json.RawMessage) error {
			inHandler = f.job(1)
			return nil
		},
	}, WithLease(30*time.Second))

	// 崩溃遗留:running、租约一分钟前已过期、attempts=1。
	f.seedRunning(string(KindDeploy), "dead-worker", base.Add(-time.Minute), 1)

	adv = base.Add(time.Second) // 时间前进:该行成为 stale
	took, err := q.takeOne(context.Background())
	if err != nil || !took {
		t.Fatalf("过期 running 应可被重新认领: took=%v err=%v", took, err)
	}

	if !inHandler.workerID.Valid || inHandler.workerID.String != q.workerID {
		t.Errorf("重新认领后 worker_id = %+v, want 新 owner %q", inHandler.workerID, q.workerID)
	}
	if inHandler.attempts != 2 {
		t.Errorf("重新认领后 attempts = %d, want 2(旧值 1 上递增)", inHandler.attempts)
	}
	if want := base.Add(time.Second).Add(30 * time.Second); !inHandler.leaseUntil.Equal(want) {
		t.Errorf("重新认领后 lease_until = %v, want %v(认领即续新租约)", inHandler.leaseUntil, want)
	}
	if f.doneIDs == nil || len(f.doneIDs) != 1 {
		t.Errorf("回收重跑成功后应落完成: %v", f.doneIDs)
	}
}

// AC3(存量):迁移前卡死的 running 行(lease_until 为 NULL)同样可被回收重认领。
func TestNullLeaseRunningJobIsReclaimable(t *testing.T) {
	f := newFakeQueries()
	q := newQueue(f, map[Kind]Handler{
		KindDiagnose: func(context.Context, json.RawMessage) error { return nil },
	})

	f.seedRunning(string(KindDiagnose), "prehistoric-worker", time.Time{}, 3) // NULL 租约

	took, err := q.takeOne(context.Background())
	if err != nil || !took {
		t.Fatalf("NULL 租约的 running 行应可被重新认领: took=%v err=%v", took, err)
	}
	if got := f.job(1); got.workerID.String != q.workerID || got.attempts != 4 {
		t.Errorf("回收后行 = worker:%q attempts:%d, want 新 owner + attempts 4", got.workerID.String, got.attempts)
	}
}

// AC4:活租约(lease_until > now)永不被回收/抢走 —— 认领谓词跳过,行原样不动。
func TestLiveLeaseIsNeverReclaimed(t *testing.T) {
	f := newFakeQueries()
	base := time.Unix(1700000000, 0)
	f.now = func() time.Time { return base }

	handlerCalled := false
	q := newQueue(f, map[Kind]Handler{
		KindDeploy: func(context.Context, json.RawMessage) error {
			handlerCalled = true
			return nil
		},
	})

	f.seedRunning(string(KindDeploy), "other-worker", base.Add(time.Hour), 1) // 活租约

	took, err := q.takeOne(context.Background())
	if err != nil {
		t.Fatalf("takeOne: %v", err)
	}
	if took {
		t.Error("活租约的 running 行不可被认领")
	}
	if handlerCalled {
		t.Error("不应执行被他人持有的任务")
	}
	got := f.job(1)
	if got.workerID.String != "other-worker" || got.attempts != 1 || got.status != statusRunning {
		t.Errorf("活租约行被触碰: worker:%q attempts:%d status:%q, want 原样不动",
			got.workerID.String, got.attempts, got.status)
	}
}

// AC5:取消不是完成 —— 队列关闭/上下文取消后,即使 handler 返回 nil 也不得落完成;
// 行保持 running,由租约过期后的回收路径接管重跑。
func TestCancellationDoesNotMarkDone(t *testing.T) {
	t.Run("handler 返回 nil 但执行上下文已取消", func(t *testing.T) {
		f := newFakeQueries()
		parent, cancel := context.WithCancel(context.Background())
		started := make(chan struct{})
		q := newQueue(f, map[Kind]Handler{
			KindDiagnose: func(ctx context.Context, _ json.RawMessage) error {
				close(started)
				<-ctx.Done() // 随队列关闭中止
				return nil   // 误报成功
			},
		}, WithLease(time.Hour), WithHeartbeatInterval(time.Hour)) // 心跳不触发:排除租约因素

		f.seedPending(string(KindDiagnose), []byte(`{}`))
		errCh := takeAsync(q, parent)
		waitFor(t, started, "handler 未启动")
		cancel() // 队列关闭

		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("takeOne: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("取消后 takeOne 未收尾")
		}

		if len(f.doneIDs) != 0 {
			t.Errorf("取消后不得写完成: %v", f.doneIDs)
		}
		if len(f.failed) != 0 {
			t.Errorf("取消不是失败,不得写失败终态: %+v", f.failed)
		}
		if got := f.job(1).status; got != statusRunning {
			t.Errorf("status = %q, want 保持 running 等待租约过期回收", got)
		}
	})

	t.Run("handler 返回 context.Canceled 不落终态", func(t *testing.T) {
		f := newFakeQueries()
		q := newQueue(f, map[Kind]Handler{
			KindDiagnose: func(ctx context.Context, _ json.RawMessage) error {
				return context.Canceled // handler 自报取消
			},
		}, WithLease(time.Hour), WithHeartbeatInterval(time.Hour))

		f.seedPending(string(KindDiagnose), []byte(`{}`))
		if _, err := q.takeOne(context.Background()); err != nil {
			t.Fatalf("takeOne: %v", err)
		}
		if len(f.doneIDs) != 0 || len(f.failed) != 0 {
			t.Errorf("取消不落终态: done=%v failed=%+v", f.doneIDs, f.failed)
		}
		if got := f.job(1).status; got != statusRunning {
			t.Errorf("status = %q, want running", got)
		}
	})
}

// settle 的最后一道租约闸门:handler 结束后租约已丢失(如接管恰好发生),
// 落终态前的续租命中 0 行,任何终态写入都必须放弃 —— 不覆盖接管者的执行。
func TestSettleFencedByLiveLease(t *testing.T) {
	t.Run("闸门 0 行:不写终态", func(t *testing.T) {
		f := newFakeQueries()
		f.hbScript = []int64{0} // 执行期间租约未丢,终态前那次续租命中 0 行:恰好被接管
		q := newQueue(f, map[Kind]Handler{
			KindDiagnose: func(context.Context, json.RawMessage) error { return errors.New("boom") },
		}, WithLease(time.Hour), WithHeartbeatInterval(time.Hour))

		f.seedPending(string(KindDiagnose), []byte(`{}`))
		if _, err := q.takeOne(context.Background()); err != nil {
			t.Fatalf("takeOne: %v", err)
		}
		if len(f.failed) != 0 || len(f.doneIDs) != 0 {
			t.Errorf("租约已易主不得写终态: done=%v failed=%+v", f.doneIDs, f.failed)
		}
		if got := f.job(1).status; got != statusRunning {
			t.Errorf("status = %q, want running(归接管者)", got)
		}
	})

	t.Run("闸门 1 行:仍持有租约,正常落失败", func(t *testing.T) {
		f := newFakeQueries()
		q := newQueue(f, map[Kind]Handler{
			KindDiagnose: func(context.Context, json.RawMessage) error { return errors.New("boom") },
		}, WithLease(time.Hour), WithHeartbeatInterval(time.Hour))

		f.seedPending(string(KindDiagnose), []byte(`{}`))
		if _, err := q.takeOne(context.Background()); err != nil {
			t.Fatalf("takeOne: %v", err)
		}
		if len(f.failed) != 1 || f.failed[0].LastError.String != "boom" {
			t.Errorf("持有租约时应落失败: %+v", f.failed)
		}
		if len(f.doneIDs) != 0 {
			t.Errorf("失败路径不应落完成: %v", f.doneIDs)
		}
	})
}
