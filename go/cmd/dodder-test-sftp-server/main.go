package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func main() {
	root := flag.String("root", ".", "Root directory for SFTP file serving")
	flag.Parse()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate host key: %v", err)
	}

	hostSigner, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		log.Fatalf("failed to create signer: %v", err)
	}

	config := &ssh.ServerConfig{
		PasswordCallback: func(
			conn ssh.ConnMetadata,
			password []byte,
		) (*ssh.Permissions, error) {
			return nil, nil
		},
	}

	config.AddHostKey(hostSigner)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	addr := listener.Addr().(*net.TCPAddr)
	pubKeyBytes := hostSigner.PublicKey().Marshal()
	pubKeyType := hostSigner.PublicKey().Type()

	fmt.Printf(
		"READY port=%d known_hosts=[127.0.0.1]:%d %s %s\n",
		addr.Port,
		addr.Port,
		pubKeyType,
		base64.StdEncoding.EncodeToString(pubKeyBytes),
	)

	os.Stdout.Sync()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigCh
		listener.Close()
		os.Exit(0)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		go handleConnection(conn, config, *root)
	}
}

func handleConnection(conn net.Conn, config *ssh.ServerConfig, root string) {
	defer conn.Close() //defer:err-checked

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("ssh handshake failed: %v", err)
		return
	}

	defer sshConn.Close() //defer:err-checked

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go func() {
			for req := range requests {
				if req.Type == "subsystem" &&
					len(req.Payload) >= 4 &&
					string(req.Payload[4:]) == "sftp" {
					req.Reply(true, nil)

					server, err := sftp.NewServer(channel,
						sftp.WithServerWorkingDirectory(root),
					)
					if err != nil {
						log.Printf("sftp server init failed: %v", err)
						return
					}

					if err := server.Serve(); err != nil && err != io.EOF {
						log.Printf("sftp server error: %v", err)
					}

					channel.Close()
					return
				}

				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}()
	}
}
