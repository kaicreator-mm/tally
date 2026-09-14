package core

import "time"

// ProjectID 标识一个项目。
type ProjectID string

// Fingerprint 是一次失败规范化后的指纹。
// 精确指纹(fp_exact)与模糊指纹(fp_fuzzy)都用这个类型,
// 但只有精确指纹能关联到已验证修复(见 docs/03-data-model.md,INV-2)。
type Fingerprint string

// ArtifactDigest 是一个已构建制品的 digest(如镜像 sha256)。
// 部署只认这个;没有 digest 的部署不允许发生。
type ArtifactDigest string

// ClauseHash 是一条需求条款某一版本的内容哈希。
// 所有追溯边指向它,而不是逻辑编号(INV-4)——
// 条款内容一变,旧边自动指向旧版本,失效传播不需要额外机制。
type ClauseHash string

// VersionCombo 记录一次拉起/部署当时的版本组合:
// 镜像 tag、包版本、运行时版本、OS 等。
// 以 JSONB 冗余存在事实面各表里——历史必须自包含,
// 以后清理旧数据时,老化检测不能跟着一起坏。
type VersionCombo map[string]string

// FileChange 是修复里的一项文件改动:改后文件的完整内容。
// 用完整内容而不是 patch:验证时执行端照写即可,不用引入 patch 引擎(铁律 6)。
type FileChange struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Fix 是一条已验证修复的内容部分(知识库三元组去掉验证证据)。
// 验证证据(何时、在哪个版本组合下验证)由 VerificationProof 单独携带,不放这里 ——
// 内容与证据分离,未验证的内容就无法冒充已验证(铁律 2)。
type Fix struct {
	Repro   string       `json:"repro"`   // 复现命令(重现这条失败用)
	Summary string       `json:"summary"` // 做了什么改动、为什么
	Changes []FileChange `json:"changes"` // 改了哪些文件(改后完整内容)
}

// FixHit 是精确指纹命中知识库的结果:修复内容 + 有效性边界。
// Stale 为 true 表示版本组合已对不上或被标了老化 —— 调用方必须把它降级成线索,
// 不能直接当答案用(见 internal/knowledge)。
type FixHit struct {
	Fix
	VerifiedAt    time.Time    `json:"verified_at"`
	VerifiedCombo VersionCombo `json:"verified_combo"`
	Stale         bool         `json:"stale"`
}

// Lead 是模糊指纹召回的线索:只有坐标与状态,没有任何修复内容。
// 模糊是「看起来像」,不等于是同一个问题 —— 只能给 LLM 提示方向,不能替代验证。
// 本类型故意不含 Fix 字段:从形状上断掉「模糊命中直接当修复用」的路。
type Lead struct {
	ExactFP       Fingerprint  `json:"exact_fp"`
	Status        string       `json:"status"`
	VerifiedCombo VersionCombo `json:"verified_combo"`
	Stale         bool         `json:"stale"`
}

// VerificationProof 是写入知识库所需的验证证据(铁律 2:未验证的不入库)。
//
// 唯一产生路径:internal/failure 的四道闸全部通过后返回(以后还有 environment
// 的健康检查)。具体类型的字段全部不导出、构造函数不导出 —— 产生者之外的包
// 构造不出带内容的值。接口放在 core,是为了 knowledge(消费方)与 failure
// (产生方)不互相导入就能共享它(业务包互不导入,协作由上层编排)。
type VerificationProof interface {
	// VerifiedAt 是验证通过的时刻。零值表示这不是闸流产出的证据。
	VerifiedAt() time.Time
	// VerifiedCombo 是验证当时的版本组合 —— 修复的有效性边界。
	VerifiedCombo() VersionCombo
	// GateRecord 是四道闸的机器记录(如 "g1=5/5red g2=5/5green ..."),不是 LLM 的分析。
	GateRecord() string
	// Valid 报告证据是否由闸门真实产出(零值/伪造值为 false)。
	Valid() bool
}
