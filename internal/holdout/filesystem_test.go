package holdout

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
)

// fsRefRe 是 FilesystemStore 产生的内容寻址 ref 形态:ho-<64 位十六进制>。
var fsRefRe = regexp.MustCompile(`^ho-[0-9a-f]{64}$`)

// newTestFilesystemStore 建立一个 store root 与受管 project root 互为兄弟目录
// 的 store(两者都在 t.TempDir() 下,天然满足 AC1 的外侧约束)。
func newTestFilesystemStore(t *testing.T) (*FilesystemStore, string) {
	t.Helper()
	project := t.TempDir()
	root := filepath.Join(t.TempDir(), "holdout")
	s, err := NewFilesystemStore(root, project)
	if err != nil {
		t.Fatalf("NewFilesystemStore(%s, %s): %v", root, project, err)
	}
	return s, project
}

// snapshotTree 返回 dir 下全部相对路径的集合,用于断言 tree 未被篡改。
func snapshotTree(t *testing.T, dir string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		out[rel] = true
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 %s: %v", dir, err)
	}
	return out
}

// ---- AC1:root 不能解析在实现 workspace/project root 之内 ----

func TestNewFilesystemStoreRejectsBadConfig(t *testing.T) {
	project := t.TempDir()
	if _, err := NewFilesystemStore("", project); !errors.Is(err, ErrInvalidRoot) {
		t.Fatalf("空 root err = %v, want ErrInvalidRoot", err)
	}
	if _, err := NewFilesystemStore("  ", project); !errors.Is(err, ErrInvalidRoot) {
		t.Fatalf("空白 root err = %v, want ErrInvalidRoot", err)
	}
	// 没有任何受管 root 约束时无法机械保证 AC1,构造必须拒绝。
	if _, err := NewFilesystemStore(filepath.Join(t.TempDir(), "holdout")); !errors.Is(err, ErrInvalidRoot) {
		t.Fatalf("缺少约束 root err = %v, want ErrInvalidRoot", err)
	}
}

func TestNewFilesystemStoreRejectsRootInsideProject(t *testing.T) {
	project := t.TempDir()
	cases := []struct {
		name string
		root string
	}{
		{"root 即 project root", project},
		{"root 在 project 之内(叶不存在)", filepath.Join(project, "holdout")},
		{"root 深层嵌套", filepath.Join(project, "a", "b", "holdout")},
		{"root 是 project 的父路径形态", project + string(filepath.Separator) + "." + string(filepath.Separator)},
	}
	if runtime.GOOS == "windows" {
		// NTFS 大小写不敏感:大小写变体也必须被拒绝。
		cases = append(cases, struct{ name, root string }{
			"root 大小写变体", strings.ToUpper(filepath.Join(project, "holdout"))})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewFilesystemStore(tc.root, project)
			if !errors.Is(err, ErrRootInsideProject) {
				t.Fatalf("NewFilesystemStore(%q) err = %v, want ErrRootInsideProject", tc.root, err)
			}
		})
	}
}

func TestNewFilesystemStoreRejectsRootInsideViaSymlink(t *testing.T) {
	project := t.TempDir()
	inner := filepath.Join(project, "inner")
	if err := os.MkdirAll(inner, fsDirPerm); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(inner, link); err != nil {
		t.Skipf("环境无法创建符号链接: %v", err)
	}
	if _, err := NewFilesystemStore(link, project); !errors.Is(err, ErrRootInsideProject) {
		t.Fatalf("经符号链接解析进 project 的 root err = %v, want ErrRootInsideProject", err)
	}
	// 正向:符号链接解析到 project 之外时必须被接受。
	outside := filepath.Join(t.TempDir(), "holdout")
	if err := os.MkdirAll(outside, fsDirPerm); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	outLink := filepath.Join(t.TempDir(), "out-link")
	if err := os.Symlink(outside, outLink); err != nil {
		t.Skipf("环境无法创建符号链接: %v", err)
	}
	if _, err := NewFilesystemStore(outLink, project); err != nil {
		t.Fatalf("外侧符号链接 root 被误拒: %v", err)
	}
}

func TestNewFilesystemStoreCreatesRootAndServesOutsideProject(t *testing.T) {
	s, project := newTestFilesystemStore(t)
	// root 与 objects 子目录都在构造期创建,配置错误不推迟到首次 Put。
	if !fsExists(filepath.Join(s.root, fsObjectsDir)) {
		t.Fatal("构造未创建 objects 目录")
	}
	ctx := context.Background()
	content := []byte("stored outside the project")
	ref, err := s.Put(ctx, content)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Read(ctx, ref)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !slices.Equal(got, content) {
		t.Fatalf("Read 内容不匹配: got %q", got)
	}
	// 受管 project root 内不得出现任何落盘痕迹。
	for rel := range snapshotTree(t, project) {
		if rel != "." {
			t.Fatalf("受管 project 内出现文件 %q", rel)
		}
	}
}

// ---- AC2:不透明 ref 只经验证权威 reader 解析 ----

func TestFilesystemStoreImplementsCapabilitySplits(t *testing.T) {
	s, _ := newTestFilesystemStore(t)
	if got := storeMethods(t, (*Store)(nil)); !slices.Equal(got, []string{"Put", "Read", "Stat"}) {
		t.Fatalf("Store 方法面 = %v", got)
	}
	// 实现权威侧的接线只持有窄写能力面:类型层面不存在任何读取路径。
	var writerOnly VerificationWriter = s
	if got := storeMethods(t, &writerOnly); !slices.Equal(got, []string{"Put"}) {
		t.Fatalf("VerificationWriter 方法面 = %v, want [Put]", got)
	}
	var reader VerificationReader = s
	ctx := context.Background()
	ref, err := writerOnly.Put(ctx, []byte("capability-split body"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := reader.Read(ctx, ref)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != "capability-split body" {
		t.Fatalf("Read = %q", got)
	}
}

func TestFilesystemStorePutReadStatRoundTrip(t *testing.T) {
	s, _ := newTestFilesystemStore(t)
	ctx := context.Background()
	content := []byte("held-out validator body")

	ref, err := s.Put(ctx, content)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if !fsRefRe.MatchString(ref.String()) {
		t.Fatalf("ref %q 不满足 ho-<64hex> 形态(内容寻址、无路径语义)", ref)
	}

	meta, err := s.Stat(ctx, ref)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if meta.Ref != ref || meta.Size != int64(len(content)) || meta.Digest != ContentDigest(content) {
		t.Fatalf("Stat 元数据不匹配: %+v", meta)
	}
	if meta.CreatedAt.IsZero() {
		t.Fatal("Stat.CreatedAt 为零值")
	}

	// 同一内容重复 Put 幂等:同 ref、CreatedAt 不被重写。
	ref2, err := s.Put(ctx, content)
	if err != nil {
		t.Fatalf("第二次 Put: %v", err)
	}
	if ref2 != ref {
		t.Fatalf("同一内容两次 Put 得到 %v 与 %v", ref, ref2)
	}
	meta2, err := s.Stat(ctx, ref)
	if err != nil {
		t.Fatalf("第二次 Stat: %v", err)
	}
	if !meta2.CreatedAt.Equal(meta.CreatedAt) {
		t.Fatalf("幂等 Put 重写了 CreatedAt: %v -> %v", meta.CreatedAt, meta2.CreatedAt)
	}

	// 不同内容得到不同 ref。
	other, err := s.Put(ctx, []byte("different held-out body"))
	if err != nil {
		t.Fatalf("Put(其他内容): %v", err)
	}
	if other == ref {
		t.Fatal("不同内容得到相同 ref")
	}
}

func TestFilesystemStoreErrors(t *testing.T) {
	s, _ := newTestFilesystemStore(t)
	ctx := context.Background()

	if _, err := s.Put(ctx, nil); !errors.Is(err, ErrInvalidContent) {
		t.Fatalf("Put 空内容 err = %v, want ErrInvalidContent", err)
	}
	// 合法但不存在的 ref → ErrNotFound;路径形态 ref → ErrInvalidRef。
	if _, err := s.Read(ctx, Ref("ho-"+strings.Repeat("0", 64))); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Read 未知 ref err = %v, want ErrNotFound", err)
	}
	if _, err := s.Stat(ctx, Ref("ho-"+strings.Repeat("f", 64))); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Stat 未知 ref err = %v, want ErrNotFound", err)
	}
	for _, ref := range []Ref{"../escape", `..\escape`, "a/b", ".", ".."} {
		if _, err := s.Read(ctx, ref); !errors.Is(err, ErrInvalidRef) {
			t.Fatalf("Read(%q) err = %v, want ErrInvalidRef", ref, err)
		}
		if _, err := s.Stat(ctx, ref); !errors.Is(err, ErrInvalidRef) {
			t.Fatalf("Stat(%q) err = %v, want ErrInvalidRef", ref, err)
		}
	}
}

// ---- AC3:traversal 被拒绝 ----

func TestFilesystemStoreTraversalCannotEscapeRoot(t *testing.T) {
	s, project := newTestFilesystemStore(t)
	ctx := context.Background()

	// ParseRef 拒绝一切路径形态的 ref(第一道防线)。
	for _, ref := range []Ref{"../escape", `..\escape`, "/etc/passwd", `C:\windows`, "a/b", `a\b`} {
		if _, err := s.Read(ctx, ref); !errors.Is(err, ErrInvalidRef) {
			t.Fatalf("Read(%q) err = %v, want ErrInvalidRef", ref, err)
		}
	}

	// 第二道防线:即使 ref 能通过 ParseRef(不含分隔符),ref → 文件名映射是
	// sha256(ref),任何 ref 都只能命中 objects 目录内的固定哈希名,无法寻址
	// root 之外。断言读失败且 root 与受管 project 的 tree 都原封不动。
	before, projectBefore := snapshotTree(t, s.root), snapshotTree(t, project)
	for _, ref := range []Ref{"....", "%2e%2e", "..%2f..", Ref(strings.Repeat(".", 300))} {
		if _, err := s.Read(ctx, ref); !errors.Is(err, ErrNotFound) {
			t.Fatalf("Read(%q) err = %v, want ErrNotFound", ref, err)
		}
		if _, err := s.Stat(ctx, ref); !errors.Is(err, ErrNotFound) {
			t.Fatalf("Stat(%q) err = %v, want ErrNotFound", ref, err)
		}
	}
	if !reflect.DeepEqual(snapshotTree(t, s.root), before) {
		t.Fatal("恶意 ref 改变了 store tree")
	}
	if !reflect.DeepEqual(snapshotTree(t, project), projectBefore) {
		t.Fatal("恶意 ref 在受管 project 内留下了文件")
	}
}

// ---- AC4:文件系统路径永不进入实现 ContextPack ----

func TestFilesystemStoreSurfaceExposesNoPaths(t *testing.T) {
	// 方法面按 *FilesystemStore 取(Put/Read/Stat 是指针接收者方法集)。
	ptrType := reflect.TypeOf((*FilesystemStore)(nil))
	names := make([]string, 0, ptrType.NumMethod())
	for i := 0; i < ptrType.NumMethod(); i++ {
		m := ptrType.Method(i)
		if !m.IsExported() {
			continue
		}
		names = append(names, m.Name)
		mt := m.Type
		for j := 0; j < mt.NumOut(); j++ {
			out := mt.Out(j)
			// 裸 string 返回值等于把路径语义放回 API;命名字符串类型只允许 holdout.Ref。
			if out == reflect.TypeOf("") {
				t.Fatalf("FilesystemStore.%s 返回裸 string;路径不得离开类型", m.Name)
			}
			if out.Kind() == reflect.String && out != reflect.TypeOf(Ref("")) {
				t.Fatalf("FilesystemStore.%s 返回未预期的命名字符串类型 %s", m.Name, out)
			}
		}
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"Put", "Read", "Stat"}) {
		t.Fatalf("FilesystemStore 导出方法面 = %v, want [Put Read Stat]", names)
	}
	// 全部字段未导出:root 与落盘路径在类型之外不可获得。
	typ := ptrType.Elem()
	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).IsExported() {
			t.Fatalf("FilesystemStore 出现导出字段 %s", typ.Field(i).Name)
		}
	}
}

func TestFilesystemStoreErrorsDoNotEchoStorageLocation(t *testing.T) {
	s, project := newTestFilesystemStore(t)
	ctx := context.Background()
	leaks := []string{s.root, s.objectsDir, project, fsObjectsDir, fsBlobSuffix, fsMetaSuffix}
	for name, call := range map[string]func() error{
		"Read 未知 ref": func() error { _, err := s.Read(ctx, Ref("ho-"+strings.Repeat("7", 64))); return err },
		"Stat 未知 ref": func() error { _, err := s.Stat(ctx, Ref("ho-"+strings.Repeat("8", 64))); return err },
		"Read 非法 ref": func() error { _, err := s.Read(ctx, Ref("../escape")); return err },
		"Stat 非法 ref": func() error { _, err := s.Stat(ctx, Ref("a/b")); return err },
		"Put 空内容":     func() error { _, err := s.Put(ctx, nil); return err },
		"Put 上下文取消": func() error {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err := s.Put(ctx, []byte("x"))
			return err
		},
	} {
		err := call()
		if err == nil {
			t.Fatalf("%s: 期望错误,得到 nil", name)
		}
		text := err.Error()
		for _, leak := range leaks {
			if strings.Contains(text, leak) {
				t.Fatalf("%s: 错误信息回显存放位置 %q: %s", name, leak, text)
			}
		}
	}
}

// ---- 并发:同一内容的并发 Put 幂等且无半成品 ----

func TestFilesystemStoreConcurrentPutIdempotent(t *testing.T) {
	s, _ := newTestFilesystemStore(t)
	ctx := context.Background()
	bodies := [][]byte{[]byte("held-out body A"), []byte("held-out body B")}
	const goroutines = 8

	refs := make([]Ref, goroutines)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ref, err := s.Put(ctx, bodies[i%len(bodies)])
			if err != nil {
				t.Errorf("Put: %v", err)
				return
			}
			refs[i] = ref
		}(i)
	}
	wg.Wait()

	want := make(map[Ref][]byte)
	for i, body := range bodies {
		ref, err := s.Put(ctx, body)
		if err != nil {
			t.Fatalf("串行 Put[%d]: %v", i, err)
		}
		want[ref] = body
	}
	for i, ref := range refs {
		body, ok := want[ref]
		if !ok {
			t.Fatalf("goroutine %d 得到意外 ref %v", i, ref)
		}
		got, err := s.Read(ctx, ref)
		if err != nil || !slices.Equal(got, body) {
			t.Fatalf("Read(%v) = %q, %v", ref, got, err)
		}
	}
}
