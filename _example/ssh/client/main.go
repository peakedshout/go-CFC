package main

import (
	"github.com/peakedshout/go-CFC/client"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"os"
)

func main() {
	user := "test"
	password := "123456"
	c, err := client.LinkLongConn("sshClient", "127.0.0.1", ":9999", "6a647c0bf889419c84e461486f83d776")
	if err != nil {
		panic(err)
	}
	defer c.Close()
	sub, err := c.GetSubConn("sshServer")
	if err != nil {
		panic(err)
	}
	defer sub.Close()
	conn, _, _, err := ssh.NewClientConn(sub.GetConn(), sub.GetConn().RemoteAddr().String(), &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	clien := ssh.Client{Conn: conn}
	session, err := clien.NewSession()
	if err != nil {
		panic(err)
	}
	defer session.Close()

	if err != nil {
		panic(err)
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.ECHOCTL:       0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	fileDescriptor := int(os.Stdin.Fd())

	if terminal.IsTerminal(fileDescriptor) {
		originalState, err := terminal.MakeRaw(fileDescriptor)
		if err != nil {
			panic(err)
		}
		defer terminal.Restore(fileDescriptor, originalState)
		termWidth, termHeight, err := terminal.GetSize(fileDescriptor)
		if err != nil {
			panic(err)
		}
		err = session.RequestPty("xterm-256color", termHeight, termWidth, modes)
		if err != nil {
			panic(err)
		}
	}

	session.Shell()
	session.Wait()
	return
}
