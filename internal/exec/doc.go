// Package exec 是动作 job 的执行端:把界面入队的环境 / 部署动作接到 environment / deploy 包,
// 真正执行并写事实面;失败再经 ingest 录进 failure_event,诊断 job 接到 failure 包。
//
// 这是 INV-5 的正路所在:事实面(environment_run / deploy_run / failure_event)只由「系统执行动作」
// 时 INSERT —— 界面 POST 只往 job 表排队(internal/web/handlers 的 enqueue),本包在 worker 进程里
// 消费这些 job,调 environment.Manager / deploy.Deployer 执行,由它们写新记录;结构化失败错误
// (environment.UpError / deploy.FailureError)在这里被 ingest 录成 failure_event。界面侧没有任何
// 事实面写入。
//
// 依赖方向与 web/jobs 的编排角色一致:exec → environment / deploy / runner 的执行能力 + jobs 的
// 任务契约(kind 与 payload)+ ingest / failure 的失败录入与诊断。业务包(environment / deploy)
// 不导入 ingest:它们只返回带 runID 与指纹的结构化错误,录入由本编排层统一做。诊断经 failure 包
// 间接用到通用推理(走 internal/airuntime 边界,P0-11),但推理只提案、四道闸判定在 failure 里(INV-1)。
package exec
