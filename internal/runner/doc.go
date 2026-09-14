// Package runner 提供命令执行抽象。
//
// 干什么:DockerRunner(shell 调 docker CLI)与 SSHRunner,
// 返回 stdout / stderr / exitCode / duration 的原始结果。
//
// 不干什么:不解释结果。判断成功失败是调用方的事。
//
// 允许导入:core、标准库、x/crypto/ssh。
package runner
