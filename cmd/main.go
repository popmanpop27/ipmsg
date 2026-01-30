package main

import (
	"context"
	"flag"
	"fmt"
	"ipmsg/internal/filesaver"
	"ipmsg/internal/server"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main()  {

	const (
		defaultSavePath = "/usr/share/ipmsg.txt"
		defaultHost     = "localhost"
		defaultPort     = 745
	)

	log := slog.Default()

	var savePath, host string
	var port uint
	flag.StringVar(&savePath, "save_path", defaultSavePath, "path to file with messages")
	flag.StringVar(&host, "host", defaultHost, "host")
	flag.UintVar(&port, "port", defaultPort, "port")
	flag.Parse()

	if port > 65535 {
		log.Error("invalid port")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	fileWriter := filesaver.FileSaver{}
	server := server.New(log, &fileWriter, host, uint16(port), savePath)

	go func() {
		if err := server.Init(ctx); err != nil {
			log.Error("failed init server", "err", err)
			os.Exit(1)
		}
	}()
	log.Info("starting TCP server", "addr", fmt.Sprintf("%s:%d", host, uint16(port)))

	grasefullStop(log, cancel)
}

func grasefullStop(log *slog.Logger, cancel func()) {

	
	var sig os.Signal
	sysStop := make(chan os.Signal, 1)
	signal.Notify(sysStop, syscall.SIGTERM, syscall.SIGINT)

	sig = <-sysStop
	cancel()
	log.Info("application fully stoped", slog.String("SIGNAL", sig.String()))
	os.Exit(1)
}
