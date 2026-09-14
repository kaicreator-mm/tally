package deploy

import "context"

// Rollback 调工具自带的回滚:把 def.Digest(通常是上一个能跑的制品 digest)部回去。
//
// 回滚逻辑在工具里,本包只转发(kamal rollback / ansible --tags rollback /
// compose 再部一次目标 digest),读退出码,记一行 deploy_run:
// 成功 rolled_back,失败 rollback_failed(失败输出同样送 logpipe,source=deploy)。
// 不等不重试,不实现回滚策略(铁律 6)。
func (d *Deployer) Rollback(ctx context.Context, def Definition) (Result, error) {
	return d.run(ctx, def, true)
}
