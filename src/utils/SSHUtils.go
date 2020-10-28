package utils

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	SshPort        = "22"
	PrivateKeyPath = "/home/cms/.ssh/id_rsa" // TODO: load like hostkey
)

func ConnectSSH(user string, host string) *ssh.Client {
	// get host public key
	hostKey := getHostKey(host)
	fmt.Println(hostKey)
	// ssh client config
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			getPublicKey(PrivateKeyPath),
		},
		// allow any host key to be used (non-prod)
		//HostKeyCallback: ssh.InsecureIgnoreHostKey(),

		// verify host public key
		HostKeyCallback: ssh.FixedHostKey(hostKey),
		// optional host key algo list
		HostKeyAlgorithms: []string{
			ssh.KeyAlgoRSA,
			ssh.KeyAlgoDSA,
			ssh.KeyAlgoECDSA256,
			ssh.KeyAlgoECDSA384,
			ssh.KeyAlgoECDSA521,
			ssh.KeyAlgoED25519,
		},
		// optional tcp connect timeout
		Timeout: 5 * time.Second,
	}

	// Connect via ssh
	client, err := ssh.Dial("tcp", host+":"+SshPort, config)
	if err != nil {
		panic(err)
	}
	return client
}

func getHostKey(host string) ssh.PublicKey {
	// parse OpenSSH known_hosts file
	// ssh or use ssh-keyscan to get initial key
	file, err := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		panic(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var hostKey ssh.PublicKey
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) != 3 {
			continue
		}
		if strings.Contains(fields[0], host) {
			var err error
			hostKey, _, _, _, err = ssh.ParseAuthorizedKey(scanner.Bytes())
			if err != nil {
				log.Fatalf("error parsing %q: %v", fields[2], err)
			}
			break
		}
	}

	if hostKey == nil {
		panic("no hostkey found for " + host)
	}

	return hostKey
}

// From: https://medium.com/tarkalabs/ssh-recipes-in-go-part-one-5f5a44417282
func getPublicKey(path string) ssh.AuthMethod {
	key, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		panic(err)
	}
	return ssh.PublicKeys(signer)
}

// Source: https://medium.com/tarkalabs/ssh-recipes-in-go-part-one-5f5a44417282
func RunCommandSSH(cmd string, conn *ssh.Client) {
	// start session
	sess, err := conn.NewSession()
	if err != nil {
		panic(err)
	}
	defer sess.Close()
	// setup standard out and error
	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr
	//fmt.Println(cmd)
	err = sess.Run(cmd) // Execute command
	//if err != nil && !strings.Contains(cmd, "pkill") {
	//	panic(err)
	//}

	_ = sess.Close()
}
