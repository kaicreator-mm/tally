package taskcontract

import (
	"strings"
	"testing"
)

// ValidateStatic 是 §7.4「Tally static validation」/§31.1 Proposal Admission
// 的确定性子集:合法合同必须通过,PRD 形态之外的 ID/refs/scope 一律拒绝,
// 且同一输入永远得到同一结论(纯函数,无 LLM/文件系统/网络)。

// 合法基线:PRD §7.3 示例、最小 standard 合同、无 acceptance 的 light 合同
// 都必须通过静态校验。
func TestValidateStaticAcceptsValidContracts(t *testing.T) {
	for _, src := range []string{prdExample, minimalExample} {
		c := mustParse(t, src)
		if err := c.ValidateStatic(); err != nil {
			t.Fatalf("合法合同应通过静态校验: %v", err)
		}
	}
	light := mustParse(t, minimalExample)
	light.Profile = ProfileLight
	light.Acceptance = nil
	if err := light.ValidateStatic(); err != nil {
		t.Fatalf("light 无 acceptance 应通过: %v", err)
	}
}

// AC1:非法 ID / 空 objective / 缺 acceptance 必须被拒绝。
func TestValidateStaticRejectsInvalidIDs(t *testing.T) {
	for _, id := range []string{
		"task-023", // 小写
		"TASK_023", // 下划线
		"TASK-",    // 缺编号
		"TASK-ABC", // 编号不是数字
		"TASK-23A", // 混入字母
		"TASK--1",  // 双连字符
		"23",       // 缺前缀
	} {
		c := mustParse(t, prdExample)
		c.ID = id
		err := c.ValidateStatic()
		if err == nil || !strings.Contains(err.Error(), "TASK-<数字>") {
			t.Errorf("id %q 应被拒绝, got %v", id, err)
		}
	}

	// 空 objective(结构硬约束,经 Validate 委托拒绝)。
	c := mustParse(t, prdExample)
	c.Objective = "   "
	if err := c.ValidateStatic(); err == nil || !strings.Contains(err.Error(), "objective") {
		t.Fatalf("空 objective 应被拒绝, got %v", err)
	}
	// standard 缺 acceptance(PRD §7.2)。
	c = mustParse(t, prdExample)
	c.Acceptance = nil
	if err := c.ValidateStatic(); err == nil || !strings.Contains(err.Error(), "acceptance") {
		t.Fatalf("standard 缺 acceptance 应被拒绝, got %v", err)
	}
}

// depends_on 条目必须是合法任务 id,且不得自引用(单合同内可判定的最小
// DAG 检查,§31.1)。合法前置(TASK-019)在 TestValidateStaticAcceptsValidContracts 覆盖。
func TestValidateStaticDependsOn(t *testing.T) {
	for _, dep := range []string{"TASK19", "task-019", "REQ-018", "TASK-019b", "TASK-"} {
		c := mustParse(t, prdExample)
		c.DependsOn = []string{dep}
		if err := c.ValidateStatic(); err == nil || !strings.Contains(err.Error(), dep) {
			t.Errorf("depends_on %q 应被拒绝, got %v", dep, err)
		}
	}
	c := mustParse(t, prdExample)
	c.DependsOn = []string{c.ID} // TASK-023 依赖 TASK-023
	if err := c.ValidateStatic(); err == nil || !strings.Contains(err.Error(), "自身") {
		t.Fatalf("自引用 depends_on 应被拒绝, got %v", err)
	}
}

// acceptance 条款 id 固定为 AC-<数字> 形态(PRD §7.3)。
func TestValidateStaticAcceptanceIDForm(t *testing.T) {
	for _, id := range []string{"ac-01", "AC_01", "AC", "AC-a", "clause-1", "AC-1-a"} {
		c := mustParse(t, prdExample)
		c.Acceptance[0].ID = id
		if err := c.ValidateStatic(); err == nil || !strings.Contains(err.Error(), "AC-<数字>") {
			t.Errorf("条款 id %q 应被拒绝, got %v", id, err)
		}
	}
}

// AC3:条款 id 必须唯一,independence_required 必须是 none/source/information。
// (唯一性与合法性由 Validate 承担,这里确认静态校验确实拦截。)
func TestValidateStaticAcceptanceUniquenessAndIndependence(t *testing.T) {
	c := mustParse(t, prdExample)
	c.Acceptance[2].ID = "AC-01"
	if err := c.ValidateStatic(); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatalf("重复条款 id 应被拒绝, got %v", err)
	}

	c = mustParse(t, prdExample)
	c.Acceptance[0].IndependenceRequired = "absolute"
	if err := c.ValidateStatic(); err == nil || !strings.Contains(err.Error(), "independence_required") {
		t.Fatalf("非法 independence_required 应被拒绝, got %v", err)
	}

	// 三个合法等级都通过。
	c = mustParse(t, prdExample)
	for _, lvl := range []IndependenceLevel{IndependenceNone, IndependenceSource, IndependenceInformation} {
		c.Acceptance[0].IndependenceRequired = lvl
		if err := c.ValidateStatic(); err != nil {
			t.Fatalf("等级 %v 应合法: %v", lvl, err)
		}
	}
}

// refs 语法:必须位于 §6.1 对应类型目录下、扁平、Markdown、允许 #fragment;
// requirement/ADR 另有固定文件名文法。
func TestValidateStaticRefsSyntax(t *testing.T) {
	valid := map[string][]string{
		"product_refs": {
			".tally/product/PRD.md#job-reliability", // §7.3 原样
			".tally/product/PRD.md",                 // fragment 可省略
			".tally/product/ROADMAP.md",             // 目录下任意 Markdown
		},
		"requirement_refs": {
			".tally/requirements/REQ-018.md",
			".tally/requirements/REQ-018.md#acceptance",
			".tally/requirements/REQ-7.md", // 前导零不强制
		},
		"architecture_refs": {
			".tally/architecture/ADR-012.md",
			".tally/architecture/ARCHITECTURE.md", // §6.1 布局中的第二个合法文件
			".tally/architecture/ADR-012.md#decision",
		},
	}
	for field, refs := range valid {
		for _, ref := range refs {
			c := mustParse(t, prdExample)
			switch field {
			case "product_refs":
				c.ProductRefs = []string{ref}
			case "requirement_refs":
				c.RequirementRefs = []string{ref}
			case "architecture_refs":
				c.ArchitectureRefs = []string{ref}
			}
			if err := c.ValidateStatic(); err != nil {
				t.Errorf("%s %q 应合法, got %v", field, ref, err)
			}
		}
	}

	invalid := []struct {
		field   string
		ref     string
		wantErr string
	}{
		{"requirement_refs", ".tally/requirements/REQ-018", "文件名"},            // 缺 .md
		{"requirement_refs", ".tally/requirements/REQ-abc.md", "文件名"},         // 编号非数字
		{"requirement_refs", ".tally/requirements/RANDOM-018.md", "文件名"},      // 前缀不对
		{"architecture_refs", ".tally/architecture/adr-012.md", "文件名"},        // 小写
		{"architecture_refs", ".tally/architecture/NOTE-001.md", "文件名"},       // 既非 ADR 也非 ARCHITECTURE
		{"requirement_refs", "docs/requirements/REQ-018.md", "应位于"},           // 目录不对
		{"product_refs", ".tally/product/PRD.md#", "锚点"},                      // 空 fragment
		{"product_refs", ".tally/product/PRD.md#job reliability", "fragment"}, // fragment 含空白
		{"product_refs", ".tally/product/PRD.md#a#b", "fragment"},             // 多个 #
		{"requirement_refs", ".tally/requirements/sub/REQ-018.md", "子目录"},     // 布局扁平
		{"requirement_refs", "/etc/passwd", "仓库相对路径"},                         // 绝对路径
		{"requirement_refs", "../secrets/REQ-018.md", "仓库内"},                  // 逃出仓库
		{"requirement_refs", ".tally\\requirements\\REQ-018.md", "斜杠"},        // 反斜杠
		{"requirement_refs", ".tally/requirements/REQ-*.md", "通配符"},           // refs 指向真实文件
		{"requirement_refs", ".tally/product/PR D.md", "空白"},                  // 路径含空白
	}
	for _, tt := range invalid {
		c := mustParse(t, prdExample)
		c.ProductRefs = nil
		c.RequirementRefs = nil
		c.ArchitectureRefs = nil
		switch tt.field {
		case "product_refs":
			c.ProductRefs = []string{tt.ref}
		case "requirement_refs":
			c.RequirementRefs = []string{tt.ref}
		case "architecture_refs":
			c.ArchitectureRefs = []string{tt.ref}
		}
		err := c.ValidateStatic()
		if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("%s %q 应被拒绝(报错含 %q), got %v", tt.field, tt.ref, tt.wantErr, err)
		}
	}
}

// scope 条目语法:通配只允许整段 * 与末段 **,其余一律拒绝。
func TestValidateStaticScopeSyntax(t *testing.T) {
	// 合法形态:纯路径、末段 **、中间整段 *。
	for _, p := range []string{"internal/jobs/**", "migrations", "internal/*/auth", "cmd/server"} {
		c := mustParse(t, prdExample)
		c.Scope.AllowedPaths = []string{p}
		c.Scope.ForbiddenPaths = nil
		if err := c.ValidateStatic(); err != nil {
			t.Errorf("scope 条目 %q 应合法, got %v", p, err)
		}
	}

	for _, p := range []string{
		"internal/**/auth", // ** 不在末段
		"internal*/x",      // 通配符混在段内
		"internal/j?bs/**", // ? 通配
		"internal//jobs",   // 空段
		"/internal/**",     // 绝对路径
		"../secrets/**",    // 逃出仓库
		"C:\\repo\\**",     // 反斜杠/盘符
		"C:/repo/**",       // 盘符
		"internal /jobs",   // 段内空白
		"internal/**[x]",   // 其它通配字符
	} {
		c := mustParse(t, prdExample)
		c.Scope.AllowedPaths = []string{p}
		c.Scope.ForbiddenPaths = nil
		if err := c.ValidateStatic(); err == nil {
			t.Errorf("scope 条目 %q 应被拒绝", p)
		}
	}
}

// AC2:allowed 与 forbidden 覆盖到同一路径 = 自相矛盾合同,必须失败;
// 互不重叠的兄弟目录合法(forbidden 是硬边界,不是「允许范围减去禁区」)。
func TestValidateStaticScopeContradiction(t *testing.T) {
	// base:allowed = [internal/jobs/**, internal/store/**, migrations/**]
	tests := []struct {
		name      string
		forbidden string
		mutate    func(*TaskContract)
	}{
		{"条目完全相同", "internal/jobs/**", nil},
		{"forbidden 覆盖 allowed", "internal/**", nil},
		{"裸目录覆盖子路径", "internal", nil},
		{"裸文件覆盖所在目录", "migrations", nil},
		{"forbidden 覆盖裸 allowed", "", func(c *TaskContract) {
			c.Scope.AllowedPaths = []string{"migrations"}
			c.Scope.ForbiddenPaths = []string{"migrations/**"}
		}},
		{"单段通配命中禁区", "", func(c *TaskContract) {
			c.Scope.AllowedPaths = []string{"internal/*/**"}
			c.Scope.ForbiddenPaths = []string{"internal/auth/**"}
		}},
		{"裸禁区覆盖其中的 allowed 文件", "", func(c *TaskContract) {
			c.Scope.AllowedPaths = []string{"internal/auth/session.go"}
			c.Scope.ForbiddenPaths = []string{"internal/auth"}
		}},
	}
	for _, tt := range tests {
		c := mustParse(t, prdExample)
		if tt.mutate != nil {
			tt.mutate(&c)
		} else {
			c.Scope.ForbiddenPaths = []string{tt.forbidden}
		}
		err := c.ValidateStatic()
		if err == nil || !strings.Contains(err.Error(), "自相矛盾") {
			t.Errorf("%s(forbidden=%q) 应判矛盾, got %v", tt.name, tt.forbidden, err)
		}
	}

	// 不重叠:兄弟目录、无关前缀都合法。
	for _, forbidden := range []string{"internal/auth/**", "cmd/**", "internal/auth", "web/**"} {
		c := mustParse(t, prdExample)
		c.Scope.ForbiddenPaths = []string{forbidden}
		if err := c.ValidateStatic(); err != nil {
			t.Errorf("forbidden %q 与 allowed 不重叠,应通过: %v", forbidden, err)
		}
	}
}

// AC4:校验是确定性纯函数 —— 不需要 LLM 判断;同一输入重复执行结论一致,
// 且校验过程不修改合同内容(哈希不变)。
func TestValidateStaticDeterministic(t *testing.T) {
	c := mustParse(t, prdExample)
	first := c.ValidateStatic()
	hashBefore := c.RevisionHash()
	for i := 0; i < 3; i++ {
		if err := c.ValidateStatic(); (first == nil) != (err == nil) || (err != nil && err.Error() != first.Error()) {
			t.Fatalf("第 %d 次校验结论与首次不同: %v vs %v", i, err, first)
		}
	}
	if hashBefore != c.RevisionHash() {
		t.Fatal("ValidateStatic 修改了合同内容")
	}

	// 失败结论同样可复现:同一份矛盾合同,每次都拒绝。
	bad := mustParse(t, prdExample)
	bad.Scope.ForbiddenPaths = []string{"internal/jobs/**"}
	want := bad.ValidateStatic()
	if want == nil {
		t.Fatal("矛盾合同应被拒绝")
	}
	for i := 0; i < 3; i++ {
		if err := bad.ValidateStatic(); err == nil || err.Error() != want.Error() {
			t.Fatalf("第 %d 次拒绝结论与首次不同: %v vs %v", i, err, want)
		}
	}
}
