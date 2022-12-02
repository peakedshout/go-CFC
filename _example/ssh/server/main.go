package main

import (
	"encoding/binary"
	"fmt"
	"github.com/creack/pty"
	"github.com/peakedshout/go-CFC/client"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"
)

func main() {
	s := sshContext{
		sshServerConfig: nil,
		userName:        "test",
		userPassword:    "123456",
	}
	s.sshServerInit()
	c, err := client.LinkLongConn("sshServer", "127.0.0.1", ":9999", "6a647c0bf889419c84e461486f83d776")
	if err != nil {
		panic(err)
	}
	defer c.Close()
	c.ListenSubConn(func(sub *client.SubConnContext) {
		defer sub.Close()
		s.sshOpen(sub.GetConn())
	})
}

type sshContext struct {
	sshServerConfig *ssh.ServerConfig
	userName        string
	userPassword    string
}

func (s *sshContext) sshServerInit() {
	authorizedKeysBytes, privateBytes := create2Key()
	authorizedKeysMap := map[string]bool{}
	for len(authorizedKeysBytes) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			log.Fatal(err)
		}

		authorizedKeysMap[string(pubKey.Marshal())] = true
		authorizedKeysBytes = rest
	}
	signer, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		panic(err)
	}
	s.sshServerConfig = &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if conn.User() == s.userName && string(password) == s.userPassword {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", conn.User())
		},
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if authorizedKeysMap[string(pubKey.Marshal())] {
				return &ssh.Permissions{
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}
	s.sshServerConfig.AddHostKey(signer)
}

func (s *sshContext) sshOpen(c net.Conn) {
	conn, chans, reqs, err := ssh.NewServerConn(c, s.sshServerConfig)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	go ssh.DiscardRequests(reqs)

	var wait sync.WaitGroup
	wait.Add(1)
	newChannel := <-chans
	go func(newChannel ssh.NewChannel) {
		if t := newChannel.ChannelType(); t != "session" {
			newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
			return
		}
		connection, requests, err := newChannel.Accept()
		if err != nil {
			log.Println("Could not accept channel ", err)
			return
		}
		bash := exec.Command("bash")
		bash.Dir = "/home"
		close := func() {
			connection.Close()
			_, err := bash.Process.Wait()
			if err != nil {
				log.Println("Failed to exit bash ", err)
			}
			log.Println("Session closed \n")
			wait.Done()
		}
		log.Println("Creating pty...")
		bashf, err := pty.Start(bash)
		if err != nil {
			log.Println("Could not start pty ", err)
			close()
			return
		}
		var once sync.Once
		go func() {
			io.Copy(connection, bashf)
			once.Do(close)
		}()
		go func() {
			io.Copy(bashf, connection)
			once.Do(close)
		}()
		go func() {
			for req := range requests {
				switch req.Type {
				case "shell":
					if len(req.Payload) == 0 {
						req.Reply(true, nil)
					}
				case "pty-req":
					termLen := req.Payload[3]
					w, h := parseDims(req.Payload[termLen+4:])
					SetWinsize(bashf.Fd(), w, h)
					req.Reply(true, nil)
				case "window-change":
					w, h := parseDims(req.Payload)
					SetWinsize(bashf.Fd(), w, h)
				}
			}
		}()
	}(newChannel)
	wait.Wait()
	return
}

func parseDims(b []byte) (uint32, uint32) {
	w := binary.BigEndian.Uint32(b)
	h := binary.BigEndian.Uint32(b[4:])
	return w, h
}

type Winsize struct {
	Height uint16
	Width  uint16
	x      uint16 // unused
	y      uint16 // unused
}

func SetWinsize(fd uintptr, w, h uint32) {
	ws := &Winsize{Width: uint16(w), Height: uint16(h)}
	syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
}
func create2Key() (authorizedKeysBytes, privateBytes []byte) {
	keyPath := filepath.Join(os.TempDir(), "fssh.rsa")
	//如果key 不存在则 执行 ssh-keygen 创建
	_, err := os.Stat(keyPath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(keyPath), os.ModePerm)
		//执行 ssh-keygen 创建 key
		stderr, err := exec.Command("ssh-keygen", "-f", keyPath, "-t", "rsa", "-N", "").CombinedOutput()
		output := string(stderr)
		if err != nil {
			panic(output)
		}
	}
	privateBytes, err = os.ReadFile(keyPath)
	if err != nil {
		panic(err)
	}
	keyPath = filepath.Join(os.TempDir(), "fssh.rsa.pub")
	authorizedKeysBytes, err = os.ReadFile(keyPath)
	if err != nil {
		panic(err)
	}
	return
}
