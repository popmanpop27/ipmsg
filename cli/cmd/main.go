package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"ipmsgcli/internal/cache"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

type Cache interface {
	GetIps() ([]string, error)
	UpdateIps(ips []string) error
}

var noCache bool

func main() {
	var destinationIP string
	var port uint

	if err := createDirInHome("ipmsg"); err != nil {
		fmt.Println("failed create ipmsg dir in home dir")
		os.Exit(1)
	}

	defaultCachePath, err := createFile("ipmsg/cache.json", "")
	if err != nil {
		fmt.Println("failed create cache file error: " + err.Error())
		os.Exit(1)
	}

	cachePath := ""

	flag.StringVar(&destinationIP, "to", "", "recipient ip address")
	flag.UintVar(&port, "port", 6767, "recipient port")
	flag.StringVar(&cachePath, "cache", defaultCachePath, "path to json file with cache")
	flag.BoolVar(&noCache, "scan", false, "if set ipmsg ignores cache file and scan network")
	flag.Parse()


	cacheManager, err := cache.New(cachePath)
	if err != nil {
		fmt.Println("failed init cache, err: " + err.Error())
		os.Exit(1)
	}

	if destinationIP == "" {

		myIP, err := getLocalIP()
		if err != nil {
			fmt.Printf("failed get local ip, err: %s\n", err.Error())
			os.Exit(1)
		}

		localIPs := getIPRange(myIP, cacheManager)

		fmt.Print("ip range: ")
		fmt.Println(localIPs)

		fmt.Println("Type your message, to stop typing press 'CTRL+D'")

		msgText, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Printf("failed read from stdin, err: %s\n", err.Error())
			os.Exit(1)
		}

		length := len(msgText)
		suc := 0


		fmt.Print("Sending: [")
		for i, ip := range localIPs {

			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), time.Millisecond*200)
			if err != nil {
				continue
			}

			_, err = conn.Write([]byte(fmt.Sprintf(
				"ipmsg\nfrom:%s\nlen:%d\ndate:%d\nmsg:%s\x00",
				myIP,
				length,
				time.Now().Unix(),
				msgText,
			)))
			if err == nil {
				suc++
				fmt.Print("=")
			}
			conn.Close()

			time.Sleep(time.Millisecond * 10)

			if i == len(localIPs)-1 {
				fmt.Println("]") 
			}
		}

		fmt.Printf("] Success sent to %d machines in local net\n", suc)
		return
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", destinationIP, port))
	if err != nil {
		fmt.Printf("failed connect to %s err: %s\n", destinationIP, err.Error())
		os.Exit(1)
	}

	myIP, err := getLocalIP()
	if err != nil {
		fmt.Printf("failed get local ip, err: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Println("Type your message, to stop typing press 'CTRL+D'")

	msgText, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Printf("failed read from stdin, err: %s\n", err.Error())
		os.Exit(1)
	}

	length := len(msgText)

	_, err = conn.Write([]byte(fmt.Sprintf(
		"ipmsg\nfrom:%s\nlen:%d\ndate:%d\nmsg:%s\x00",
		myIP,
		length,
		time.Now().Unix(),
		msgText,
	)))
	if err != nil {
		fmt.Printf("failed send msg, err: %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Println("Sent to 1 machine")
}

func getIPRange(localIP string, cache Cache) []string {

	if !noCache {		
		cached, err := cache.GetIps()
		if err == nil && len(cached) > 0 {
			return cached
		} else {
				fmt.Printf("Failed get ip`s from cache, pinging network")
		}
	} else {
		fmt.Println("Ignoring cache file")
	}

	ip := net.ParseIP(localIP)
	ipv4 := ip.To4()

	res := []string{}
	var wg sync.WaitGroup
	resChan := make(chan string, 255)

	for i := 1; i <= 255; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nIP := fmt.Sprintf("%d.%d.%d.%d", ipv4[0], ipv4[1], ipv4[2], i)
			pinger := probing.New(nIP)
			pinger.Count = 1

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()

			if err := pinger.RunWithContext(ctx); err == nil && pinger.PacketsRecv > 0 {
				resChan <- nIP
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(resChan)
	}()

	fmt.Print("\n[")

	for ip := range resChan {
		fmt.Print("=")
		res = append(res, ip)
	}
	fmt.Printf("]\n")

	err := cache.UpdateIps(res)
	if err != nil {
		fmt.Println("failed update cache, err: " + err.Error())
	}

	return res
}


func getLocalIP() (string, error) {

	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, face := range interfaces {
		if face.Flags&net.FlagUp == 0 ||
			face.Flags&net.FlagLoopback != 0 {
			continue
		}

		if isVPNInterface(face.Name) {
			continue
		}

		addrs, err := face.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil ||
				ip.IsLoopback() ||
				ip.To4() == nil {
				continue
			}

			return ip.String(), nil
		}
	}

	return "", errors.New("no suitable local IP found")
}

func isVPNInterface(name string) bool {
	return strings.HasPrefix(name, "tun") ||
		strings.HasPrefix(name, "tap") ||
		strings.HasPrefix(name, "wg") ||
		strings.HasPrefix(name, "ppp")
}


func createFile(filename, path string) (string, error) {
	if path == "" {
		currentUser, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("failed get target user: %w", err)
		}
		path = currentUser.HomeDir
	}

	filePath := filepath.Join(path, filename)

	 _, err := os.Stat(filePath)
	if os.IsNotExist(err) {		
		file, err := os.Create(filePath)
		if err != nil {
			return "", fmt.Errorf("failed create file: %w", err)
		}
		defer file.Close()
	} else if err != nil {
		return "", err
	}


	return filePath, nil
}

func createDirInHome(path string) error {

	userHome, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(userHome, path), 0755)
	if err != nil {
		return err
	}

	return nil
}