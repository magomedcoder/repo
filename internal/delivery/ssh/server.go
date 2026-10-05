package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/internal/usecase"
	"golang.org/x/crypto/ssh"
)

type Server struct {
	keys    *usecase.SSHKeyUseCase
	git     *usecase.GitUseCase
	addr    string
	hostDir string
}

func NewServer(keys *usecase.SSHKeyUseCase, git *usecase.GitUseCase, hostDir, addr string) *Server {
	if addr == "" {
		addr = ":2222"
	}

	if hostDir == "" {
		hostDir = "data/ssh"
	}

	return &Server{
		keys:    keys,
		git:     git,
		addr:    addr,
		hostDir: hostDir,
	}
}

func (s *Server) ListenAndServe() error {
	return s.ListenAndServeAnnounce(nil)
}

func (s *Server) ListenAndServeAnnounce(addrCh chan<- string) error {
	signer, err := loadOrCreateHostKey(s.hostDir)
	if err != nil {
		return err
	}

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			user, authErr := s.keys.AuthenticateFingerprint(ssh.FingerprintSHA256(key))
			if authErr != nil {
				return nil, authErr
			}
			return &ssh.Permissions{
				Extensions: map[string]string{
					"user_id":  fmt.Sprintf("%d", user.ID),
					"username": user.Username,
				},
			}, nil
		},
		NoClientAuth: false,
	}
	config.AddHostKey(signer)

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	actual := ln.Addr().String()
	log.Printf("ssh listening on %s", actual)
	if addrCh != nil {
		addrCh <- actual
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}

		go s.handleConn(conn, config)
	}
}

func (s *Server) handleConn(nConn net.Conn, config *ssh.ServerConfig) {
	defer nConn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			return
		}

		go s.handleSession(sshConn, channel, requests)
	}
}

func (s *Server) handleSession(conn *ssh.ServerConn, channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	for req := range requests {
		switch req.Type {
		case "exec":
			if len(req.Payload) < 4 {
				_ = req.Reply(false, nil)
				return
			}

			cmdLen := int(req.Payload[0])<<24 | int(req.Payload[1])<<16 | int(req.Payload[2])<<8 | int(req.Payload[3])
			if cmdLen < 0 || 4+cmdLen > len(req.Payload) {
				_ = req.Reply(false, nil)
				return
			}

			command := string(req.Payload[4 : 4+cmdLen])
			_ = req.Reply(true, nil)
			status := s.runGitCommand(conn, channel, command)
			_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{uint32(status)}))
			return
		case "env", "pty-req", "shell":
			_ = req.Reply(req.Type != "shell" && req.Type != "pty-req", nil)
		default:
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
		}
	}
}

func (s *Server) runGitCommand(conn *ssh.ServerConn, channel ssh.Channel, command string) int {
	service, repoPath, err := parseGitSSHCommand(command)
	if err != nil {
		_, _ = io.WriteString(channel.Stderr(), "repo: only git-upload-pack and git-receive-pack are allowed\n")
		return 128
	}

	owner, folderPath, name, err := splitRepoPath(repoPath)
	if err != nil {
		_, _ = io.WriteString(channel.Stderr(), "repo: invalid repository path\n")
		return 128
	}

	repo, err := s.git.Resolve(owner, folderPath, name)
	if err != nil {
		_, _ = io.WriteString(channel.Stderr(), "repo: repository not found\n")
		return 128
	}

	viewer := &domain.User{
		ID:       parseUint(conn.Permissions.Extensions["user_id"]),
		Username: conn.Permissions.Extensions["username"],
	}
	if viewer.ID == 0 {
		_, _ = io.WriteString(channel.Stderr(), "repo: unauthorized\n")
		return 128
	}

	if err := s.git.Authorize(repo, viewer, service); err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			_, _ = io.WriteString(channel.Stderr(), "repo: unauthorized\n")
			return 128
		}
		_, _ = io.WriteString(channel.Stderr(), "repo: permission denied\n")
		return 128
	}

	if err := s.git.ServeSSHPack(repo.Path, service, channel, channel, channel.Stderr()); err != nil {
		_, _ = io.WriteString(channel.Stderr(), "repo: git error\n")
		return 128
	}

	if service == usecase.GitReceivePack {
		_ = s.git.TouchActivity(repo.ID)
	}

	return 0
}

func parseGitSSHCommand(command string) (usecase.GitService, string, error) {
	command = strings.TrimSpace(command)
	fields := strings.Fields(command)
	if len(fields) < 2 {
		return "", "", errors.New("invalid command")
	}

	service, err := usecase.ParseGitService(fields[0])
	if err != nil {
		return "", "", err
	}

	path := strings.Join(fields[1:], " ")
	path = strings.Trim(path, "'\"")
	path = strings.TrimSpace(path)
	if path == "" {
		return "", "", errors.New("missing path")
	}

	return service, path, nil
}

func splitRepoPath(repoPath string) (owner, folderPath, name string, err error) {
	repoPath = strings.TrimSpace(repoPath)
	repoPath = strings.TrimPrefix(repoPath, "/")
	repoPath = strings.TrimSuffix(repoPath, "/")
	if before, ok := strings.CutSuffix(repoPath, ".git"); ok {
		repoPath = before
	}

	parts := strings.Split(repoPath, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}

		if part == ".." {
			return "", "", "", errors.New("invalid path")
		}

		clean = append(clean, part)
	}

	if len(clean) < 2 {
		return "", "", "", errors.New("invalid path")
	}

	owner = clean[0]
	name = clean[len(clean)-1]

	if len(clean) > 2 {
		folderPath = strings.Join(clean[1:len(clean)-1], "/")
	}

	if owner == "" || name == "" || owner == "api" {
		return "", "", "", errors.New("invalid path")
	}

	return owner, folderPath, name, nil
}

func parseUint(raw string) uint {
	var n uint
	for _, c := range raw {
		if c < '0' || c > '9' {
			return 0
		}

		n = n*10 + uint(c-'0')
	}

	return n
}

func loadOrCreateHostKey(dir string) (ssh.Signer, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "ssh_host_ed25519_key")
	data, err := os.ReadFile(path)
	if err == nil {
		signer, parseErr := ssh.ParsePrivateKey(data)
		if parseErr != nil {
			return nil, parseErr
		}

		return signer, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, err
	}

	pemBytes := pem.EncodeToMemory(block)
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return nil, err
	}

	return ssh.NewSignerFromKey(priv)
}
