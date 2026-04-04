package haustoria_orgmode

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SFTPConfig holds SFTP connection parameters for the orgmode transport.
type SFTPConfig struct {
	Host           string
	Port           int
	User           string
	Password       string
	PrivateKeyPath string
	KnownHostsFile string
}

// sftpTransport implements Transport using SFTP.
type sftpTransport struct {
	config     SFTPConfig
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

var _ Transport = &sftpTransport{}

// MakeSFTPTransport creates a Transport backed by SFTP.
func MakeSFTPTransport(config SFTPConfig) (Transport, error) {
	transport := &sftpTransport{config: config}

	if err := transport.connect(); err != nil {
		return nil, fmt.Errorf("sftp transport connect: %w", err)
	}

	return transport, nil
}

func (transport *sftpTransport) connect() (err error) {
	var authMethods []ssh.AuthMethod

	if transport.config.Password != "" {
		authMethods = append(authMethods, ssh.Password(transport.config.Password))
	}

	if transport.config.PrivateKeyPath != "" {
		keyBytes, readErr := os.ReadFile(transport.config.PrivateKeyPath)
		if readErr != nil {
			return fmt.Errorf("read private key %s: %w", transport.config.PrivateKeyPath, readErr)
		}

		signer, parseErr := ssh.ParsePrivateKey(keyBytes)
		if parseErr != nil {
			return fmt.Errorf("parse private key: %w", parseErr)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if len(authMethods) == 0 {
		// Try SSH agent via SSH_AUTH_SOCK.
		if agentAuth := sshAgentAuth(); agentAuth != nil {
			authMethods = append(authMethods, agentAuth)
		}
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("no SSH authentication method available")
	}

	port := transport.config.Port
	if port == 0 {
		port = 22
	}

	sshConfig := &ssh.ClientConfig{
		User:            transport.config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", transport.config.Host, port)

	transport.sshClient, err = ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("ssh dial %s: %w", addr, err)
	}

	transport.sftpClient, err = sftp.NewClient(transport.sshClient)
	if err != nil {
		transport.sshClient.Close()
		return fmt.Errorf("sftp client: %w", err)
	}

	return nil
}

func (transport *sftpTransport) List(folder string) (files []RemoteFile, err error) {
	entries, err := transport.sftpClient.ReadDir(folder)
	if err != nil {
		return nil, fmt.Errorf("sftp readdir %s: %w", folder, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".org") {
			continue
		}

		files = append(files, RemoteFile{
			Path: filepath.Join(folder, name),
			Name: name,
			Size: entry.Size(),
			// SFTP has no native ETag; use mtime as a change marker.
			ETag: fmt.Sprintf("%d", entry.ModTime().UnixNano()),
		})
	}

	return files, nil
}

func (transport *sftpTransport) Read(filePath string) (content []byte, etag string, err error) {
	file, err := transport.sftpClient.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("sftp open %s: %w", filePath, err)
	}
	defer file.Close()

	content, err = io.ReadAll(file)
	if err != nil {
		return nil, "", fmt.Errorf("sftp read %s: %w", filePath, err)
	}

	// Use stat mtime as ETag equivalent.
	info, statErr := transport.sftpClient.Stat(filePath)
	if statErr == nil {
		etag = fmt.Sprintf("%d", info.ModTime().UnixNano())
	}

	return content, etag, nil
}

func (transport *sftpTransport) Write(filePath string, content []byte, _ string) (err error) {
	// Ensure parent directory exists.
	dir := filepath.Dir(filePath)
	if err = transport.sftpClient.MkdirAll(dir); err != nil {
		return fmt.Errorf("sftp mkdir %s: %w", dir, err)
	}

	file, err := transport.sftpClient.Create(filePath)
	if err != nil {
		return fmt.Errorf("sftp create %s: %w", filePath, err)
	}
	defer file.Close()

	if _, err = file.Write(content); err != nil {
		return fmt.Errorf("sftp write %s: %w", filePath, err)
	}

	return nil
}

func (transport *sftpTransport) Delete(filePath string) (err error) {
	if err = transport.sftpClient.Remove(filePath); err != nil {
		return fmt.Errorf("sftp remove %s: %w", filePath, err)
	}

	return nil
}

// sshAgentAuth attempts to create an SSH agent authentication method from
// SSH_AUTH_SOCK. Returns nil if unavailable.
func sshAgentAuth() ssh.AuthMethod {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil
	}

	return ssh.PublicKeysCallback(agent.NewClient(conn).Signers)
}
