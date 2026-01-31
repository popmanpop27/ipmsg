package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
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

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
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
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}
