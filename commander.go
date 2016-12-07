package commander

import (
	"bytes"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Connection struct {
	Host       string
	Port       int
	User       string
	Passwd     string
	Protocol   string
	Client     *ssh.Client
	SftpClient *sftp.Client
	Session    *ssh.Session
}

type PasswdConnection struct {
	Connection Connection
}

type KeyConnection struct {
	Connection Connection
	PemPath    string
	Signer     ssh.Signer
}

func (c *KeyConnection) Connect() ([]string, error) {
	var logs []string
	//Parsing SSH Private Key.
	signer, err := getPrivateKey(c.PemPath)
	if err != nil {
		logs = append(logs, "Loading SSH Private Key File\t[ NG ]")
		return logs, err
	}
	logs = append(logs, "Loading SSH Private Key File\t[ OK ]")
	c.Signer = signer

	//Generate SSH Configuration.
	config := &ssh.ClientConfig{
		User: c.Connection.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(c.Signer),
		},
	}
	logs = append(logs, "Generate SSH Configuration\t[ OK ]")

	//Connection.
	client, err := connect(config, c.Connection.Host,
		c.Connection.Port, c.Connection.Protocol)
	if err != nil {
		logs = append(logs, "SSH Connecting...\t[ NG ]")
		return logs, err
	}
	logs = append(logs, "SSH Connecting...\t[ OK ]")
	c.Connection.Client = client

	//Sftp Client.
	sftp, err := sftp.NewClient(client)
	if err != nil {
		logs = append(logs, "SFTP Connecting...\t[ NG ]")
		return logs, err
	}
	logs = append(logs, "SFTP Connecting...\t[ OK ]")
	c.Connection.SftpClient = sftp

	//Generate New Session.
	session, err := c.Connection.Client.NewSession()
	if err != nil {
		logs = append(logs, "Generate New Session\t[ NG ]")
		return logs, err
	}
	logs = append(logs, "Generate New Session\t[ OK ]")
	c.Connection.Session = session

	return logs, nil
}

func (c *KeyConnection) Command(command string) ([]string, string) {
	var logs []string
	var b bytes.Buffer
	logs = append(logs, fmt.Sprintf("Executing Command: %s", command))
	c.Connection.Session.Stdout = &b
	err := c.Connection.Session.Run(command)
	if err != nil {
		logs = append(logs, "Failed to executing command.")
	} else {
		logs = append(logs, "Complated Command!")
	}
	return logs, b.String()
}

func (c *KeyConnection) Download(fromDir string, destDir string, fname string) error {
	src, err := c.Connection.SftpClient.Open(filepath.Join(fromDir, fname))
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.Create(filepath.Join(destDir, fname))
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = src.WriteTo(dest)
	if err != nil {
		return err
	}
	return nil
}

func (c *KeyConnection) Dispose() error {
	if err := c.Connection.Client.Close(); err != nil {
		return err
	}
	return nil
}

func getPrivateKey(pemPath string) (ssh.Signer, error) {
	buf, err := ioutil.ReadFile(pemPath)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(buf)
}

func connect(config *ssh.ClientConfig, host string,
	port int, protocol string) (*ssh.Client, error) {
	return ssh.Dial(protocol,
		fmt.Sprintf("%s:%d", host, port), config)
}
