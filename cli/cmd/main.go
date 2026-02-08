package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"ipmsg/pkg/alias"
	"ipmsgcli/internal/cache"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var noCache bool
var port uint
var stopKey string 

func main() {
	var destinationIP string

	if err := createDirInHome("ipmsg"); err != nil {
		fmt.Println("failed create ipmsg dir in home dir")
		os.Exit(1)
	}

	if runtime.GOOS == "windows" {
		stopKey = "CTRL+Z than ENTER"
	} else {
		stopKey = "CTRL+D"
	}

	defaultCachePath, err := createFile("ipmsg/cache.json", "")
	if err != nil {
		fmt.Println("failed create cache file error: " + err.Error())
		os.Exit(1)
	}

	defaultAliasPath, err := createFile("ipmsg/alias.txt", "")
	if err != nil {
		fmt.Println("failed create alias file")
		os.Exit(1)
	}

	cachePath := ""
	aliasPath     := ""

	var newAlias string
	var addrAlias string
	flag.StringVar(&destinationIP, "to", "", "recipient ip address")
	flag.UintVar(&port, "port", 6767, "recipient port")
	flag.StringVar(&cachePath, "cache", defaultCachePath, "path to json file with cache")
	flag.BoolVar(&noCache, "scan", false, "if set ipmsg ignores cache file and scan network")
	flag.StringVar(&newAlias, "alias", "", "add new alias")
	flag.StringVar(&addrAlias, "ip", "", "add new alias(address)")
	flag.StringVar(&aliasPath, "alias_path", defaultAliasPath, "path to file where aliases saved")
	flag.Parse()

	al := alias.New(aliasPath)
	if newAlias != "" && addrAlias != "" {
		if err := al.AddName(newAlias, addrAlias); err != nil {
			fmt.Println("failed add new alias, err: " + err.Error())
			os.Exit(1)
		}
		fmt.Printf("Added alias %s %s", newAlias, addrAlias)
		return
	}

	cacheManager, err := cache.New(cachePath)
	if err != nil {
		fmt.Println("failed init cache, err: " + err.Error())
		os.Exit(1)
	}


	aliases, err := al.GetNames()
	if err != nil {
		fmt.Println("failed get aliases, err: " + err.Error())
		os.Exit(1)
	}

	fmt.Println("Aliases:")
	fmt.Println("-----------------------")
	for k, v := range aliases {
		fmt.Println(k + " " + v)
	}
	fmt.Println("-----------------------")

	if destinationIP == "" {

		myIP, err := getLocalIP()
		if err != nil {
			fmt.Printf("failed get local ip, err: %s\n", err.Error())
			os.Exit(1)
		}

		localIPs := getIPRange(myIP, cacheManager)

		fmt.Print("ip range: ")
		for _, localIp := range localIPs {
			if alias, ex := aliases[localIp]; ex {
				fmt.Printf("%s(%s) ", alias, localIp)
			} else {
				fmt.Printf("%s ", localIp)
			}
		}
		fmt.Print("\n")

		myName := getName(cacheManager)
		
		fmt.Println("Type your message, to stop typing press " + stopKey)
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
				"ipmsg\nfrom:%s\nlen:%d\ndate:%d\nalias:%s\nmsg:%s\x00",
				myIP,
				length,
				time.Now().Unix(),
				myName,
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

	if ipAlias, exists := aliases[destinationIP]; exists  {
		i := net.ParseIP(ipAlias)
		if i != nil {
			fmt.Printf("Sending to %s(%s)\n", destinationIP, ipAlias)
			destinationIP = ipAlias
		}
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
	
	myName := getName(cacheManager)

	fmt.Println("Type your message, to stop typing press " + stopKey)
	msgText, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Printf("failed read from stdin, err: %s\n", err.Error())
		os.Exit(1)
	}

	length := len(msgText)


	_, err = conn.Write([]byte(fmt.Sprintf(
		"ipmsg\nfrom:%s\nlen:%d\ndate:%d\nalias:%s\nmsg:%s\x00",
		myIP,
		length,
		time.Now().Unix(),
		myName,
		msgText,
	)))
	if err != nil {
		fmt.Printf("failed send msg, err: %s\n", err.Error())
		os.Exit(1)
	}
	fmt.Println("Sent to 1 machine")
}

func getName(mgr *cache.Cache) string {
	nameCache, _ := mgr.GetName()
	if nameCache != "" {
		return nameCache
	}

	name := ""

	fmt.Print("Write your name\n-> ")
	fmt.Scan(&name)

	ips, err := mgr.GetIps()
	if err != nil {
		fmt.Println("failed get ips, err: " + err.Error())
		os.Exit(1)
	}

	err = mgr.UpdateIps(ips, name)
	if err != nil {
		fmt.Println("failed save name to cache, err: " + err.Error())
	}

	return name
}

func getIPRange(localIP string, cache *cache.Cache) []string {
	// ===== CACHE =====
	if !noCache {
		if cached, err := cache.GetIps(); err == nil && len(cached) > 0 {
			return cached
		}
		fmt.Println("Failed to get IPs from cache, scanning network")
	} else {
		fmt.Println("Ignoring cache file")
	}

	// ===== IP VALIDATION =====
	ip := net.ParseIP(localIP)
	ipv4 := ip.To4()
	if ipv4 == nil {
		fmt.Println("Invalid IPv4 address:", localIP)
		return nil
	}

	base := fmt.Sprintf("%d.%d.%d.", ipv4[0], ipv4[1], ipv4[2])

	// ===== WORKER POOL =====
	const (
		maxHosts   = 255
		workers    = 50
		timeout    = 2 * time.Second
	)

	jobs := make(chan int, maxHosts)
	results := make(chan string, maxHosts)

	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for i := range jobs {
			ip := base + strconv.Itoa(i)
			if tcpPing(ip, timeout) {
				results <- ip
			}
		}
	}

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go worker()
	}

	for i := 1; i <= maxHosts; i++ {
		jobs <- i
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	// ===== COLLECT =====
	fmt.Print("\n[")
	res := make([]string, 0)
	for ip := range results {
		fmt.Print("=")
		res = append(res, ip)
	}
	fmt.Println("]")

	// ===== UPDATE CACHE =====
	if err := cache.UpdateIps(res, cache.Name); err != nil {
		fmt.Println("failed to update cache:", err)
	}

	return res
}

func tcpPing(ip string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), timeout)
	if err == nil {
		conn.Close()
		return true
	}
	return false
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