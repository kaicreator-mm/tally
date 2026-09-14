// Package core 定义跨包共享的领域类型。
//
// 干什么:提供 ProjectID、Fingerprint、ArtifactDigest、ClauseHash、
// VersionCombo 等具名类型,避免裸 string 互相串用。
//
// 不干什么:不放业务逻辑。只被一个包使用的类型放那个包里。
//
// 允许导入:标准库。不导入任何 internal 包(由 test/arch 强制)。
package core
