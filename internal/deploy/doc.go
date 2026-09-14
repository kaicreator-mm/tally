// Package deploy 把已构建的制品部到环境上。
//
// 干什么:三件事 —— 决定部什么、调外部工具执行、把结果记下来。
//
// 不干什么:不实现零停机切换、健康等待、滚动更新、回滚。
// 那些交给 Kamal / Ansible / Compose。本包超过 500 行就说明写错了方向。
//
// 硬校验:输入必须是已存在的制品 digest,不接受"拉代码后构建"。
//
// 允许导入:core、runner、store、logpipe。
package deploy
