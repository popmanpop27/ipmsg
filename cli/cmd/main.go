package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

func main() {
	var destinationIP string
	var port uint

	flag.StringVar(&destinationIP, "to", "", "recipient ip address")
	flag.UintVar(&port, "port", 6767, "recipient port")
	flag.Parse()

	if destinationIP == "" {

		myIP, err := getLocalIP()
		if err != nil {
			fmt.Printf("failed get local ip, err: %s\n", err.Error())
			os.Exit(1)
		}

		localIPs := getIPRange(myIP)

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

		// Создаем строку загрузки
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
				// Добавляем символ прогресса
				fmt.Print("=")
			}
			conn.Close()

			// Дополнительно можно добавить небольшой sleep, чтобы анимация была видна
			time.Sleep(time.Millisecond * 10)

			// Обновляем строку прогресса, если хотим
			if i == len(localIPs)-1 {
				fmt.Println("]") // закрываем прогресс-бар
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

func getIPRange(localip string) []string {
	ip := net.ParseIP(localip)
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

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
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

	for ip := range resChan {
		res = append(res, ip)
	}

	return res
}


func getLocalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		// интерфейс должен быть активным и не loopback
		if iface.Flags&net.FlagUp == 0 ||
			iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// отсеиваем VPN-интерфейсы по имени
		if isVPNInterface(iface.Name) {
			continue
		}

		addrs, err := iface.Addrs()
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