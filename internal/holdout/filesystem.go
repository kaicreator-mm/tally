// filesystem.go 实现 P1-27:Holdout Evidence Store 的 P1 具体 Store(PRD
// INV-VER-05 / §8.4.1 / §36 P1 Trust capability,决策锁 11)。
//
// 存储位置是一个配置的、Tally 拥有的 filesystem root,构造时机械保证它解析在
// 受管项目 repo / 实现 workspace 之外(AC1);ref → 文件名的映射是
// sha256(ref) 十六进制,与 ref 拼写、任何路径语义彻底解耦,配合 ParseRef
// 拒绝分隔符,任何 ref 都无法寻址 store root 之外的路径(AC3)。
//
// 边界(P1,沿用本包 store.go 抽象):
//   - Ref → 内容的解析只存在于 VerificationReader 能力之后(AC2):本类型把
//     Put/Read/Stat 提供给控制面,控制面按权威分发窄能力;blob/meta 的落盘
//     路径是未导出实现细节,没有任何导出方法返回文件系统路径(AC4),错误
//     信息对预期失败(不存在/非法 ref)不回显存放位置。
//   - 这是 P1 实现基线而非永久产品承诺:未来存储仍在本包 Store 接口之后替换。
package holdout

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// 哨兵错误:调用方用 errors.Is 区分;包装文案只提供字段级上下文。
var (
	// ErrInvalidRoot 表示 filesystem store 的 root 配置不合法(空、无法解析,
	// 或没有给出任何实现 workspace/project root 约束)。
	ErrInvalidRoot = errors.New("holdout: invalid filesystem store root")
	// ErrRootInsideProject 表示配置的 store root 解析在受管项目 repo /
	// 实现 workspace 之内(AC1);此时 store 不得建立,否则 held-out 内容
	// 会落在实现权威可达的路径上。
	ErrRootInsideProject = errors.New("holdout: filesystem store root resolves inside implementation workspace/project root")
)

const (
	// fsObjectsDir 是 root 下存放记录对象的子目录。
	fsObjectsDir = "objects"
	// fsBlobSuffix / fsMetaSuffix 是记录内容与元数据文件的固定后缀。
	fsBlobSuffix = ".blob"
	fsMetaSuffix = ".meta.json"
	// fsRefPrefix 是本实现产生的内容寻址 ref 前缀:ho-<64 位十六进制>。
	fsRefPrefix = "ho-"
	// fsTempPattern 是原子写临时文件名模式(位于 objects 目录内,同盘 rename)。
	fsTempPattern = "holdout-tmp-*"
	// fsDirPerm / fsFilePerm:Tally 拥有的私有目录与文件权限。
	fsDirPerm  os.FileMode = 0o700
	fsFilePerm os.FileMode = 0o600
)

// FilesystemStore 是决策锁 11 的 P1 具体 Store:内容与元数据存放在 Tally 拥有、
// 位于受管项目 repo / 实现 workspace 之外的 filesystem root 下。
//
// root 与每个记录的落盘路径是未导出字段/私有方法:类型没有任何导出途径把
// 文件系统路径交给调用方(AC4),实现权威侧接线按 store.go 的能力接口只会
// 拿到(也只能拿到)不透明 Ref。
type FilesystemStore struct {
	// root 是解析符号链接后的绝对 store root;永不导出、永不进入错误信息
	// 与返回值。
	root string
	// objectsDir 是 root 下的记录对象目录。
	objectsDir string
	// putMu 串行化 Put 的「检查存在性 → 原子写」序列,使并发 Put 幂等且
	// 不产生半成品记录。Read/Stat 只读已完整落成(ref 改名)的文件,无需加锁。
	putMu sync.Mutex
}

// 编译期断言:本类型实现完整 Store 面与两个窄能力面(AC2 的类型层落点)。
var (
	_ Store              = (*FilesystemStore)(nil)
	_ VerificationReader = (*FilesystemStore)(nil)
	_ VerificationWriter = (*FilesystemStore)(nil)
)

// NewFilesystemStore 建立以 root 为根的 filesystem Holdout Store。
//
// outside 列出受管项目 repo / 实现 workspace 的 root(至少一个):store root
// 必须解析在每一个之外,否则返回 ErrRootInsideProject(AC1,决策锁 11)。
// 内外比较在解析符号链接与 Windows 大小写规范化之后进行,调用方无法用
// 相对路径、符号链接或大小写差异让 root 落进受管范围。
//
// 构造会创建 root 及其 objects 子目录(权限 0700),使配置错误在接线期
// 而不是首次 Put 时暴露。构造期错误属于控制面诊断,可以携带路径;
// Put/Read/Stat 的错误则遵守 store.go 的「不回显存放位置」约束。
func NewFilesystemStore(root string, outside ...string) (*FilesystemStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("%w: root 不能为空", ErrInvalidRoot)
	}
	if len(outside) == 0 {
		return nil, fmt.Errorf("%w: 缺少受管项目/workspace root 约束(AC1)", ErrInvalidRoot)
	}
	realRoot, err := fsResolve(root)
	if err != nil {
		return nil, err
	}
	for _, p := range outside {
		realOutside, err := fsResolve(p)
		if err != nil {
			return nil, fmt.Errorf("%w: 解析受管 root: %v", ErrInvalidRoot, err)
		}
		if fsInsideOrEqual(realOutside, realRoot) {
			return nil, fmt.Errorf("%w: store root %s 解析在受管 root %s 之内",
				ErrRootInsideProject, realRoot, realOutside)
		}
	}
	objectsDir := filepath.Join(realRoot, fsObjectsDir)
	if err := os.MkdirAll(objectsDir, fsDirPerm); err != nil {
		return nil, fmt.Errorf("holdout: filesystem: 创建 store root: %w", err)
	}
	return &FilesystemStore{root: realRoot, objectsDir: objectsDir}, nil
}

// fsRecord 是一条记录落盘的元数据文件内容(<key>.meta.json)。
// 只含内容指纹与落成时间,不含任何存放位置信息。
type fsRecord struct {
	CreatedAt time.Time `json:"created_at"` // 记录落成时间(blob 的落盘时刻)
	Digest    string    `json:"digest"`     // ContentDigest 格式
	Size      int64     `json:"size"`       // 内容字节数
}

// Put 实现 VerificationWriter:内容寻址落盘并返回不透明 ref(ho-<sha256 hex>)。
// 同一内容重复 Put 幂等返回同一 ref(本实现的选择,契约不要求确定性);
// 空内容返回 ErrInvalidContent。写入经临时文件 + 同盘 rename,中断不产生半成品。
func (s *FilesystemStore) Put(ctx context.Context, content []byte) (Ref, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(content) == 0 {
		return "", ErrInvalidContent
	}
	digest := ContentDigest(content)
	ref, err := ParseRef(fsRefPrefix + strings.TrimPrefix(digest, "sha256:"))
	if err != nil {
		return "", err // 不可达:内容寻址 ref 恒为 ho-<64 位十六进制>
	}

	s.putMu.Lock()
	defer s.putMu.Unlock()

	blob := s.blobPath(ref)
	if !fsExists(blob) {
		if err := s.writeAtomic(objectKey(ref)+fsBlobSuffix, content); err != nil {
			return "", fmt.Errorf("holdout: filesystem: 写入记录 %s: %w", ref, err)
		}
	}
	if fsExists(s.metaPath(ref)) {
		return ref, nil // 记录已完整落成,幂等返回
	}
	// 落成时间取 blob 的落盘时刻:同一记录无论何时补写元数据(例如上次 Put
	// 在 blob 与 meta 之间中断后的修复),CreatedAt 都稳定为记录诞生时刻。
	fi, err := os.Stat(blob)
	if err != nil {
		return "", fmt.Errorf("holdout: filesystem: 记录 %s 落成失败: %w", ref, err)
	}
	meta, err := json.Marshal(fsRecord{CreatedAt: fi.ModTime(), Digest: digest, Size: int64(len(content))})
	if err != nil {
		return "", fmt.Errorf("holdout: filesystem: 编码记录 %s 元数据: %w", ref, err)
	}
	if err := s.writeAtomic(objectKey(ref)+fsMetaSuffix, meta); err != nil {
		return "", fmt.Errorf("holdout: filesystem: 写入记录 %s 元数据: %w", ref, err)
	}
	return ref, nil
}

// Read 实现 VerificationReader:把不透明 ref 解析回内容。这是本实现唯一的
// Ref → 内容路径(AC2)。ref 非法返回 ErrInvalidRef;记录不存在返回
// ErrNotFound。错误信息不回显落盘位置;意外 I/O 错误保留底层诊断细节,
// 该能力按接线只到达验证权威侧。
func (s *FilesystemStore) Read(ctx context.Context, ref Ref) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := ParseRef(ref.String()); err != nil {
		return nil, err
	}
	content, err := os.ReadFile(s.blobPath(ref))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, ref)
		}
		return nil, fmt.Errorf("holdout: filesystem: 读取记录 %s: %w", ref, err)
	}
	return content, nil
}

// Stat 实现 VerificationReader:只读元数据文件,不读取内容(blob 不触碰),
// 供完整性核对与 exposure accounting 使用。元数据损坏返回错误而不是猜测。
func (s *FilesystemStore) Stat(ctx context.Context, ref Ref) (Metadata, error) {
	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}
	if _, err := ParseRef(ref.String()); err != nil {
		return Metadata{}, err
	}
	raw, err := os.ReadFile(s.metaPath(ref))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Metadata{}, fmt.Errorf("%w: %s", ErrNotFound, ref)
		}
		return Metadata{}, fmt.Errorf("holdout: filesystem: 读取记录 %s 元数据: %w", ref, err)
	}
	var rec fsRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return Metadata{}, fmt.Errorf("holdout: filesystem: 记录 %s 元数据损坏: %w", ref, err)
	}
	return Metadata{Ref: ref, Size: rec.Size, Digest: rec.Digest, CreatedAt: rec.CreatedAt}, nil
}

// blobPath / metaPath 是记录的落盘位置:未导出,类型之外不可获得(AC4)。
func (s *FilesystemStore) blobPath(ref Ref) string {
	return filepath.Join(s.objectsDir, objectKey(ref)+fsBlobSuffix)
}

func (s *FilesystemStore) metaPath(ref Ref) string {
	return filepath.Join(s.objectsDir, objectKey(ref)+fsMetaSuffix)
}

// writeAtomic 经 objects 目录内的临时文件 + rename 落盘(同盘 rename 原子),
// 失败时清理临时文件。权限 0600:Tally 拥有的私有内容。
func (s *FilesystemStore) writeAtomic(name string, data []byte) error {
	tmp, err := os.CreateTemp(s.objectsDir, fsTempPattern)
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if tmpName != "" {
			os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, fsFilePerm); err != nil {
		return err
	}
	if err := os.Rename(tmpName, filepath.Join(s.objectsDir, name)); err != nil {
		return err
	}
	tmpName = "" // 已改名,defer 不再清理
	return nil
}

// objectKey 把 ref 映射为固定长度的对象文件名:sha256(ref) 十六进制。
// 键与 ref 拼写完全解耦:即使一个 ref 通过了 ParseRef(如 "...."、"%2e%2e"),
// 它也只能命中 objects 目录内的一个哈希名,永远派生不出路径分隔符、盘符
// 或父目录分量(AC3 的第二道机械防线;第一道是 ParseRef)。
func objectKey(ref Ref) string {
	sum := sha256.Sum256([]byte(ref))
	return hex.EncodeToString(sum[:])
}

// fsResolve 把 path 解析为绝对路径,并对已存在的前缀解析符号链接(Windows
// 上同时得到卷上真实大小写),使 AC1 的内外比较基于真实落盘位置而不是
// 调用方拼写。不存在时向上找到最深的已存在祖先,解析后拼回剩余分量。
func fsResolve(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("holdout: filesystem: 解析绝对路径: %w", err)
	}
	cur := filepath.Clean(abs)
	var missing []string
	for {
		real, err := filepath.EvalSymlinks(cur)
		if err == nil {
			return filepath.Join(append([]string{real}, missing...)...), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("holdout: filesystem: 解析路径: %w", err)
		}
		dir, file := filepath.Split(cur)
		if file == "" {
			// 已到卷根仍不存在:仅剩不可能的理论路径,保守返回当前值。
			return cur, nil
		}
		missing = append([]string{file}, missing...)
		cur = filepath.Clean(dir)
	}
}

// fsInsideOrEqual 报告 child 是否解析为 parent 自身或其内部(含相等)。
// Windows 卷大小写不敏感,比较前做大小写折叠。
func fsInsideOrEqual(parent, child string) bool {
	if runtime.GOOS == "windows" {
		parent = strings.ToLower(parent)
		child = strings.ToLower(child)
	}
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil {
		return false
	}
	if filepath.IsAbs(rel) {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// fsExists 报告路径存在(文件、目录、符号链接)。
func fsExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
