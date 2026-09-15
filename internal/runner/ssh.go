package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHConfig describes a remote target. Host may include an explicit port.
// Password and private key material are transport-only secrets and must never
// be copied into provenance/log output.
type SSHConfig struct {
	Host                string
	User                string
	Password            string
	KeyFile             string
	KnownHostsFile      string
	InsecureSkipHostKey bool
}

// SSHRunner executes Command in a remote POSIX shell. Dir, Env and Argv are
// preserved at the transport boundary; cancellation closes the remote session.
type SSHRunner struct {
	Config          SSHConfig
	RunRemote       func(context.Context, SSHConfig, string) (Result, error)
	HostKeyCallback ssh.HostKeyCallback
}

func NewSSHRunner(cfg SSHConfig) *SSHRunner { return &SSHRunner{Config: cfg} }

var (
	envNameRE   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	shellSafeRE = regexp.MustCompile(`^[A-Za-z0-9_./:@%+=,-]+$`)
)

func (r *SSHRunner) Run(ctx context.Context, cmd Command) (Result, error) {
	remote, err := remoteShellCommand(cmd)
	if err != nil {
		return Result{}, err
	}
	if r.RunRemote != nil {
		return r.RunRemote(ctx, r.Config, remote)
	}
	return sshRunRemoteWithHostKey(ctx, r.Config, remote, r.HostKeyCallback)
}

func remoteShellCommand(cmd Command) (string, error) {
	if len(cmd.Argv) == 0 {
		return "", errors.New("runner: 空命令")
	}
	argv := make([]string, len(cmd.Argv))
	for i, arg := range cmd.Argv {
		argv[i] = shellQuote(arg)
	}
	parts := make([]string, 0, 4)
	if cmd.Dir != "" {
		parts = append(parts, "cd -- "+shellQuote(cmd.Dir)+" &&")
	}
	parts = append(parts, "exec")
	if len(cmd.Env) > 0 {
		parts = append(parts, "env")
		for _, entry := range cmd.Env {
			key, _, ok := strings.Cut(entry, "=")
			if !ok || !envNameRE.MatchString(key) {
				return "", fmt.Errorf("runner: 无效环境变量 %q", entry)
			}
			parts = append(parts, shellQuote(entry))
		}
	}
	parts = append(parts, argv...)
	return strings.Join(parts, " "), nil
}

func shellQuote(s string) string {
	if s != "" && shellSafeRE.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func sshRunRemote(ctx context.Context, cfg SSHConfig, command string) (Result, error) {
	return sshRunRemoteWithHostKey(ctx, cfg, command, nil)
}

func sshRunRemoteWithHostKey(ctx context.Context, cfg SSHConfig, command string, callback ssh.HostKeyCallback) (Result, error) {
	client, err := dialSSHWithHostKey(ctx, cfg, callback)
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
	if err := sess.Start(command); err != nil {
		return Result{}, fmt.Errorf("SSH 启动: %w", err)
	}
	waitCh := make(chan error, 1)
	go func() { waitCh <- sess.Wait() }()

	var runErr error
	select {
	case runErr = <-waitCh:
	case <-ctx.Done():
		_ = sess.Signal(ssh.SIGTERM)
		_ = sess.Close()
		select {
		case <-waitCh:
		case <-time.After(time.Second):
		}
		return Result{Stdout: stdout.String(), Stderr: stderr.String(), Duration: time.Since(start)}, fmt.Errorf("SSH 执行: %w", ctx.Err())
	}

	res := Result{Stdout: stdout.String(), Stderr: stderr.String(), Duration: time.Since(start)}
	if runErr != nil {
		var ee *ssh.ExitError
		if errors.As(runErr, &ee) {
			res.ExitCode = ee.ExitStatus()
			return res, nil
		}
		return res, fmt.Errorf("SSH 执行: %w", runErr)
	}
	return res, nil
}

func dialSSH(ctx context.Context, cfg SSHConfig) (*ssh.Client, error) {
	return dialSSHWithHostKey(ctx, cfg, nil)
}

func dialSSHWithHostKey(ctx context.Context, cfg SSHConfig, callback ssh.HostKeyCallback) (*ssh.Client, error) {
	if strings.TrimSpace(cfg.Host) == "" || strings.TrimSpace(cfg.User) == "" {
		return nil, errors.New("SSH 配置缺少 Host 或 User")
	}
	auth, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}
	callback, err = resolveHostKeyCallback(cfg, callback)
	if err != nil {
		return nil, err
	}
	host := cfg.Host
	if !strings.Contains(host, ":") {
		host += ":22"
	}
	clientCfg := &ssh.ClientConfig{User: cfg.User, Auth: auth, HostKeyCallback: callback, Timeout: 15 * time.Second}
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

func resolveHostKeyCallback(cfg SSHConfig, explicit ssh.HostKeyCallback) (ssh.HostKeyCallback, error) {
	if explicit != nil {
		return explicit, nil
	}
	knownHostsFile := strings.TrimSpace(cfg.KnownHostsFile)
	if knownHostsFile == "" {
		if home, err := os.UserHomeDir(); err == nil {
			candidate := filepath.Join(home, ".ssh", "known_hosts")
			if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
				knownHostsFile = candidate
			}
		}
	}
	if knownHostsFile != "" {
		callback, err := knownhosts.New(knownHostsFile)
		if err != nil {
			return nil, fmt.Errorf("读取 SSH known_hosts %s: %w", knownHostsFile, err)
		}
		return callback, nil
	}
	if cfg.InsecureSkipHostKey {
		return ssh.InsecureIgnoreHostKey(), nil // explicit opt-in only
	}
	return nil, errors.New("SSH host-key policy 未配置: 提供 KnownHostsFile/HostKeyCallback，或显式设置 InsecureSkipHostKey")
}

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
