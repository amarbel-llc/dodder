package blob_stores

import (
	"fmt"
	"net"
	"os"

	"code.linenisgreat.com/dodder/go/internal/hotel/blob_store_configs"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func MakeSSHAgent(
	ctx interfaces.ActiveContext,
	uiPrinter ui.Printer,
) (sshAgent agent.ExtendedAgent, err error) {
	socket := os.Getenv("SSH_AUTH_SOCK")

	if socket == "" {
		err = errors.Errorf("SSH_AUTH_SOCK empty or unset")
		return
	}

	var connSshSock net.Conn

	ui.Log().Print("connecting to SSH_AUTH_SOCK: %s", socket)
	if connSshSock, err = net.Dial("unix", socket); err != nil {
		err = errors.Wrapf(err, "failed to connect to SSH_AUTH_SOCK")
		return
	}

	ctx.After(errors.MakeFuncContextFromFuncErr(connSshSock.Close))

	ui.Log().Print("creating ssh-agent client")
	sshAgent = agent.NewClient(connSshSock)

	return
}

// TODO refactor `blob_store_configs.ConfigSFTP` for ssh-client-specific methods
func MakeSSHClientForExplicitConfig(
	ctx interfaces.ActiveContext,
	uiPrinter ui.Printer,
	config blob_store_configs.ConfigSFTPConfigExplicit,
) (sshClient *ssh.Client, err error) {
	clientConfig := &ssh.ClientConfig{
		User:            config.GetUser(),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: make this configurable
	}

	// Configure authentication
	if config.GetPrivateKeyPath() != "" {
		var key ssh.Signer
		var keyBytes []byte

		if keyBytes, err = os.ReadFile(config.GetPrivateKeyPath()); err != nil {
			err = errors.Wrapf(err, "failed to read private key")
			return sshClient, err
		}

		if key, err = ssh.ParsePrivateKey(keyBytes); err != nil {
			err = errors.Wrapf(err, "failed to parse private key")
			return sshClient, err
		}

		clientConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(key)}
	} else if config.GetPassword() != "" {
		clientConfig.Auth = []ssh.AuthMethod{ssh.Password(config.GetPassword())}
	} else {
		var clientAgent agent.ExtendedAgent

		if clientAgent, err = MakeSSHAgent(ctx, uiPrinter); err != nil {
			err = errors.Wrap(err)
			return
		}

		clientConfig.Auth = []ssh.AuthMethod{
			ssh.PublicKeysCallback(clientAgent.Signers),
		}
	}

	addr := fmt.Sprintf(
		"%s:%d",
		config.GetHost(),
		config.GetPort(),
	)

	if sshClient, err = sshDial(ctx, uiPrinter, clientConfig, addr); err != nil {
		err = errors.Wrap(err)
		return sshClient, err
	}

	return sshClient, err
}

func MakeSSHClientFromSSHConfig(
	ctx interfaces.ActiveContext,
	uiPrinter ui.Printer,
	config blob_store_configs.ConfigSFTPUri,
) (sshClient *ssh.Client, err error) {
	var clientAgent agent.ExtendedAgent

	if clientAgent, err = MakeSSHAgent(ctx, uiPrinter); err != nil {
		err = errors.Wrap(err)
		return
	}

	uri := config.GetUri()
	url := uri.GetUrl()

	clientConfig := &ssh.ClientConfig{
		User: url.User.Username(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(clientAgent.Signers),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO make configurable
	}

	port := url.Port()

	if port == "" {
		port = "22"
	}

	addr := fmt.Sprintf("%s:%s", url.Hostname(), port)

	if sshClient, err = sshDial(ctx, uiPrinter, clientConfig, addr); err != nil {
		err = errors.Wrap(err)
		return sshClient, err
	}

	return sshClient, err
}

func sshDial(
	ctx interfaces.ActiveContext,
	uiPrinter ui.Printer,
	configClient *ssh.ClientConfig,
	addr string,
) (sshClient *ssh.Client, err error) {
	uiPrinter.Printf("dialing %q...", addr)
	if sshClient, err = ssh.Dial("tcp", addr, configClient); err != nil {
		err = errors.Wrapf(err, "failed to connect to SSH server")
		return sshClient, err
	}
	uiPrinter.Printf("connected to %q...", addr)

	ctx.After(errors.MakeFuncContextFromFuncErr(sshClient.Close))

	return sshClient, err
}
