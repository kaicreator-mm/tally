package intentevidence

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// FrozenIntent 的 Git 制品形态(P0-01 对 DECISION_LOCKS「Open seams」中
// 「FrozenIntent exact file path / schema」seam 的裁决):
//
//   - 精确路径:.tally/intents/<task_id>.yaml。扁平目录、每任务一份 *.yaml,
//     与 .tally/tasks/TASK-*.yaml、.tally/verifications/TASK-*.yaml、
//     .tally/policies/*.yaml 完全同构(INV-ART-01 的 Git-first 制品布局);
//     制品身份以内容为准(task_id + Revision),文件名只是布局惯例 ——
//     同一策略/合同装载纪律的直接延续。
//   - 精确 schema(恰四字段,DL-06/任务要求 1):
//
//     task_id: TASK-0023          # 必填,不含空白
//     task_type: bug_fix          # 必填,恰为 bug_fix | db_migration
//     source_ref: GitHub Issue #142
//     risk_hints:                 # 可选;确认声明保留的风险提示原文
//     - schema_migration
//
//     schema 之外的任何字段(含 revision —— 哈希只能从内容重算,声明出来
//     的哈希按未知字段拒收,与 engineeringpolicy 同纪律)fail-closed 拒收;
//     provider/model/推理 prose 没有任何字段可写(要求 3)。
//   - 只接受单文档 YAML;空文档报错。与 taskcontract.Parse 同一纪律。

// canonicalVersion 是规范文本的版本前缀(与 taskcontract/v1、contract/v1
// 同一做法)。规范格式将来若变化,递增它,避免不同代格式的哈希空间重叠。
const canonicalVersion = "intent/v1"

// ArtifactDir 是 FrozenIntent 制品目录(见上)。测试与装载方(P0-02)都以
// 本常量为准,不得散落字面量。
const ArtifactDir = ".tally/intents"

// ArtifactPath 返回任务身份对应的制品路径(.tally/intents/<task_id>.yaml)。
// task_id 为空或含空白时报错:路径由身份派生,身份非法就不存在合法路径。
func ArtifactPath(taskID string) (string, error) {
	id := strings.TrimSpace(taskID)
	switch {
	case id == "":
		return "", fmt.Errorf("%w: 缺少 task_id", ErrInvalidEvidence)
	case strings.ContainsAny(id, " \t\r\n"):
		return "", fmt.Errorf("%w: task_id %q 含空白字符", ErrInvalidEvidence, taskID)
	}
	return ArtifactDir + "/" + id + ".yaml", nil
}

// artifactFile 是制品 YAML 的解码载体,字段与冻结 schema 一一对应。
// 刻意没有 revision/status 等任何附加字段:严格模式(KnownFields)会把
// schema 之外的键(包括「声明出来的哈希」)按未知字段拒收。
type artifactFile struct {
	TaskID    string   `yaml:"task_id"`
	TaskType  TaskType `yaml:"task_type"`
	SourceRef string   `yaml:"source_ref,omitempty"`
	RiskHints []string `yaml:"risk_hints,omitempty"`
}

// Canonical 返回证据的规范文本:Revision 的唯一输入。文本由规范化后的内容
// 按固定顺序排列,与 YAML 键序/列表顺序/折叠写法无关 —— 内容相同则文本
// 相同,任何实质差异都会改变文本(进而改变 Revision)。
func (f FrozenIntent) Canonical() string {
	var b strings.Builder
	b.WriteString(canonicalVersion)
	b.WriteByte('\n')
	fmt.Fprintf(&b, "task:%s\n", f.taskID)
	fmt.Fprintf(&b, "type:%s\n", f.taskType)
	if f.sourceRef != "" {
		fmt.Fprintf(&b, "source:%s\n", f.sourceRef)
	}
	for _, h := range f.riskHints {
		fmt.Fprintf(&b, "hint:%s\n", h)
	}
	return b.String()
}

// Render 返回证据的规范制品字节(确定性 YAML)。同一份证据无论渲染多少次、
// 在哪个进程渲染,字节都相同;Parse(Render(f)) 恒等于 f(往返确定性,
// 验收标准 1)。零值不是证据,渲染失败(fail closed)。
func (f FrozenIntent) Render() ([]byte, error) {
	if f.IsZero() {
		return nil, fmt.Errorf("%w: 零值不是冻结意图证据,拒绝渲染", ErrInvalidEvidence)
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(artifactFile{
		TaskID:    f.taskID,
		TaskType:  f.taskType,
		SourceRef: f.sourceRef,
		RiskHints: f.RiskHints(),
	}); err != nil {
		enc.Close()
		return nil, fmt.Errorf("%w: YAML 编码: %w", ErrInvalidEvidence, err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("%w: YAML 编码收尾: %w", ErrInvalidEvidence, err)
	}
	return buf.Bytes(), nil
}

// Parse 从制品字节解析并冻结一份 FrozenIntent。解析纪律(确定性,不猜):
//
//   - 严格模式:schema 之外的键(治理字段、provider/model prose、声明出来的
//     revision 等)按未知字段 fail-closed 拒收 —— 「声明了什么」与「生效了
//     什么」不得分叉;
//   - 只接受单文档 YAML;空文档报错;
//   - 内容经与 Freeze 完全相同的规范化 + 校验 + 哈希计算:Parse(Render(f))
//     恒等于 f,Revision 永远绑定实际内容,不存在声明/观测漂移的通道。
func Parse(data []byte) (FrozenIntent, error) {
	var af artifactFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // 未知字段是错误,不是可忽略数据
	if err := dec.Decode(&af); err != nil {
		if errors.Is(err, io.EOF) {
			return FrozenIntent{}, fmt.Errorf("%w: 空 YAML 文档", ErrInvalidEvidence)
		}
		return FrozenIntent{}, fmt.Errorf("%w: YAML 解析: %w", ErrInvalidEvidence, err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return FrozenIntent{}, fmt.Errorf("%w: 只接受单文档 YAML", ErrInvalidEvidence)
	}
	fi, err := Freeze(Facts{
		TaskID:    af.TaskID,
		TaskType:  af.TaskType,
		SourceRef: af.SourceRef,
		RiskHints: af.RiskHints,
	})
	if err != nil {
		return FrozenIntent{}, err
	}
	return fi, nil
}
