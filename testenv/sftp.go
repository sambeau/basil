package testenv

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	gliderssh "github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
)

const (
	sftpUser     = "testuser"
	sftpPassword = "testpass"
)

// osHandler implements sftp.Handlers backed by a real directory on disk.
// All paths are resolved relative to root.
type osHandler struct {
	root string
}

func (h *osHandler) abs(p string) string {
	return filepath.Join(h.root, filepath.FromSlash(p))
}

func (h *osHandler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	f, err := os.OpenFile(h.abs(r.Filepath), os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (h *osHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	abs := h.abs(r.Filepath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (h *osHandler) Filecmd(r *sftp.Request) error {
	switch r.Method {
	case "Mkdir":
		return os.MkdirAll(h.abs(r.Filepath), 0o755)
	case "Rmdir":
		return os.Remove(h.abs(r.Filepath))
	case "Remove":
		return os.Remove(h.abs(r.Filepath))
	case "Rename":
		return os.Rename(h.abs(r.Filepath), h.abs(r.Target))
	case "Symlink":
		return os.Symlink(h.abs(r.Target), h.abs(r.Filepath))
	case "Setstat":
		return nil // ignore chmod/chown for tests
	}
	return nil
}

type listerAt []os.FileInfo

func (l listerAt) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}
	n := copy(ls, l[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}

func (h *osHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	switch r.Method {
	case "List":
		entries, err := os.ReadDir(h.abs(r.Filepath))
		if err != nil {
			return nil, err
		}
		infos := make([]os.FileInfo, 0, len(entries))
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			infos = append(infos, info)
		}
		return listerAt(infos), nil

	case "Stat":
		fi, err := os.Stat(h.abs(r.Filepath))
		if err != nil {
			return nil, err
		}
		return listerAt([]os.FileInfo{fi}), nil

	case "Lstat":
		fi, err := os.Lstat(h.abs(r.Filepath))
		if err != nil {
			return nil, err
		}
		return listerAt([]os.FileInfo{fi}), nil

	case "Readlink":
		target, err := os.Readlink(h.abs(r.Filepath))
		if err != nil {
			return nil, err
		}
		// Return a synthetic FileInfo for the link target name.
		fi, err := os.Lstat(target)
		if err != nil {
			return nil, err
		}
		return listerAt([]os.FileInfo{fi}), nil
	}
	return nil, sftp.ErrSSHFxOpUnsupported
}

// startSFTP starts a fake SSH/SFTP server backed by a temp directory.
// It populates env.SFTPAddr, env.SFTPUser, env.SFTPPassword, and env.sftpRoot.
func startSFTP(t testing.TB, env *Env) {
	t.Helper()

	root := t.TempDir()
	env.sftpRoot = root
	env.SFTPUser = sftpUser
	env.SFTPPassword = sftpPassword

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("testenv: failed to listen for SFTP: %v", err)
	}
	env.SFTPAddr = ln.Addr().String()

	handler := &osHandler{root: root}
	handlers := sftp.Handlers{
		FileGet:  handler,
		FilePut:  handler,
		FileCmd:  handler,
		FileList: handler,
	}

	srv := &gliderssh.Server{
		PasswordHandler: func(ctx gliderssh.Context, password string) bool {
			return ctx.User() == sftpUser && password == sftpPassword
		},
		SubsystemHandlers: map[string]gliderssh.SubsystemHandler{
			"sftp": func(sess gliderssh.Session) {
				rs := sftp.NewRequestServer(sess, handlers)
				_ = rs.Serve()
				_ = rs.Close()
			},
		},
		// When HostSigners is empty, gliderlabs generates a host key automatically.
	}

	go func() {
		_ = srv.Serve(ln)
	}()

	t.Cleanup(func() {
		_ = srv.Close()
		_ = ln.Close()
	})
}

// SFTPWriteFile writes content to path (relative to the SFTP root) on the
// fake server's local filesystem, creating parent directories as needed.
// Use this to seed files before tests run.
func (e *Env) SFTPWriteFile(path, content string) {
	fullPath := filepath.Join(e.sftpRoot, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		panic("testenv SFTPWriteFile: MkdirAll: " + err.Error())
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		panic("testenv SFTPWriteFile: WriteFile: " + err.Error())
	}
}

// SFTPReadFile reads and returns the contents of path (relative to the SFTP
// root) from the fake server's local filesystem.
// Use this to assert on files written by tests.
func (e *Env) SFTPReadFile(path string) string {
	fullPath := filepath.Join(e.sftpRoot, filepath.FromSlash(path))
	data, err := os.ReadFile(fullPath)
	if err != nil {
		panic("testenv SFTPReadFile: " + err.Error())
	}
	return string(data)
}

// sftpConn holds both the SFTP client and the underlying SSH connection.
// Both must be closed to avoid resource leaks.
type sftpConn struct {
	SFTP *sftp.Client
	SSH  *gossh.Client
}

func (c *sftpConn) Close() {
	_ = c.SFTP.Close()
	_ = c.SSH.Close()
}

// dialSFTP dials the fake server with the given credentials and returns an
// sftpConn. The caller must call Close() when done.
func dialSFTP(addr, user, password string) (*sftpConn, error) {
	config := &gossh.ClientConfig{
		User: user,
		Auth: []gossh.AuthMethod{
			gossh.Password(password),
		},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}

	sshClient, err := gossh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, err
	}

	return &sftpConn{SFTP: sftpClient, SSH: sshClient}, nil
}
