package main

import (
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"

	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

// Get default location of a private key
func privateKeyPath() string {
	return os.Getenv("HOME") + "/.ssh/id_rsa"
}

// Get private key for ssh authentication
func parsePrivateKey(keyPath string) (ssh.Signer, error) {
	buff, _ := ioutil.ReadFile(keyPath)
	return ssh.ParsePrivateKey(buff)
}

// Get ssh client config for our connection
// SSH config will use 2 authentication strategies: by key and by password
func makeSSHConfig(user, authMethod, sshPassword string) (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	if authMethod == "password" {
		authMethods = []ssh.AuthMethod{
			ssh.Password(sshPassword),
		}
	} else {
		key, err := parsePrivateKey(privateKeyPath())
		if err != nil {
			return nil, err
		}

		authMethods = []ssh.AuthMethod{
			ssh.PublicKeys(key),
		}
	}
	config := ssh.ClientConfig{
		User: user,
		Auth: authMethods,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	return &config, nil
}

func handleClient(client net.Conn, sshConn *ssh.Client, remoteAddr string) {
	defer client.Close()

	log.Printf("Started a new client")

	for {
		// Establish connection with remote server
		remote, err := sshConn.Dial("tcp", remoteAddr)
		if err != nil {
			log.Println(err)
			return
		}

		wg := &sync.WaitGroup{}
		wg.Add(2)

		// Start remote -> local data transfer
		go func(wg *sync.WaitGroup) {
			_, err := io.Copy(client, remote)
			if err != nil {
				log.Println("error while copy remote->local:", err)
			}
			wg.Done()
			return
		}(wg)

		// Start local -> remote data transfer
		go func(wg *sync.WaitGroup) {
			_, err := io.Copy(remote, client)
			if err != nil {
				log.Println(err)
			}
			wg.Done()
			return
		}(wg)

		wg.Wait()
	}
}

func main() {
	viper.SetConfigName("mole")
	viper.AddConfigPath("/etc/mole/")
	viper.AddConfigPath("$HOME/.mole/")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Cannot read config, error: %v", err)
	}

	// Get config for all tunnels
	tunnelsConfigs := viper.Get("tunnels")
	wg := &sync.WaitGroup{}
	wg.Add(len(tunnelsConfigs.([]interface{})))
	for _, tunnelMap := range tunnelsConfigs.([]interface{}) {
		tunnelConfig := tunnelMap.(map[interface{}]interface{})
		sshAddress := tunnelConfig["ssh_address"].(string)
		sshUser := tunnelConfig["ssh_user"].(string)
		localAddr := tunnelConfig["local_address"].(string)
		remoteAddr := tunnelConfig["remote_address"].(string)
		authMethod := ""
		sshPassword := ""
		if tunnelConfig["ssh_auth_method"] != nil {
			authMethod = tunnelConfig["ssh_auth_method"].(string)
		}
		if tunnelConfig["ssh_password"] != nil {
			sshPassword = tunnelConfig["ssh_password"].(string)
		}
		go createTunnel(sshAddress, localAddr, remoteAddr, sshUser, authMethod, sshPassword, wg)
	}
	wg.Wait()
}

func createTunnel(sshAddr, localAddr, remoteAddr, user, authMethod, sshPassword string, wg *sync.WaitGroup) {
	// Build SSH client configuration
	cfg, err := makeSSHConfig(user, authMethod, sshPassword)
	if err != nil {
		log.Fatalln(err)
	}

	// Establish connection with SSH server
	conn, err := ssh.Dial("tcp", sshAddr, cfg)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	// Start local server to forward traffic to remote connection
	local, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Fatalln(err)
	}
	defer local.Close()

	log.Printf("Starting tunnel for local port: %s to remote port: %s from server: %s", localAddr, remoteAddr, sshAddr)

	// Handle incoming connections
	for {
		client, err := local.Accept()
		if err != nil {
			log.Fatalln(err)
		}

		go handleClient(client, conn, remoteAddr)
	}
}
