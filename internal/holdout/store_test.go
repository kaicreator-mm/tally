package holdout

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// memStore 是测试专用的最小内存实现,用于在本任务内演示抽象的语义。
// 真正的 Tally 控制 filesystem root 实现归 P1-27,不在本文件。
type memStore struct {
	mu   sync.Mutex
	docs map[Ref][]byte
	now  time.Time
}

func newMemStore() *memStore {
	return &memStore{docs: map[Ref][]byte{}, now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

// Put 落成记录并返回内容寻址的不透明 ref(内容寻址只是本 fake 的选择,
// 契约不要求 Put 的确定性)。
func (m *memStore) Put(_ context.Context, content []byte) (Ref, error) {
	if len(content) == 0 {
		return "", ErrInvalidContent
	}
	ref, err := ParseRef("ho-" + strings.TrimPrefix(ContentDigest(content), "sha256:"))
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.docs[ref] = slices.Clone(content)
	return ref, nil
}

func (m *memStore) Read(_ context.Context, ref Ref) ([]byte, error) {
	if _, err := ParseRef(ref.String()); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	doc, ok := m.docs[ref]
	if !ok {
		return nil, ErrNotFound
	}
	return slices.Clone(doc), nil
}

func (m *memStore) Stat(_ context.Context, ref Ref) (Metadata, error) {
	if _, err := ParseRef(ref.String()); err != nil {
		return Metadata{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	doc, ok := m.docs[ref]
	if !ok {
		return Metadata{}, ErrNotFound
	}
	return Metadata{Ref: ref, Size: int64(len(doc)), Digest: ContentDigest(doc), CreatedAt: m.now}, nil
}

var _ Store = (*memStore)(nil)

// sha256DigestRe 与 VerificationContract protection.baseline_hash 的格式约束一致。
var sha256DigestRe = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// storeMethods 返回接口类型的有序方法名,用于对 API 面做机械断言。
// 实参传 (*T)(nil) 形式的接口指针。
func storeMethods(t *testing.T, it any) []string {
	t.Helper()
	typ := reflect.TypeOf(it)
	if typ == nil || typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Interface {
		t.Fatalf("%T 不是接口指针", it)
	}
	ifc := typ.Elem()
	names := make([]string, 0, ifc.NumMethod())
	for i := 0; i < ifc.NumMethod(); i++ {
		names = append(names, ifc.Method(i).Name)
	}
	slices.Sort(names)
	return names
}

// AC1:Store API 用不透明 ref,而非项目文件系统路径。
func TestParseRefAcceptsOpaqueRefs(t *testing.T) {
	// 合法 ref:不透明 id 形态,包括验证契约 fixture 里 evidence.ref 会出现的
	// 「ref#selector」形态(types_test.go 的 "x#T")。
	for _, s := range []string{"x#T", "ho-0123abcd", "case-1_body", "a.b:c"} {
		ref, err := ParseRef(s)
		if err != nil {
			t.Fatalf("ParseRef(%q) = %v, want ok", s, err)
		}
		if ref.String() != s {
			t.Fatalf("ref.String() = %q, want 原样 %q(不派生路径)", ref.String(), s)
		}
	}
}

// AC1:任何能表达文件系统位置的串都被拒绝 —— ref 无法寻址实现 workspace 路径。
func TestParseRefRejectsPathSemantics(t *testing.T) {
	cases := []string{
		"",                               // 空
		".",                              // 路径相对分量
		"..",                             // 路径相对分量
		"a/b",                            // POSIX 分隔符
		`a\b`,                            // Windows 分隔符
		"/etc/passwd",                    // 绝对路径
		`C:\x\y`,                         // 盘符路径(含 ':' 合法但这里有分隔符)
		" x",                             // 前导空白
		"x ",                             // 尾随空白
		"line\nbreak",                    // 控制字符
		"nul\x00byte",                    // NUL
		"\x7f",                           // DEL
		strings.Repeat("a", MaxRefLen+1), // 超长
	}
	for _, s := range cases {
		if _, err := ParseRef(s); !errors.Is(err, ErrInvalidRef) {
			t.Fatalf("ParseRef(%q) err = %v, want ErrInvalidRef", s, err)
		}
	}
}

// AC1:接口面只收发 Ref 类型与内容字节,不出现任何路径形参/路径返回值。
func TestAPISignatureUsesRefsNotPaths(t *testing.T) {
	refType := reflect.TypeOf(Ref(""))
	strType := reflect.TypeOf("")
	for name, it := range map[string]any{
		"VerificationWriter": (*VerificationWriter)(nil),
		"VerificationReader": (*VerificationReader)(nil),
		"Store":              (*Store)(nil),
	} {
		ifc := reflect.TypeOf(it).Elem()
		for i := 0; i < ifc.NumMethod(); i++ {
			m := ifc.Method(i)
			mt := m.Type
			for j := 1; j < mt.NumIn(); j++ {
				if mt.In(j) == strType {
					// Ref 以类型化参数传递;字符串形参会让路径混进 API。
					t.Fatalf("%s.%s 有 string 形参;API 必须用 holdout.Ref", name, m.Name)
				}
			}
			for j := 0; j < mt.NumOut(); j++ {
				if out := mt.Out(j); out == strType {
					t.Fatalf("%s.%s 返回裸 string;位置信息不得出现在返回值里", name, m.Name)
				} else if out.Kind() == reflect.String && out != refType {
					t.Fatalf("%s.%s 返回未预期的命名字符串类型 %s", name, m.Name, out)
				}
			}
		}
	}
	// Metadata 不携带位置信息:没有 path/location 字段。
	mt := reflect.TypeOf(Metadata{})
	for i := 0; i < mt.NumField(); i++ {
		name := strings.ToLower(mt.Field(i).Name)
		if strings.Contains(name, "path") || strings.Contains(name, "location") {
			t.Fatalf("Metadata 出现位置字段 %s", mt.Field(i).Name)
		}
	}
}

// AC2:读能力按权威分离 —— 只接到 VerificationWriter 的实现权威侧接线在类型
// 层面没有任何读取方法;内容只能经 VerificationReader 解析。
func TestReadCapabilityIsSeparable(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	content := []byte("func TestHidden(t *testing.T) { t.Fatal(\"hidden verdict\") }")

	// 实现权威侧接线:只持有写能力面(此处用于断言其方法面;真实接线中
	// 实现权威组件连这个能力也拿不到)。
	var writerOnly VerificationWriter = store
	if got := storeMethods(t, &writerOnly); !slices.Equal(got, []string{"Put"}) {
		t.Fatalf("VerificationWriter 方法面 = %v, want [Put];不得存在读取路径", got)
	}

	// 验证权威侧:经 VerificationReader 才能解析内容。
	var reader VerificationReader = store
	ref, err := writerOnly.Put(ctx, content)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := reader.Read(ctx, ref)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !slices.Equal(got, content) {
		t.Fatalf("Read 内容不匹配: got %q", got)
	}
}

// AC2:只有 VerificationReader 能从 Ref 取回内容;Stat 提供完整性核对元数据。
func TestReaderRoundTripAndMetadata(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	content := []byte("held-out validator body")

	ref, err := store.Put(ctx, content)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	meta, err := store.Stat(ctx, ref)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if meta.Ref != ref {
		t.Fatalf("Stat.Ref = %v, want %v", meta.Ref, ref)
	}
	if meta.Size != int64(len(content)) {
		t.Fatalf("Stat.Size = %d, want %d", meta.Size, len(content))
	}
	if meta.Digest != ContentDigest(content) {
		t.Fatalf("Stat.Digest = %q, want %q", meta.Digest, ContentDigest(content))
	}
	if !sha256DigestRe.MatchString(meta.Digest) {
		t.Fatalf("Digest %q 不满足 sha256:<hex> 约定", meta.Digest)
	}
	if meta.CreatedAt.IsZero() {
		t.Fatal("Stat.CreatedAt 为零值")
	}

	// 未知 ref → ErrNotFound;非法 ref → ErrInvalidRef。
	if _, err := store.Read(ctx, Ref("ho-missing")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Read 未知 ref err = %v, want ErrNotFound", err)
	}
	if _, err := store.Stat(ctx, Ref("../escape")); !errors.Is(err, ErrInvalidRef) {
		t.Fatalf("Stat 非法 ref err = %v, want ErrInvalidRef", err)
	}
	if _, err := store.Put(ctx, nil); !errors.Is(err, ErrInvalidContent) {
		t.Fatalf("Put 空内容 err = %v, want ErrInvalidContent", err)
	}
}

// AC3:API 面就是隔离 + 最终执行的最小闭包,没有任何 P2 恢复/消费/
// 枚举工作流的入口。
func TestNoRecoveryOrEnumerationSurface(t *testing.T) {
	if got := storeMethods(t, (*Store)(nil)); !slices.Equal(got, []string{"Put", "Read", "Stat"}) {
		t.Fatalf("Store 方法面 = %v, want [Put Read Stat];不得引入恢复/枚举 API", got)
	}
	for name, it := range map[string]any{
		"VerificationWriter": (*VerificationWriter)(nil),
		"VerificationReader": (*VerificationReader)(nil),
	} {
		for _, m := range storeMethods(t, it) {
			lower := strings.ToLower(m)
			for _, banned := range []string{"list", "recover", "restore", "consume", "reset", "enumerate", "walk"} {
				if strings.Contains(lower, banned) {
					t.Fatalf("%s.%s 属于 P2 恢复/枚举面(决策锁 12)", name, m)
				}
			}
		}
	}
}

// ContentDigest 与 VerificationContract 的 protection.baseline_hash 同格式,
// 便于最终执行侧直接比对冻结哈希。
func TestContentDigestFormat(t *testing.T) {
	d := ContentDigest([]byte("x"))
	if !sha256DigestRe.MatchString(d) {
		t.Fatalf("ContentDigest = %q, want sha256:<64 hex>", d)
	}
	if ContentDigest([]byte("x")) != d {
		t.Fatal("ContentDigest 不稳定")
	}
	if ContentDigest([]byte("y")) == d {
		t.Fatal("不同内容得到相同摘要")
	}
}
