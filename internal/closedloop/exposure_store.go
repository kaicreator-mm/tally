// exposure_store.go 是 P2-06 ExposureSink 端口的 runstore 生产适配(P2-08
// 组合根持久化接线;取代事实侧见 lineage.go 的 NewStoreLineageSink)。
//
// 生产接线零桥接:internal/runstore 的 *Store 原生具备
// `RecordExposure(ctx, runstore.ExposureRecord) (evidence.Exposure, error)`
// 形态(append-only:evidence_exposure 只 INSERT,身份在落库时分配,
// UPDATE/DELETE 被语句级触发器拦截),因此 *Store 结构性满足本文件定义的
// 消费面,组合根直接 `NewStoreExposureSink(store, revision)` 即完成接线。
//
// 纪律(ADR-0001 §5;INV-V14-08):
//   - 只追加:适配器没有任何更新/删除方法;事实原样透传,不改写身份、
//     通道、时刻或 provenance;
//   - 透传不重写:RepairExposureSpec 的字段逐项映射,锚定候选经
//     evidence.ChangeSetID 换词表原样携带;
//   - revision 钉住:暴露事实按 (project, task, contract revision, binding)
//     归账;RepairExposureSpec 不携带 revision(闭环 Run 内恒为同一冻结
//     契约),由组合根在构造时钉住 —— 组合根换契约即换 Run;
//   - fail-safe:存储缺席或 revision 非法时拒绝构造,绝不产出静默丢弃
//     事实的出口。
package closedloop

import (
	"context"
	"errors"
	"fmt"

	"tally/internal/evidence"
	"tally/internal/runstore"
)

// ErrInvalidExposureStoreSink:runstore 暴露出口构造输入不合法。
// errors.Is 判因,不做字符串匹配。
var ErrInvalidExposureStoreSink = errors.New("closedloop: runstore 暴露出口构造不合法")

// ExposureFactAppender 是本适配消费的最小持久化面(小接口由消费方定义):
// 追加一条暴露事实并返回带已分配身份的领域事实。*runstore.Store 原生满足。
type ExposureFactAppender interface {
	RecordExposure(ctx context.Context, rec runstore.ExposureRecord) (evidence.Exposure, error)
}

// *runstore.Store 结构性满足消费面(编译期钉住;签名漂移即编译错误)。
var _ ExposureFactAppender = (*runstore.Store)(nil)

// NewStoreExposureSink 把 runstore 适配为闭环 ExposureSink。contractRevision
// 是冻结验证契约的 revision 号(≥1)。
func NewStoreExposureSink(store ExposureFactAppender, contractRevision int) (ExposureSink, error) {
	switch {
	case store == nil:
		return nil, ErrInvalidExposureStoreSink
	case contractRevision < 1:
		return nil, fmt.Errorf("%w: contract revision %d 非法(必须 >=1)", ErrInvalidExposureStoreSink, contractRevision)
	}
	return storeExposureSink{store: store, revision: int32(contractRevision)}, nil
}

// storeExposureSink 是 ExposureSink 的持久化适配:唯一职责是把追加调用
// 原样透传给存储。没有更新/删除方法;存储错误向上返回,由记账调用方裁决
// (exposure.go:未被记账的暴露绝不静默影响裁决)。
type storeExposureSink struct {
	store    ExposureFactAppender
	revision int32
}

// ExposureSink 端口编译期钉住。
var _ ExposureSink = storeExposureSink{}

// RecordExposure 实现 ExposureSink:RepairExposureSpec → runstore.ExposureRecord
// 逐项透传映射(零 ExposedAt 交由存储取当下,与 runstore 合同一致)。
func (s storeExposureSink) RecordExposure(ctx context.Context, spec RepairExposureSpec) (evidence.Exposure, error) {
	return s.store.RecordExposure(ctx, runstore.ExposureRecord{
		Task:             spec.TaskID,
		ContractRevision: s.revision,
		Binding:          spec.Binding,
		Acceptance:       spec.Acceptance,
		ChangeSet:        evidence.ChangeSetID(spec.Anchor),
		Channel:          spec.Channel,
		Reason:           spec.Reason,
		ExposedAt:        spec.ExposedAt,
	})
}
