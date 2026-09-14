// Package environment 拉起和维护开发/测试环境。
//
// 干什么:解析环境定义、调 runner 拉起、跑健康检查、记录版本组合、推倒重建。
//
// 不干什么:不碰生产;不部署应用制品(那是 deploy);不做 k8s。
//
// 硬校验:没有健康检查的环境定义拒绝注册,返回 ErrNoHealthCheck。
//
// 允许导入:core、runner、store、logpipe。
package environment
