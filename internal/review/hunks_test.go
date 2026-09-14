package review

import (
	"errors"
	"strings"
	"testing"
)

// twoFileDiff 含两个文本文件、共 3 个 hunk,文件在输入中乱序,
// 用于验证排序、坐标、正文与计数。
const twoFileDiff = `diff --git a/zeta.txt b/zeta.txt
index 1111111..2222222 100644
--- a/zeta.txt
+++ b/zeta.txt
@@ -1,2 +1,2 @@ head
 ctx one
-removed line
+added line
@@ -20,2 +20,3 @@ tail
 ctx two
-older
+newer
+extra
diff --git a/alpha.txt b/alpha.txt
index 3333333..4444444 100644
--- a/alpha.txt
+++ b/alpha.txt
@@ -10,3 +10,4 @@ func main() {
 	if err != nil {
 		panic(err)
 	}
+	_ = extra
`

// AC3/AC4:清单包含全部文件与 hunk,文件按路径排序;解析纯确定性,
// 无任何 LLM/分类痕迹。
func TestInventoryFromDiffCollectsAllFilesAndHunks(t *testing.T) {
	inv, err := InventoryFromDiff(twoFileDiff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	if got := inv.FileCount(); got != 2 {
		t.Fatalf("FileCount = %d, want 2", got)
	}
	if got := inv.HunkCount(); got != 3 {
		t.Fatalf("HunkCount = %d, want 3", got)
	}
	// 文件顺序不是语义:按主键路径排序。
	files := inv.Files
	if files[0].Path != "alpha.txt" || files[1].Path != "zeta.txt" {
		t.Fatalf("Files 未按路径排序:%q, %q", files[0].Path, files[1].Path)
	}
	for _, f := range files {
		if f.Kind != ChangeModified || f.Content != ContentText {
			t.Errorf("%s: Kind/Content = %q/%q, want modified/text", f.Path, f.Kind, f.Content)
		}
		if f.OldPath != f.Path || f.NewPath != f.Path {
			t.Errorf("%s: OldPath/NewPath = %q/%q, 非重命名时应与 Path 相同", f.Path, f.OldPath, f.NewPath)
		}
	}
	// 扁平 hunk 视图遵循文件排序。
	hunks := inv.Hunks()
	if len(hunks) != 3 {
		t.Fatalf("Hunks() = %d 个, want 3", len(hunks))
	}
	if hunks[0].Path != "alpha.txt" || hunks[1].Path != "zeta.txt" || hunks[2].Path != "zeta.txt" {
		t.Errorf("Hunks() 顺序不对:%v", []string{hunks[0].Path, hunks[1].Path, hunks[2].Path})
	}
}

// AC1:文本 hunk 坐标、section 与正文被完整解析,且每个 hunk 得到
// 稳定 ID;重复重建 ID 不变,内容一变 ID 必变。
func TestHunkCoordinatesBodyAndStableID(t *testing.T) {
	inv, err := InventoryFromDiff(twoFileDiff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	zeta := inv.Files[1].Hunks
	if len(zeta) != 2 {
		t.Fatalf("zeta.txt hunk 数 = %d, want 2", len(zeta))
	}
	h := zeta[0]
	if h.OldStart != 1 || h.OldLines != 2 || h.NewStart != 1 || h.NewLines != 2 {
		t.Errorf("hunk0 坐标 = -%d,%d +%d,%d, want -1,2 +1,2",
			h.OldStart, h.OldLines, h.NewStart, h.NewLines)
	}
	if h.Section != "head" {
		t.Errorf("hunk0 Section = %q, want %q", h.Section, "head")
	}
	wantBody := []string{" ctx one", "-removed line", "+added line"}
	if len(h.Body) != len(wantBody) {
		t.Fatalf("hunk0 Body = %v, want %v", h.Body, wantBody)
	}
	for i, l := range wantBody {
		if h.Body[i] != l {
			t.Errorf("hunk0 Body[%d] = %q, want %q", i, h.Body[i], l)
		}
	}
	if !strings.HasPrefix(string(h.ID), "hunk-") || len(h.ID) != len("hunk-")+16 {
		t.Errorf("hunk0 ID = %q, 形应为 hunk-<16 位十六进制>", h.ID)
	}
	// alpha.txt 的 hunk:计数缺省写法之外的 +10,4 与上下文缩进保持原样。
	a := inv.Files[0].Hunks[0]
	if a.OldStart != 10 || a.OldLines != 3 || a.NewStart != 10 || a.NewLines != 4 {
		t.Errorf("alpha 坐标 = -%d,%d +%d,%d, want -10,3 +10,4",
			a.OldStart, a.OldLines, a.NewStart, a.NewLines)
	}
	if a.Section != "func main() {" {
		t.Errorf("alpha Section = %q", a.Section)
	}

	// 稳定性:同一 diff 重建,ID 逐一相同(AC1 stable-in-review)。
	all := inv.Hunks()
	again, err := InventoryFromDiff(twoFileDiff)
	if err != nil {
		t.Fatalf("InventoryFromDiff 重建: %v", err)
	}
	for i := range all {
		if again.Hunks()[i].ID != all[i].ID {
			t.Errorf("重建后 hunk %d ID = %q, want %q", i, again.Hunks()[i].ID, all[i].ID)
		}
	}
	// 敏感性:正文一行之差必须换 ID。
	changed := strings.Replace(twoFileDiff, "+added line", "+added line ", 1)
	inv2, err := InventoryFromDiff(changed)
	if err != nil {
		t.Fatalf("InventoryFromDiff(改一处正文): %v", err)
	}
	if inv2.Files[1].Hunks[0].ID == h.ID {
		t.Error("正文变化后 hunk ID 未变:ID 不是内容推导的")
	}
}

// AC1:同一清单内 hunk ID 互不相同;不同文件中的相同正文因路径参与
// 推导而得到不同 ID。
func TestHunkIDsUniqueWithinInventory(t *testing.T) {
	diff := `diff --git a/x.txt b/x.txt
--- a/x.txt
+++ b/x.txt
@@ -1,2 +1,2 @@
 same
-same line
+new line
diff --git a/y.txt b/y.txt
--- a/y.txt
+++ b/y.txt
@@ -1,2 +1,2 @@
 same
-same line
+new line
`
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	hunks := inv.Hunks()
	if len(hunks) != 2 {
		t.Fatalf("hunk 数 = %d, want 2", len(hunks))
	}
	if hunks[0].ID == hunks[1].ID {
		t.Error("不同文件的相同正文得到相同 ID:路径未参与推导")
	}
}

// AC2:新增文件(old 侧 /dev/null)必须表示为 added 条目,路径约定成立。
func TestAddedFileRepresented(t *testing.T) {
	diff := `diff --git a/new/pkg.go b/new/pkg.go
new file mode 100644
index 0000000..e69de29
--- /dev/null
+++ b/new/pkg.go
@@ -0,0 +1,2 @@
+package new
+
`
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	if inv.FileCount() != 1 || inv.HunkCount() != 1 {
		t.Fatalf("FileCount/HunkCount = %d/%d, want 1/1", inv.FileCount(), inv.HunkCount())
	}
	f := inv.Files[0]
	if f.Kind != ChangeAdded {
		t.Errorf("Kind = %q, want added", f.Kind)
	}
	if f.OldPath != "" || f.NewPath != "new/pkg.go" || f.Path != "new/pkg.go" {
		t.Errorf("路径约定破坏:Path/Old/New = %q/%q/%q", f.Path, f.OldPath, f.NewPath)
	}
	if h := f.Hunks[0]; h.OldStart != 0 || h.OldLines != 0 || h.NewStart != 1 || h.NewLines != 2 {
		t.Errorf("新增 hunk 坐标 = -%d,%d +%d,%d, want -0,0 +1,2",
			h.OldStart, h.OldLines, h.NewStart, h.NewLines)
	}
}

// AC2:删除文件必须表示为 deleted 条目,整文件删除仍是文本 hunk,
// 绝不允许「删了就消失」。
func TestDeletedFileRepresented(t *testing.T) {
	diff := `diff --git a/old/legacy.go b/old/legacy.go
deleted file mode 100644
index 9daeafb..0000000
--- a/old/legacy.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package legacy
-
-func Old() {}
`
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	f := inv.Files[0]
	if f.Kind != ChangeDeleted {
		t.Errorf("Kind = %q, want deleted", f.Kind)
	}
	if f.Path != "old/legacy.go" || f.OldPath != "old/legacy.go" || f.NewPath != "" {
		t.Errorf("路径约定破坏:Path/Old/New = %q/%q/%q", f.Path, f.OldPath, f.NewPath)
	}
	if f.Content != ContentText || len(f.Hunks) != 1 {
		t.Fatalf("删除应是文本 hunk:Content=%q, hunk 数=%d", f.Content, len(f.Hunks))
	}
	h := f.Hunks[0]
	if h.OldStart != 1 || h.OldLines != 3 || h.NewStart != 0 || h.NewLines != 0 {
		t.Errorf("删除 hunk 坐标 = -%d,%d +%d,%d, want -1,3 +0,0",
			h.OldStart, h.OldLines, h.NewStart, h.NewLines)
	}
	if len(h.Body) != 3 {
		t.Errorf("删除 hunk 正文行数 = %d, want 3(含空行删除)", len(h.Body))
	}
}

// AC2:纯重命名(零内容 delta)必须有条目,而不是被丢弃。
func TestPureRenameRepresented(t *testing.T) {
	diff := `diff --git a/docs/a.md b/docs/b.md
similarity index 100%
rename from docs/a.md
rename to docs/b.md
`
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	if inv.FileCount() != 1 {
		t.Fatalf("FileCount = %d, want 1", inv.FileCount())
	}
	f := inv.Files[0]
	if f.Kind != ChangeRenamed || f.Content != ContentNone {
		t.Errorf("Kind/Content = %q/%q, want renamed/none", f.Kind, f.Content)
	}
	if f.OldPath != "docs/a.md" || f.NewPath != "docs/b.md" || f.Path != "docs/b.md" {
		t.Errorf("rename 路径 = %q/%q/%q, want a.md/b.md/b.md", f.Path, f.OldPath, f.NewPath)
	}
	if len(f.Hunks) != 0 || inv.HunkCount() != 0 {
		t.Errorf("纯 rename 不应有 hunk,得到 %d", len(f.Hunks))
	}
}

// AC2:重命名伴随内容修改时,条目仍是 renamed 且带 hunk,
// hunk 归属重命名后的主键路径。
func TestRenameWithEditsKeepsHunks(t *testing.T) {
	diff := `diff --git a/docs/a.md b/docs/b.md
similarity index 80%
rename from docs/a.md
rename to docs/b.md
index 1111111..2222222 100644
--- a/docs/a.md
+++ b/docs/b.md
@@ -1 +1 @@
-old title
+new title
`
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	f := inv.Files[0]
	if f.Kind != ChangeRenamed || f.Content != ContentText {
		t.Errorf("Kind/Content = %q/%q, want renamed/text", f.Kind, f.Content)
	}
	if f.OldPath != "docs/a.md" || f.NewPath != "docs/b.md" {
		t.Errorf("rename 路径 = %q -> %q", f.OldPath, f.NewPath)
	}
	if len(f.Hunks) != 1 || f.Hunks[0].Path != "docs/b.md" {
		t.Fatalf("rename 后 hunk 应存在且归属 docs/b.md,得到 %+v", f.Hunks)
	}
}

// AC2:二进制变化(Binary files ... differ / GIT binary patch)必须以
// binary 条目表示;GIT binary patch 的 payload 不得吞掉后续条目。
func TestBinaryChangesRepresented(t *testing.T) {
	diff := `diff --git a/assets/logo.png b/assets/logo.png
index 1111111..2222222 100644
Binary files a/assets/logo.png and b/assets/logo.png differ
diff --git a/assets/raw.bin b/assets/raw.bin
index 3333333..4444444 100644
GIT binary patch
literal 16
YcmZQ$cccccccccccccccc

diff --git a/next/file.txt b/next/file.txt
index 5555555..6666666 100644
--- a/next/file.txt
+++ b/next/file.txt
@@ -1 +1 @@
-a
+b
`
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	if inv.FileCount() != 3 {
		t.Fatalf("FileCount = %d, want 3(二进制不能吞掉后续条目)", inv.FileCount())
	}
	for _, i := range []int{0, 1} {
		f := inv.Files[i]
		if f.Content != ContentBinary {
			t.Errorf("%s: Content = %q, want binary", f.Path, f.Content)
		}
		if len(f.Hunks) != 0 {
			t.Errorf("%s: 二进制条目不应有 hunk,得到 %d", f.Path, len(f.Hunks))
		}
	}
	last := inv.Files[2]
	if last.Path != "next/file.txt" || last.Content != ContentText || len(last.Hunks) != 1 {
		t.Errorf("payload 之后的条目解析受损:%+v", last)
	}
}

// AC2:纯 mode 变化(如可执行位)没有内容 delta,但必须有条目表示。
func TestModeOnlyChangeRepresented(t *testing.T) {
	diff := `diff --git a/run.sh b/run.sh
old mode 100644
new mode 100755
`
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	f := inv.Files[0]
	if f.Kind != ChangeModified || f.Content != ContentNone {
		t.Errorf("Kind/Content = %q/%q, want modified/none", f.Kind, f.Content)
	}
	if !f.ModeChanged || f.OldMode != "100644" || f.NewMode != "100755" {
		t.Errorf("mode = %q -> %q (changed=%v), want 100644 -> 100755 (true)",
			f.OldMode, f.NewMode, f.ModeChanged)
	}
	if len(f.Hunks) != 0 {
		t.Errorf("mode-only 不应有 hunk,得到 %d", len(f.Hunks))
	}
}

// AC2:copy from/to 与 rename 同理必须有条目,伴随的 hunk 保留。
func TestCopyRepresented(t *testing.T) {
	diff := `diff --git a/base/conf.yml b/copied/conf.yml
similarity index 90%
copy from base/conf.yml
copy to copied/conf.yml
index 1111111..2222222 100644
--- a/base/conf.yml
+++ b/copied/conf.yml
@@ -1,1 +1,2 @@
 key: a
+key: b
`
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	f := inv.Files[0]
	if f.Kind != ChangeCopied {
		t.Errorf("Kind = %q, want copied", f.Kind)
	}
	if f.OldPath != "base/conf.yml" || f.NewPath != "copied/conf.yml" {
		t.Errorf("copy 路径 = %q -> %q", f.OldPath, f.NewPath)
	}
	if len(f.Hunks) != 1 {
		t.Errorf("copy 伴随 hunk 数 = %d, want 1", len(f.Hunks))
	}
}

// AC2:gitlink/指针类条目(只有 index 行,零 hunk)是元数据事实,
// 必须表示为 modified/none 条目而不是被丢弃。
func TestMetadataOnlyEntryRepresented(t *testing.T) {
	diff := `diff --git a/vendor/lib b/vendor/lib
index abcdef1..2345678 160000
`
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	f := inv.Files[0]
	if f.Path != "vendor/lib" || f.Kind != ChangeModified || f.Content != ContentNone {
		t.Errorf("得到 %+v, want vendor/lib modified/none", f)
	}
	if inv.HunkCount() != 0 {
		t.Errorf("HunkCount = %d, want 0", inv.HunkCount())
	}
}

// 空输入(没有改动可记账)= 空清单,不是错误。
func TestEmptyDiffYieldsEmptyInventory(t *testing.T) {
	for _, in := range []string{"", "   \n  \n"} {
		inv, err := InventoryFromDiff(in)
		if err != nil {
			t.Fatalf("InventoryFromDiff(%q): %v", in, err)
		}
		if inv.FileCount() != 0 || inv.HunkCount() != 0 || len(inv.Hunks()) != 0 {
			t.Errorf("空输入得到非空清单:%d 文件 %d hunk", inv.FileCount(), inv.HunkCount())
		}
	}
}

// 完整性兜底:任何解析失败都必须报 ErrBadDiff 哨兵错误并整体拒绝,
// 绝不允许产出静默残缺的清单。
func TestMalformedDiffRejected(t *testing.T) {
	cases := []struct {
		name string
		diff string
	}{
		{"不是 diff 的输入", "hello world\n"},
		{"截断的 hunk 正文", `diff --git a/f.txt b/f.txt
--- a/f.txt
+++ b/f.txt
@@ -1,3 +1,3 @@
-only one line
`},
		{"hunk 头缺结尾", `diff --git a/f.txt b/f.txt
--- a/f.txt
+++ b/f.txt
@@ -1,2 +1,2
 old
`},
		{"hunk 头非法范围", `diff --git a/f.txt b/f.txt
--- a/f.txt
+++ b/f.txt
@@ -x,2 +1,2 @@
 old
`},
		{"--- 之后没有 +++", `diff --git a/f.txt b/f.txt
--- a/f.txt
`},
		{"正文外无法识别的行", `diff --git a/f.txt b/f.txt
<<<<<<< MERGE CONFLICT
`},
		{"引用路径未闭合", `diff --git "a/broken.txt b/broken.txt
--- "a/broken.txt"
+++ "b/broken.txt"
@@ -1 +1 @@
-old
+new
`},
	}
	for _, tc := range cases {
		inv, err := InventoryFromDiff(tc.diff)
		if err == nil {
			t.Errorf("%s: 应报错,得到清单(%d 文件)", tc.name, inv.FileCount())
			continue
		}
		if !errors.Is(err, ErrBadDiff) {
			t.Errorf("%s: 错误 %v 未包装 ErrBadDiff", tc.name, err)
		}
	}
}

// 计数缺省写法 "@@ -3 +3 @@" 等价于 "@@ -3,1 +3,1 @@";
// "\ No newline at end of file" 行不计入坐标计数但保留在正文中。
func TestHunkEdgeSyntax(t *testing.T) {
	diff := `diff --git a/f.txt b/f.txt
--- a/f.txt
+++ b/f.txt
@@ -3 +3 @@
-old
\ No newline at end of file
+new
`
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	h := inv.Files[0].Hunks[0]
	if h.OldStart != 3 || h.OldLines != 1 || h.NewStart != 3 || h.NewLines != 1 {
		t.Errorf("坐标 = -%d,%d +%d,%d, want -3,1 +3,1",
			h.OldStart, h.OldLines, h.NewStart, h.NewLines)
	}
	want := []string{"-old", `\ No newline at end of file`, "+new"}
	if len(h.Body) != len(want) {
		t.Fatalf("正文 = %v, want %v", h.Body, want)
	}
	for i := range want {
		if h.Body[i] != want[i] {
			t.Errorf("正文[%d] = %q, want %q", i, h.Body[i], want[i])
		}
	}
}

// git 对含空格/特殊字符的路径输出 C 引号形式;清单必须还原真实路径。
func TestQuotedPathsDecoded(t *testing.T) {
	diff := `diff --git "a/we ird/name.txt" "b/we ird/name.txt"
index 1111111..2222222 100644
--- "a/we ird/name.txt"
+++ "b/we ird/name.txt"
@@ -1 +1 @@
-old
+new
`
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	f := inv.Files[0]
	if f.Path != "we ird/name.txt" || f.OldPath != "we ird/name.txt" {
		t.Errorf("引号路径未还原:Path/OldPath = %q/%q", f.Path, f.OldPath)
	}
	if len(f.Hunks) != 1 || f.Hunks[0].Path != "we ird/name.txt" {
		t.Errorf("hunk 归属路径错误:%+v", f.Hunks)
	}
}

// 带时间戳的 ---/+++ 行(git diff 与 diff -u 常见输出)必须剥掉
// 时间戳再取路径。
func TestTimestampsStrippedFromContentSides(t *testing.T) {
	diff := `diff --git a/f.txt b/f.txt
--- a/f.txt	2026-01-02 03:04:05.000000000 +0000
+++ b/f.txt	2026-01-03 03:04:05.000000000 +0000
@@ -1 +1 @@
-old
+new
`
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	if f := inv.Files[0]; f.Path != "f.txt" {
		t.Errorf("Path = %q, want f.txt(时间戳必须剥离)", f.Path)
	}
}

// --- / +++ 的 /dev/null 侧在没有 new file mode / deleted file mode
// 头部时也必须推出 added/deleted(容错非 git 生成的内容侧标记)。
func TestDevNullSidesImplyAddAndDelete(t *testing.T) {
	add := `diff --git a/added.txt b/added.txt
index 0000000..1111111 100644
--- /dev/null
+++ b/added.txt
@@ -0,0 +1 @@
+content
`
	del := `diff --git a/gone.txt b/gone.txt
index 1111111..0000000 100644
--- a/gone.txt
+++ /dev/null
@@ -1 +0,0 @@
-content
`
	invAdd, err := InventoryFromDiff(add)
	if err != nil {
		t.Fatalf("InventoryFromDiff(add): %v", err)
	}
	if f := invAdd.Files[0]; f.Kind != ChangeAdded || f.OldPath != "" {
		t.Errorf("added 推导失败:%+v", f)
	}
	invDel, err := InventoryFromDiff(del)
	if err != nil {
		t.Fatalf("InventoryFromDiff(del): %v", err)
	}
	if f := invDel.Files[0]; f.Kind != ChangeDeleted || f.NewPath != "" {
		t.Errorf("deleted 推导失败:%+v", f)
	}
}

// CRLF 内容的 \r 属于正文,必须原样保留(保真优先于归一)。
func TestCRLFContentPreserved(t *testing.T) {
	diff := "diff --git a/win.txt b/win.txt\n" +
		"--- a/win.txt\n" +
		"+++ b/win.txt\n" +
		"@@ -1 +1 @@\r\n" +
		"-old\r\n" +
		"+new\r\n"
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	h := inv.Files[0].Hunks[0]
	if len(h.Body) != 2 {
		t.Fatalf("正文行数 = %d, want 2", len(h.Body))
	}
	if h.Body[1] != "+new\r" {
		t.Errorf("正文行 = %q, want %q(\\r 是内容,必须保留)", h.Body[1], "+new\r")
	}
}

// 空文件新增:有 ---/+++ 内容侧但零 hunk,也是内容事实(text/0 hunk)。
func TestEmptyAddedFileIsContentFact(t *testing.T) {
	diff := `diff --git a/empty.txt b/empty.txt
new file mode 100644
index 0000000..e69de29
--- /dev/null
+++ b/empty.txt
`
	inv, err := InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	f := inv.Files[0]
	if f.Kind != ChangeAdded || f.Content != ContentText || len(f.Hunks) != 0 {
		t.Errorf("得到 Kind/Content/hunks = %q/%q/%d, want added/text/0",
			f.Kind, f.Content, len(f.Hunks))
	}
}

// 确定性:条目间混入空行不影响结果(同一清单逐字段相等)。
func TestBlankLinesBetweenEntriesIgnored(t *testing.T) {
	withBlank := strings.Replace(twoFileDiff, "diff --git a/alpha.txt", "\ndiff --git a/alpha.txt", 1)
	a, err := InventoryFromDiff(twoFileDiff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	b, err := InventoryFromDiff(withBlank)
	if err != nil {
		t.Fatalf("InventoryFromDiff(带空行): %v", err)
	}
	if a.FileCount() != b.FileCount() || a.HunkCount() != b.HunkCount() {
		t.Fatalf("空行改变了清单:%d/%d vs %d/%d",
			a.FileCount(), a.HunkCount(), b.FileCount(), b.HunkCount())
	}
	for i := range a.Hunks() {
		if a.Hunks()[i].ID != b.Hunks()[i].ID {
			t.Errorf("空行改变了 hunk %d 的 ID", i)
		}
	}
}
