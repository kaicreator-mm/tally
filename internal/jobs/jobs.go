package jobs

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"

	"tally/internal/store"
)

// Kind 是任务类型。固定枚举,不支持运行时注册新类型 —— 任务图是设计期定死的。
type Kind string

// 队列基建在 T00 就位;各任务类型在对应功能任务里接线(T01/T03/T06)。
const (
	KindEnvironmentUp      Kind = "environment_up"
	KindEnvironmentDown    Kind = "environment_down"    // T05b:界面一键推倒,入队不直接动环境
	KindEnvironmentRebuild Kind = "environment_rebuild" // T05b:界面一键重建
	KindDeploy             Kind = "deploy"
	KindDeployRollback     Kind = "deploy_rollback" // T05b:界面一键回滚
	KindDiagnose           Kind = "diagnose"
)

// EnvActionPayload 是环境动作(拉起 / 推倒 / 重建)的入队参数。
//
// Def 是环境定义的**仓库相对路径**(.tally/environments/*.yaml):执行端(worker)按自己的
// 内容仓库根目录解析它,读到的是执行时刻的最新定义 —— 定义内容一变哈希就变,
// 新旧拉起记录自然分开(INV-4)。payload 只带坐标,不嵌定义全文,避免队列里躺着过时的定义副本。
type EnvActionPayload struct {
	Project string `json:"project"` // 回显与日志用;定义文件是权威
	Name    string `json:"name"`    // 环境名
	Def     string `json:"def"`     // 环境定义仓库相对路径,必填
}

// DeployPayload 是部署的入队参数:按 Def 指定的部署定义,把定义里已构建的制品部到目标环境。
// 与环境动作同理,定义在执行时刻按内容仓库根目录重新读取(执行到的是最新定义,INV-4);
// 制品 digest 以定义为准 —— 部署的对象是已存在的制品,不接受「拉代码再构建」(PRD 第二部分规则 ①)。
type DeployPayload struct {
	Project string `json:"project"` // 回显与日志用;定义文件是权威
	Def     string `json:"def"`     // 部署定义仓库相对路径,必填
}

// DeployRollbackPayload 是回滚的入队参数:把 Digest(通常是上一个能跑的制品)经 Def 指定的
// 部署定义部回 TargetEnv。执行端解析定义后覆盖 target_env / artifact_digest 再调工具回滚。
type DeployRollbackPayload struct {
	Project   string `json:"project"`
	TargetEnv string `json:"target_env"`
	Digest    string `json:"digest"` // 目标制品 digest:回滚就是把它再部一次
	Def       string `json:"def"`    // 部署定义仓库相对路径,必填
}

// DiagnosePayload 是重诊断的入队参数:对哪个精确指纹重新走一遍诊断。
// 三轮预算由入队侧(界面后端)先守,超了根本不入队;worker 执行时还会再查一次预算。
type DiagnosePayload struct {
	Fingerprint string `json:"fingerprint"`
	Project     string `json:"project"` // 回显与日志用;指纹是权威
}

// Handler 执行一种任务。payload 是该任务的 JSON 参数。
type Handler func(ctx context.Context, payload json.RawMessage) error

// querier 是队列依赖的查询面:只收队列实际调用的租约原语与终态写入
// (P0-01 / PRD §17.1 Job Lease)。*store.Queries 天然满足;测试用内存实现替换,不依赖真库。
type querier interface {
	InsertJob(ctx context.Context, arg store.InsertJobParams) (int64, error)
	ClaimNextJob(ctx context.Context, arg store.ClaimNextJobParams) (store.ClaimNextJobRow, error)
	HeartbeatJob(ctx context.Context, arg store.HeartbeatJobParams) (int64, error)
	MarkJobDone(ctx context.Context, id int64) error
	MarkJobFailed(ctx context.Context, arg store.MarkJobFailedParams) error
}

// Queue 是 Postgres 表 + 进程内 worker pool。
//
// 租约合同(PRD §17.1 / migrations/0014):认领(ClaimNextJob)单条 UPDATE 内嵌
// FOR UPDATE SKIP LOCKED,原子地取 pending 或 stale 的 running 行并写入 owner+租约;
// 执行期间按间隔心跳续租(HeartbeatJob),命中 0 行即租约丢失(已被接管者认领或过期),
// 必须中止执行且不写任何终态;取消(队列关闭/上下文取消)不是完成,保持 running
// 由租约过期后的 stale 谓词回收重跑。活租约永不被认领/回收谓词触碰。
type Queue struct {
	queries     querier
	handlers    map[Kind]Handler
	concurrency int
	pollDelay   time.Duration
	lease       time.Duration // 租约窗口:认领即写入,心跳按 heartbeat 续
	heartbeat   time.Duration // 心跳间隔:须小于租约,否则按租约/3 收紧
	workerID    string        // 本 worker 实例标识:认领/心跳的租约持有人
}

// Option 配置 Queue。
type Option func(*Queue)

// 租约默认:60s 窗口、20s 心跳。部署/重建这类长任务靠持续续租保持活租约。
const (
	defaultLease     = 60 * time.Second
	defaultHeartbeat = 20 * time.Second
)

// WithConcurrency 设置 worker 并发度(默认 2:拉环境和跑构建都是重活,并发高了机器扛不住)。
func WithConcurrency(n int) Option {
	return func(q *Queue) {
		if n > 0 {
			q.concurrency = n
		}
	}
}

// WithPollDelay 设置队列空转的轮询间隔(默认 2s)。
func WithPollDelay(d time.Duration) Option {
	return func(q *Queue) {
		if d > 0 {
			q.pollDelay = d
		}
	}
}

// WithLease 设置租约窗口(默认 60s)。认领即写入,心跳按间隔持续续租。
func WithLease(d time.Duration) Option {
	return func(q *Queue) {
		if d > 0 {
			q.lease = d
		}
	}
}

// WithHeartbeatInterval 设置心跳间隔(默认租约/3)。间隔必须小于租约,
// 否则按租约/3 收紧 —— 心跳比租约还慢等于从不续租。
func WithHeartbeatInterval(d time.Duration) Option {
	return func(q *Queue) {
		if d > 0 {
			q.heartbeat = d
		}
	}
}

// New 建立队列;handlers 为 nil 时为空注册表。
func New(s *store.Store, handlers map[Kind]Handler, opts ...Option) *Queue {
	return newQueue(s.Queries(), handlers, opts...)
}

// newQueue 按给定查询面装配队列;测试用内存实现替换真库查询。
func newQueue(qs querier, handlers map[Kind]Handler, opts ...Option) *Queue {
	q := &Queue{
		queries:     qs,
		handlers:    handlers,
		concurrency: 2,
		pollDelay:   2 * time.Second,
		lease:       defaultLease,
		heartbeat:   defaultHeartbeat,
		workerID:    newWorkerID(),
	}
	for _, o := range opts {
		o(q)
	}
	if q.handlers == nil {
		q.handlers = map[Kind]Handler{}
	}
	if q.heartbeat <= 0 || q.heartbeat >= q.lease {
		q.heartbeat = q.lease / 3
	}
	if q.heartbeat <= 0 {
		q.heartbeat = time.Millisecond
	}
	return q
}

// newWorkerID 生成本 worker 实例标识(主机-进程-随机数):租约持有人坐标,
// 认领与心跳都以此匹配行归属。两个 worker 撞名会让 fencing 失效
// (一方的心跳续到另一方认领的行上),所以带随机数防重启复用。
func newWorkerID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "localhost"
	}
	var nonce [8]byte
	if _, err := crand.Read(nonce[:]); err != nil {
		// crypto/rand 极罕见地不可用时退化为时间戳;可读性无碍,fencing 仍按 owner 匹配。
		binary.LittleEndian.PutUint64(nonce[:], uint64(time.Now().UnixNano()))
	}
	return fmt.Sprintf("%s-%d-%s", host, os.Getpid(), hex.EncodeToString(nonce[:]))
}

// leaseInterval 把租约窗口转成 Postgres interval 入参。
func leaseInterval(d time.Duration) pgtype.Interval {
	return pgtype.Interval{Microseconds: d.Microseconds()}
}

// ownerID 返回本 worker 的租约持有人标识。
func (q *Queue) ownerID() pgtype.Text {
	return pgtype.Text{String: q.workerID, Valid: true}
}

// Enqueue 入队一个任务,返回 job id。
func (q *Queue) Enqueue(ctx context.Context, kind Kind, payload any) (int64, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("序列化 payload: %w", err)
	}
	id, err := q.queries.InsertJob(ctx, store.InsertJobParams{Kind: string(kind), Payload: raw})
	if err != nil {
		return 0, fmt.Errorf("入队 %s: %w", kind, err)
	}
	return id, nil
}

// EnqueueDiagnose 按精确指纹入队一个诊断任务,返回 job id。
// 界面手动重诊断、webhook/CLI/exec 的自动触发(T11e)共用这一组装点,避免各处重复拼 payload。
// 预算不在此预检(D-e2):worker 执行时 failure.Handle 对终态/超预算指纹零 LLM 短路。
func (q *Queue) EnqueueDiagnose(ctx context.Context, project, fingerprint string) (int64, error) {
	return q.Enqueue(ctx, KindDiagnose, DiagnosePayload{Fingerprint: fingerprint, Project: project})
}

// Run 启动 worker pool,阻塞直到 ctx 取消。
func (q *Queue) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for i := 0; i < q.concurrency; i++ {
		g.Go(func() error { return q.loop(ctx) })
	}
	return g.Wait()
}

func (q *Queue) loop(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		took, err := q.takeOne(ctx)
		if err != nil {
			slog.Error("取任务", "err", err)
		}
		if !took {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(q.pollDelay):
			}
		}
	}
}

// takeOne 认领并执行一个任务。没有可认领的任务返回 false。
func (q *Queue) takeOne(ctx context.Context) (bool, error) {
	// 认领即标记 running + 写 owner/租约:ClaimNextJob 是单条 UPDATE 内嵌
	// FOR UPDATE SKIP LOCKED 子查询,语句自身原子(隐式事务),并发 worker
	// 不争抢、不重复认领 —— 不再需要 0010 时代 DequeueJob+MarkJobRunning
	// 的显式事务配对(两语句才需要包在一个 Tx 里)。
	row, err := q.queries.ClaimNextJob(ctx, store.ClaimNextJobParams{
		WorkerID: q.ownerID(),
		Column2:  leaseInterval(q.lease),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("认领任务: %w", err)
	}

	kind := Kind(row.Kind)
	handler, ok := q.handlers[kind]
	if !ok {
		q.fail(ctx, row.ID, "未注册的任务类型: "+string(kind))
		return true, nil
	}
	q.execute(ctx, row, handler)
	return true, nil
}

// execute 运行 handler 并用租约心跳守护:fencing 式续租(HeartbeatJob 只续
// 当前 owner 的活租约)命中 0 行即租约丢失,取消执行上下文。
//
// 终态语义:
//   - 租约丢失:行已归属接管者(或等待回收),任何终态写入(含 done)都会
//     覆盖别人的执行 —— 一律不写;
//   - 取消(队列关闭/上下文取消/心跳中止):取消不是完成,也不伪造成失败
//     —— 保持 running,租约到期后由 stale 谓词回收重跑;
//   - 正常结束:先续租做最后一道租约闸门,仍持有租约才落 done/failed。
func (q *Queue) execute(parent context.Context, row store.ClaimNextJobRow, handler Handler) {
	hctx, cancel := context.WithCancel(parent)

	var lostLease atomic.Bool
	hbDone := make(chan struct{})
	go func() {
		defer close(hbDone)
		ticker := time.NewTicker(q.heartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-hctx.Done():
				return
			case <-ticker.C:
				rows, err := q.queries.HeartbeatJob(hctx, store.HeartbeatJobParams{
					ID:       row.ID,
					WorkerID: q.ownerID(),
					Column3:  leaseInterval(q.lease),
				})
				switch {
				case err != nil:
					// 单次心跳打不进库不立即放弃:租约谓词兜底 —— 租约真过期后,
					// 恢复后的第一次心跳必命中 0 行,届时按丢失处理。
					slog.Warn("任务心跳失败", "id", row.ID, "err", err)
				case rows == 0:
					// fencing:行已易主或租约已过期(过期不允许复活)。
					// 必须中止执行,任务由回收路径接管。
					lostLease.Store(true)
					slog.Warn("任务租约丢失,中止执行", "id", row.ID, "worker", q.workerID)
					cancel()
					return
				}
			}
		}
	}()

	herr := handler(hctx, json.RawMessage(row.Payload))
	cancelled := hctx.Err() != nil // handler 期间执行上下文被取消(队列关闭或租约丢失)
	cancel()                       // 停心跳
	<-hbDone                       // 心跳协程收尾后再读 lostLease/写终态,无竞态

	switch {
	case lostLease.Load():
		slog.Warn("任务租约丢失,不写终态", "id", row.ID)
	case cancelled || isCancellation(herr):
		slog.Warn("任务被取消,保持 running 等待回收", "id", row.ID, "err", herr)
	default:
		q.settle(parent, row, herr)
	}
}

// settle 写终态。落笔前先用同一条心跳续租做闸门:命中 1 行 = 租约仍归本 worker,
// 可以落终态;命中 0 行 = 租约已被接管/过期,任何终态写入都会覆盖接管者的执行。
func (q *Queue) settle(ctx context.Context, row store.ClaimNextJobRow, herr error) {
	rows, err := q.queries.HeartbeatJob(ctx, store.HeartbeatJobParams{
		ID:       row.ID,
		WorkerID: q.ownerID(),
		Column3:  leaseInterval(q.lease),
	})
	if err != nil {
		slog.Error("任务终态前续租失败,保持 running 等待回收", "id", row.ID, "err", err)
		return
	}
	if rows == 0 {
		slog.Warn("任务租约已易主,不写终态", "id", row.ID)
		return
	}
	if herr != nil {
		slog.Warn("任务失败", "kind", row.Kind, "id", row.ID, "err", herr)
		q.fail(ctx, row.ID, herr.Error())
		return
	}
	if err := q.queries.MarkJobDone(ctx, row.ID); err != nil {
		slog.Error("标记任务完成", "id", row.ID, "err", err)
	}
}

// isCancellation 判定 handler 错误是否为取消/超时(而非任务自身失败):
// 取消不落任何终态,与队列关闭/租约丢失同等对待,保持 running 由租约过期回收。
func isCancellation(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func (q *Queue) fail(ctx context.Context, id int64, msg string) {
	err := q.queries.MarkJobFailed(ctx, store.MarkJobFailedParams{
		ID:        id,
		LastError: pgtype.Text{String: msg, Valid: true},
	})
	if err != nil {
		slog.Error("标记任务失败", "id", id, "err", err)
	}
}
