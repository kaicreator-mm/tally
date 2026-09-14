# npm / native-module-abi —— 原生模块 ABI 不匹配(碰撞测试用)

bcrypt 的原生 `.node` 模块是按旧版 Node(NODE_MODULE_VERSION 108)编译的,
当前运行的是新版(NODE_MODULE_VERSION 115),加载即崩。

三份日志 `run-1/2/3.log` 是**同一个错误**在不同时间、不同机器(pid)、不同构建路径
(/home/alice、/home/bob、/tmp/build-xk29d)下的记录。它们必须算出**同一个** `fp_exact`
(见 `expected-fp.txt`)——这是整套测试里最重要的一组:它们不撞,知识库就永远查不到
同一个修复,而且不会报错,只表现为「这些错误好像各不相同」。
