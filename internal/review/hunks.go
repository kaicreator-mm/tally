// Package review 实现 PRD §19 Patch Review 的确定性前置阶段(P1-07):
// Deterministic Hunk Inventory。在任何规则归账/LLM 残差分类之前,先从
// ChangeSet 固定 revision(base..candidate)的 Git diff 完整、确定地枚举
// 全部改动文件与文本 hunk(INV-CHANGE-01:approval 前每个 changed hunk
// 都必须被记账;PRD §19.2 流水线第一段:Git Diff → Deterministic
// Hunk Inventory)。
//
// 边界(P1):
//   - 本包只做确定性解析与枚举:不分配 primary review bucket、不打
//     semantic tag、不做 Completeness Validator、Unattributed Changes
//     也不在这里判定 —— 这些是 PRD §19.2 的后续阶段;本包绝不调用任何
//     LLM(AC4:无 LLM 分类)。
//   - 输入是 Git 风格统一 diff 文本。解析失败一律报错,绝不静默丢弃
//     条目:被丢掉的 hunk 会制造「所有改动都已看过」的假覆盖感,正是
//     INV-CHANGE-01 要防的事。
//
// 确定性:同一 diff 输入永远得到同一 Inventory(文件按路径排序、hunk
// 保持 diff 顺序、hunk ID 由内容推导)。Inventory 是可重建的投影值
// (INV-STATE-01),不是 append-only 事实本身。
package review

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ErrBadDiff 是所有 diff 解析失败的哨兵错误:调用方用 errors.Is 判因。
// 解析失败意味着清单会不完整,宁可报错也不产出残缺 Inventory。
var ErrBadDiff = errors.New("无效的 Git diff 输入")

// FileChangeKind 是文件级改动的种类。删除/重命名/复制/二进制都必须
// 有条目表示,不允许从清单中消失(AC2)。
type FileChangeKind string

const (
	// ChangeModified 内容被修改(可能同时含 mode 变化)。
	ChangeModified FileChangeKind = "modified"
	// ChangeAdded 新增文件(old 侧为 /dev/null)。
	ChangeAdded FileChangeKind = "added"
	// ChangeDeleted 文件被删除(new 侧为 /dev/null)。整文件删除仍是
	// 文本 hunk(有 @@ 坐标),不是「无事发生」。
	ChangeDeleted FileChangeKind = "deleted"
	// ChangeRenamed 路径重命名(可伴随内容修改,此时仍带 hunk)。
	ChangeRenamed FileChangeKind = "renamed"
	// ChangeCopied 复制(可伴随内容修改)。
	ChangeCopied FileChangeKind = "copied"
)

// ContentKind 是条目的内容表示:文本、二进制、或纯元数据变化
// (纯 rename/mode/gitlink 指针变化,没有内容 delta)。
type ContentKind string

const (
	// ContentText 有文本 hunk 或 ---/+++ 内容侧(哪怕零 hunk:空文件
	// 新增也是内容事实)。
	ContentText ContentKind = "text"
	// ContentBinary 二进制内容变化(Binary files ... differ /
	// GIT binary patch)。
	ContentBinary ContentKind = "binary"
	// ContentNone 没有内容 delta:纯 rename/copy、纯 mode 变化或
	// gitlink 指针变化等元数据事实。
	ContentNone ContentKind = "none"
)

// HunkID 标识清单内的一个文本 hunk,对同一 ChangeSet revision 的反复
// 重建稳定不变(stable-in-review,AC1)。由文件路径 + hunk 坐标 +
// hunk 正文做 sha256 推导(取前 128 位十六进制):内容/坐标/落点一变
// ID 必变,相同改动重算必得同 ID。
type HunkID string

// Hunk 是一个文本 diff hunk:坐标 + section(扩展 @@ 头的函数上下文
// 提示)+ 逐行正文(保留 ' '/'+/'-'/'\\' 前缀与 "\ No newline" 行原样)。
// 不含任何 bucket/tag/分类 —— 分类属后续阶段。
type Hunk struct {
	ID       HunkID
	Path     string   // 所属文件在清单中的主键路径(见 FileEntry.Path)
	OldStart int      // old 侧起始行(1 起;纯新增时为 0)
	OldLines int      // old 侧行数(0 = 纯新增)
	NewStart int      // new 侧起始行(1 起;纯删除时为 0)
	NewLines int      // new 侧行数(0 = 纯删除)
	Section  string   // "@@ ... @@" 之后的提示文本,可为空
	Body     []string // hunk 正文行,原样保留
}

// FileEntry 是清单内的一个改动文件条目。Path/OldPath/NewPath 约定:
//   - Path:清单主键。deleted 取 old 侧路径,其余取 new 侧路径;
//   - OldPath:added 时为空串,否则为 old 侧路径;
//   - NewPath:deleted 时为空串,否则为 new 侧路径。
type FileEntry struct {
	Path        string
	OldPath     string
	NewPath     string
	Kind        FileChangeKind
	Content     ContentKind
	OldMode     string // 如 "100644";未声明为空串
	NewMode     string
	ModeChanged bool   // 新旧 mode 均已知且不同(如可执行位变化)
	Hunks       []Hunk // 文本 hunk,按 diff 顺序;binary/纯 rename 等为空
}

// Inventory 是一次完整确定性 hunk 清单:全部改动文件与全部文本 hunk
// (AC3)。文件按 Path 排序(输入顺序不是语义);同一 diff 重建必得
// 同一结果。纯值投影:可整体重建。
type Inventory struct {
	Files []FileEntry
}

// FileCount 是改动文件条目数(含 binary/rename/delete/mode-only 条目)。
func (inv Inventory) FileCount() int { return len(inv.Files) }

// HunkCount 是全部文本 hunk 数:任何后续归账的记账分母(INV-CHANGE-01)。
func (inv Inventory) HunkCount() int {
	n := 0
	for _, f := range inv.Files {
		n += len(f.Hunks)
	}
	return n
}

// Hunks 返回全部文本 hunk 的扁平视图:文件按 Path、hunk 按 diff 顺序,
// 确定不变。返回副本,修改返回值不影响清单本体。
func (inv Inventory) Hunks() []Hunk {
	out := make([]Hunk, 0, inv.HunkCount())
	for _, f := range inv.Files {
		out = append(out, f.Hunks...)
	}
	return out
}

// InventoryFromDiff 从 Git 风格统一 diff 文本构建确定性 hunk 清单。
// 输入应为某个 ChangeSet 固定 revision 对(base..candidate)的 git diff
// 输出。空输入得到空清单(没有改动可记账,不是错误);任何无法解析的
// 行都返回 ErrBadDiff 包装错误并带行号,绝不静默丢条目。
func InventoryFromDiff(diff string) (Inventory, error) {
	if strings.TrimSpace(diff) == "" {
		return Inventory{}, nil
	}
	p := &diffParser{lines: splitLines(diff)}
	var inv Inventory
	for {
		p.skipBlank()
		if p.eof() {
			break
		}
		line := p.peek()
		if !strings.HasPrefix(line, "diff --git ") {
			return Inventory{}, fmt.Errorf("%w: 第 %d 行应为 \"diff --git\" 条目,得到 %q",
				ErrBadDiff, p.pos+1, clip(line))
		}
		rawOld, rawNew, err := parseDiffGitHeader(strings.TrimPrefix(line, "diff --git "))
		if err != nil {
			return Inventory{}, fmt.Errorf("%w: 第 %d 行(diff --git):%v", ErrBadDiff, p.pos+1, err)
		}
		p.pos++
		entry, err := p.parseFileEntry(rawOld, rawNew)
		if err != nil {
			return Inventory{}, err
		}
		inv.Files = append(inv.Files, entry)
	}
	// 文件顺序不是语义:按主键路径稳定排序,同一 diff 必得同一清单。
	sort.SliceStable(inv.Files, func(i, j int) bool { return inv.Files[i].Path < inv.Files[j].Path })
	return inv, nil
}

// diffParser 是逐行游标解析器:只前进、不回溯;错误一律带 1 基行号。
type diffParser struct {
	lines []string
	pos   int
}

func (p *diffParser) eof() bool    { return p.pos >= len(p.lines) }
func (p *diffParser) peek() string { return p.lines[p.pos] }

func (p *diffParser) skipBlank() {
	for !p.eof() && strings.TrimSpace(p.lines[p.pos]) == "" {
		p.pos++
	}
}

func (p *diffParser) errf(pos int, format string, args ...any) error {
	return fmt.Errorf("%w: 第 %d 行:%s", ErrBadDiff, pos+1, fmt.Sprintf(format, args...))
}

// parseFileEntry 解析一个 diff --git 条目的剩余部分(游标已越过
// diff --git 行),收敛为恰好一个 FileEntry。条目边界是下一个
// "diff --git " 或输入结束。
func (p *diffParser) parseFileEntry(rawOld, rawNew string) (FileEntry, error) {
	// 路径初值来自 diff --git 行:binary/mode-only 等没有 ---/+++ 的
	// 条目也必须有可记账的主键(AC2/AC3)。
	e := FileEntry{
		Path:    rawNew,
		OldPath: rawOld,
		NewPath: rawNew,
		Kind:    ChangeModified,
	}
	var (
		renameFrom, renameTo string
		isRename, isCopy     bool
		isNewFile, isDeleted bool
		sawDashes            bool // 出现过 ---/+++ 内容侧
		binary               bool
		hunks                []Hunk
	)

	finish := func() (FileEntry, error) {
		switch {
		case isRename:
			e.Kind = ChangeRenamed
		case isCopy:
			e.Kind = ChangeCopied
		case isDeleted:
			e.Kind = ChangeDeleted
		case isNewFile:
			e.Kind = ChangeAdded
		}
		// rename/copy 头部是路径的最终权威(纯 rename 条目没有 ---/+++)。
		if renameFrom != "" {
			e.OldPath = renameFrom
		}
		if renameTo != "" {
			e.NewPath = renameTo
			e.Path = renameTo
		}
		switch {
		case isNewFile && isDeleted:
			return FileEntry{}, p.errf(p.pos-1, "条目同时声明新增与删除")
		case isNewFile:
			e.OldPath = ""
			e.Path = e.NewPath
		case isDeleted:
			e.NewPath = ""
			e.Path = e.OldPath
		}
		if e.Path == "" {
			return FileEntry{}, p.errf(p.pos-1, "条目缺少可记账的文件路径")
		}
		e.OldMode = strings.TrimSpace(e.OldMode)
		e.NewMode = strings.TrimSpace(e.NewMode)
		e.ModeChanged = e.OldMode != "" && e.NewMode != "" && e.OldMode != e.NewMode
		switch {
		case binary:
			e.Content = ContentBinary
		case len(hunks) > 0 || sawDashes:
			e.Content = ContentText
		default:
			// 零 hunk、无 ---/+++、非二进制:纯元数据事实(纯 rename、
			// 纯 mode 变化、gitlink 指针变化)。仍必须有条目,不允许消失(AC2)。
			e.Content = ContentNone
		}
		e.Hunks = hunks
		return e, nil
	}

	for {
		if p.eof() {
			return finish()
		}
		line := p.peek()
		pos := p.pos
		if strings.TrimSpace(line) == "" {
			// 条目间/头部的空行不是事实(git diff 本身不产生,
			// 但上游文本可能混入),跳过;hunk 正文由计数消费,不受影响。
			p.pos++
			continue
		}
		switch {
		case strings.HasPrefix(line, "diff --git "):
			return finish()

		case strings.HasPrefix(line, "new file mode "):
			isNewFile = true
			e.NewMode = strings.TrimPrefix(line, "new file mode ")
			p.pos++

		case strings.HasPrefix(line, "deleted file mode "):
			isDeleted = true
			e.OldMode = strings.TrimPrefix(line, "deleted file mode ")
			p.pos++

		case strings.HasPrefix(line, "old mode "):
			e.OldMode = strings.TrimPrefix(line, "old mode ")
			p.pos++

		case strings.HasPrefix(line, "new mode "):
			e.NewMode = strings.TrimPrefix(line, "new mode ")
			p.pos++

		case strings.HasPrefix(line, "rename from "):
			val, err := parseHeaderPath(strings.TrimPrefix(line, "rename from "))
			if err != nil {
				return FileEntry{}, p.errf(pos, "%v", err)
			}
			isRename, renameFrom = true, val
			p.pos++

		case strings.HasPrefix(line, "rename to "):
			val, err := parseHeaderPath(strings.TrimPrefix(line, "rename to "))
			if err != nil {
				return FileEntry{}, p.errf(pos, "%v", err)
			}
			isRename, renameTo = true, val
			p.pos++

		case strings.HasPrefix(line, "copy from "):
			val, err := parseHeaderPath(strings.TrimPrefix(line, "copy from "))
			if err != nil {
				return FileEntry{}, p.errf(pos, "%v", err)
			}
			isCopy, renameFrom = true, val
			p.pos++

		case strings.HasPrefix(line, "copy to "):
			val, err := parseHeaderPath(strings.TrimPrefix(line, "copy to "))
			if err != nil {
				return FileEntry{}, p.errf(pos, "%v", err)
			}
			isCopy, renameTo = true, val
			p.pos++

		case strings.HasPrefix(line, "similarity index "),
			strings.HasPrefix(line, "dissimilarity index "),
			strings.HasPrefix(line, "index "):
			p.pos++ // 信息性头部,不影响记账结构

		case strings.HasPrefix(line, "Binary files "):
			binary = true
			p.pos++

		case line == "GIT binary patch":
			binary = true
			p.pos++
			// 二进制 payload(literal/delta 的编码行)没有可靠行结构,
			// 消费到下一个 diff --git 条目或 EOF 为止。
			for !p.eof() && !strings.HasPrefix(p.peek(), "diff --git ") {
				p.pos++
			}

		case strings.HasPrefix(line, "--- "):
			oldPath, oldNull, err := parseContentSide(strings.TrimPrefix(line, "--- "), "a/")
			if err != nil {
				return FileEntry{}, p.errf(pos, "%v", err)
			}
			p.pos++
			if p.eof() || !strings.HasPrefix(p.peek(), "+++ ") {
				return FileEntry{}, p.errf(pos, "\"---\" 之后必须是 \"+++\" 行")
			}
			newPath, newNull, err := parseContentSide(strings.TrimPrefix(p.peek(), "+++ "), "b/")
			if err != nil {
				return FileEntry{}, p.errf(p.pos, "%v", err)
			}
			p.pos++
			sawDashes = true
			switch {
			case oldNull && !newNull:
				isNewFile = true
			case newNull && !oldNull:
				isDeleted = true
			}
			e.OldPath, e.NewPath = oldPath, newPath
			e.Path = newPath

		case strings.HasPrefix(line, "@@ "):
			h, err := p.parseHunk(e.Path)
			if err != nil {
				return FileEntry{}, err
			}
			hunks = append(hunks, h)

		default:
			return FileEntry{}, p.errf(pos, "无法识别的 diff 行 %q", clip(line))
		}
	}
}

// parseHunk 解析 "@@ -o[,ol] +n[,nl] @@ section" 头,按声明计数消费
// 正文行,再推导稳定 ID。计数未满足就遇到别的行 = diff 被截断,报错。
func (p *diffParser) parseHunk(path string) (Hunk, error) {
	pos := p.pos
	oldStart, oldLines, newStart, newLines, section, err := parseHunkHeader(p.peek())
	if err != nil {
		return Hunk{}, p.errf(pos, "%v", err)
	}
	p.pos++
	h := Hunk{Path: path, OldStart: oldStart, OldLines: oldLines,
		NewStart: newStart, NewLines: newLines, Section: section}

	oldLeft, newLeft := oldLines, newLines
	for oldLeft > 0 || newLeft > 0 {
		if p.eof() {
			return Hunk{}, p.errf(pos, "hunk 正文在消费完声明的行数前结束(截断的 diff)")
		}
		body := p.peek()
		switch {
		case body == "":
			// 统一 diff 中内容为空的上下文行可能被写成空行。
			oldLeft--
			newLeft--
		case body[0] == '\\':
			// "\ No newline at end of file":不计数,但属于 hunk 正文。
		case body[0] == ' ':
			oldLeft--
			newLeft--
		case body[0] == '-':
			oldLeft--
		case body[0] == '+':
			newLeft--
		default:
			return Hunk{}, p.errf(p.pos, "期望 hunk 正文行,得到 %q", clip(body))
		}
		h.Body = append(h.Body, body)
		p.pos++
	}
	h.ID = deriveHunkID(h)
	return h, nil
}

// parseHunkHeader 解析 hunk 头。行数计数缺省为 1(unified diff 惯例)。
func parseHunkHeader(line string) (oldStart, oldLines, newStart, newLines int, section string, err error) {
	rest, ok := strings.CutPrefix(line, "@@ ")
	if !ok {
		return 0, 0, 0, 0, "", fmt.Errorf("hunk 头必须以 \"@@ \" 开头,得到 %q", clip(line))
	}
	ranges, section, found := strings.Cut(rest, " @@")
	if !found {
		return 0, 0, 0, 0, "", fmt.Errorf("hunk 头缺少结尾 \" @@\",得到 %q", clip(line))
	}
	section = strings.TrimPrefix(section, " ") // 单个分隔空格不是提示文本的一部分
	oldPart, newPart, found := strings.Cut(ranges, " ")
	if !found {
		return 0, 0, 0, 0, "", fmt.Errorf("hunk 头缺少 new 侧范围,得到 %q", clip(line))
	}
	oldStart, oldLines, err = parseRange(oldPart, '-')
	if err != nil {
		return 0, 0, 0, 0, "", err
	}
	newStart, newLines, err = parseRange(newPart, '+')
	if err != nil {
		return 0, 0, 0, 0, "", err
	}
	return oldStart, oldLines, newStart, newLines, section, nil
}

// parseRange 解析 "-o[,ol]" / "+n[,nl]" 形式的 hunk 范围。
func parseRange(part string, sign byte) (start, lines int, err error) {
	if len(part) < 2 || part[0] != sign {
		return 0, 0, fmt.Errorf("hunk 范围 %q 缺少 %q 前缀", clip(part), string(sign))
	}
	startStr, countStr, hasCount := strings.Cut(part[1:], ",")
	if start, err = strconv.Atoi(startStr); err != nil || start < 0 {
		return 0, 0, fmt.Errorf("hunk 范围 %q 的起始行不是非负数字", clip(part))
	}
	lines = 1 // "@@ -3 +3 @@" 等价于 "@@ -3,1 +3,1 @@"
	if hasCount {
		if lines, err = strconv.Atoi(countStr); err != nil || lines < 0 {
			return 0, 0, fmt.Errorf("hunk 范围 %q 的行数不是非负数字", clip(part))
		}
	}
	return start, lines, nil
}

// deriveHunkID 从 path + 坐标 + 正文推导 stable-in-review ID(AC1):
// 同一 ChangeSet revision 重算必得同 ID;任何内容/坐标/落点变化必换 ID。
// 取 sha256 前 128 位:同一清单内 (path, 坐标) 互不重叠,冲突概率可忽略。
func deriveHunkID(h Hunk) HunkID {
	var b strings.Builder
	b.WriteString("review-hunk-v1\n")
	fmt.Fprintf(&b, "path:%s\n", h.Path)
	fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@%s\n", h.OldStart, h.OldLines, h.NewStart, h.NewLines, h.Section)
	for _, l := range h.Body {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return HunkID("hunk-" + hex.EncodeToString(sum[:])[:16])
}

// parseDiffGitHeader 解析 "diff --git <old> <new>" 的两个路径 token。
// git 对含特殊字符的路径用 C 风格引号;未加引号的路径不含空格,
// 按引号感知切分即无歧义。返回值已剥 a/、b/ 前缀(--no-prefix 无前缀,
// 原样保留)。
func parseDiffGitHeader(rest string) (oldPart, newPart string, err error) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", "", errors.New("diff --git 行缺少路径")
	}
	first, second := rest, ""
	if strings.HasPrefix(rest, `"`) {
		decoded, remainder, err := cUnquote(rest)
		if err != nil {
			return "", "", err
		}
		first, second = decoded, strings.TrimSpace(remainder)
		if strings.HasPrefix(second, `"`) {
			if second, _, err = cUnquote(second); err != nil {
				return "", "", err
			}
		}
	} else {
		i := strings.IndexByte(rest, ' ')
		if i < 0 {
			return "", "", fmt.Errorf("diff --git 行缺少第二个路径:%q", clip(rest))
		}
		first, second = rest[:i], strings.TrimSpace(rest[i+1:])
	}
	if first == "" || second == "" {
		return "", "", fmt.Errorf("diff --git 行缺少路径:%q", clip(rest))
	}
	return strings.TrimPrefix(first, "a/"), strings.TrimPrefix(second, "b/"), nil
}

// parseHeaderPath 解析单路径头部值(rename/copy from/to;无 a/ b/ 前缀,
// 可为 C 引号形式)。
func parseHeaderPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("头部路径为空")
	}
	if strings.HasPrefix(raw, `"`) {
		decoded, _, err := cUnquote(raw)
		return decoded, err
	}
	return raw, nil
}

// parseContentSide 解析 ---/+++ 行的路径侧:先剥制表符时间戳,识别
// /dev/null,再剥 prefix(--- 侧 "a/"、+++ 侧 "b/";--no-prefix 无前缀,
// 原样保留)。devNull=true 表示该侧为 /dev/null。
func parseContentSide(raw, prefix string) (path string, devNull bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, errors.New("内容侧路径为空")
	}
	if i := strings.IndexByte(raw, '\t'); i >= 0 {
		raw = strings.TrimSpace(raw[:i])
	}
	if strings.HasPrefix(raw, `"`) {
		if raw, _, err = cUnquote(raw); err != nil {
			return "", false, err
		}
	}
	if raw == "" {
		return "", false, errors.New("内容侧路径为空")
	}
	if raw == "/dev/null" {
		return "", true, nil
	}
	return strings.TrimPrefix(raw, prefix), false, nil
}

// cUnquote 解码 git 的 C 风格引号路径(空格/制表/非 ASCII 的转义)。
// 返回解码值与闭引号之后的剩余文本。
func cUnquote(s string) (decoded, rest string, err error) {
	if !strings.HasPrefix(s, `"`) {
		return "", "", fmt.Errorf("路径缺少开引号:%q", clip(s))
	}
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			return b.String(), s[i+1:], nil
		case '\\':
			i++
			if i >= len(s) {
				return "", "", fmt.Errorf("路径引号未闭合:%q", clip(s))
			}
			switch e := s[i]; e {
			case 'a':
				b.WriteByte('\a')
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case 'v':
				b.WriteByte('\v')
			case '"', '\\':
				b.WriteByte(e)
			default:
				if e < '0' || e > '7' {
					return "", "", fmt.Errorf("不支持的路径转义 %q", string(e))
				}
				// 八进制转义最多 3 位(git 惯例 \NNN)。
				val := int(e - '0')
				for k := 0; k < 2 && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '7'; k++ {
					i++
					val = val*8 + int(s[i]-'0')
				}
				b.WriteByte(byte(val))
			}
		default:
			b.WriteByte(c)
		}
	}
	return "", "", fmt.Errorf("路径引号未闭合:%q", clip(s))
}

// splitLines 按 '\n' 切行;不去除行尾 '\r'(CRLF 文件的 \r 是内容,
// 原样保留才保得住正文保真)。末尾缺换行不影响最后一行。
func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// clip 截断诊断文本,避免错误信息吞下整行二进制 payload。
func clip(s string) string {
	if len(s) > 80 {
		return s[:80] + "..."
	}
	return s
}
