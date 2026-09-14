package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConfig 描述一台目标机。
type SSHConfig struct {
	Host     string // host 或 host:port;缺省端口 22
	User     string
	Password string // 口令认证(与 KeyFile 二选一)
	KeyFile  string // 私钥文件路径
}

// SSHRunner 通过 SSH 在远端执行命令(x/crypto/ssh)。
//
// 建连与执行委托给 RunRemote,单测注入替身即可离线验证,不必连真机。
type SSHRunner struct {
	Config    SSHConfig
	RunRemote func(ctx context.Context, cfg SSHConfig, command string) (Result, error)
	// HostKeyCallback 主机密钥校验;空则 InsecureIgnoreHostKey。
	// 单用户自部署的内网场景可接受不校验;要严格校验由调用方注入。
	HostKeyCallback ssh.HostKeyCallback
}

// NewSSHRunner 返回连往 cfg 的 SSH 执行器。
func NewSSHRunner(cfg SSHConfig) *SSHRunner {
	return &SSHRunner{Config: cfg, RunRemote: sshRunRemote}
}

// Run 实现 Runner:把 cmd.Argv 拼成一条远端 shell 命令执行。
func (r *SSHRunner) Run(ctx context.Context, cmd Command) (Result, error) {
	run := r.RunRemote
	if run == nil {
		run = sshRunRemote
	}
	return run(ctx, r.Config, strings.Join(cmd.Argv, " "))
}

// sshRunRemote 用 x/crypto/ssh 建连执行一条命令,分离 stdout/stderr 并取退出码。
func sshRunRemote(ctx context.Context, cfg SSHConfig, command string) (Result, error) {
	client, err := dialSSH(ctx, cfg)
	if err != nil {
		return Result{}, err
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return Result{}, fmt.Errorf("开 SSH 会话: %w", err)
	}
	defer sess.Close()

	var stdout, stderr bytes.Buffer
	sess.Stdout = &stdout
	sess.Stderr = &stderr

	start := time.Now()
	runErr := sess.Run(command)
	res := Result{Stdout: stdout.String(), Stderr: stderr.String(), Duration: time.Since(start)}
	if runErr != nil {
		var ee *ssh.ExitError
		if errors.As(runErr, &ee) {
			res.ExitCode = ee.ExitStatus() // 跑完了,退出码非零不是 error
			return res, nil
		}
		return res, fmt.Errorf("SSH 执行: %w", runErr) // 会话/网络层错误
	}
	return res, nil
}

// dialSSH 建立 SSH 客户端连接。ctx 只用于拨号阶段(ssh 握手本身不吃 ctx)。
func dialSSH(ctx context.Context, cfg SSHConfig) (*ssh.Client, error) {
	auth, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}
	host := cfg.Host
	if !strings.Contains(host, ":") {
		host += ":22"
	}
	clientCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // 单用户自部署内网,见 SSHRunner.HostKeyCallback
		Timeout:         15 * time.Second,
	}
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", host)
	if err != nil {
		return nil, fmt.Errorf("拨号 %s: %w", host, err)
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, host, clientCfg)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("SSH 握手 %s: %w", host, err)
	}
	return ssh.NewClient(c, chans, reqs), nil
}

// authMethods 按配置产出认证方式:私钥优先,其次口令。
func authMethods(cfg SSHConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	if cfg.KeyFile != "" {
		pem, err := os.ReadFile(cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("读私钥 %s: %w", cfg.KeyFile, err)
		}
		signer, err := ssh.ParsePrivateKey(pem)
		if err != nil {
			return nil, fmt.Errorf("解析私钥 %s: %w", cfg.KeyFile, err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if cfg.Password != "" {
		methods = append(methods, ssh.Password(cfg.Password))
	}
	if len(methods) == 0 {
		return nil, errors.New("SSH 配置缺少认证方式(KeyFile 或 Password)")
	}
	return methods, nil
}
