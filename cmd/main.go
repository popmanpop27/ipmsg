package main

import (
	"context"
	"flag"
	"fmt"
	"ipmsg/internal/beep"
	"ipmsg/internal/filesaver"
	"ipmsg/internal/server"
	"log/slog"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"syscall"

	"ipmsg/pkg/alias"
)

func main()  {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		runtime.LockOSThread()

		beep.Init()
		<-ctx.Done()
		beep.Close()
	}()
	
	if err := createDirInHome("ipmsg"); err != nil {
		fmt.Println("failed create ipmsg dir in home dir")
		os.Exit(1)
	}

	const (
		defaultHost     = "0.0.0.0"
		defaultPort     =  6767
	)
	log := slog.Default()

	defaultSavePath, err := createFile("ipmsg.txt", "")

	if err != nil {
		log.Error("failed create storage file", "err", err)
		os.Exit(1)
	}

	defaultAliasPath, err := createFile("ipmsg/alias.txt", "")
	if err != nil {
		log.Error("failed create alias file", "err", err)
		os.Exit(1)
	}

	var savePath, host string
	var port uint
	var aliasPath string
	flag.StringVar(&savePath, "save_path", defaultSavePath, "path to file with messages")
	flag.StringVar(&host, "host", defaultHost, "host")
	flag.UintVar(&port, "port", defaultPort, "port")
	flag.StringVar(&aliasPath, "alias_path", defaultAliasPath, "path to file with aliases")
	flag.Parse()

	if port > 65535 {
		log.Error("invalid port")
		os.Exit(1)
	}

	alsManager := alias.New(aliasPath)
	als, err := alsManager.GetNames()
	if err != nil {
		log.Error("failed get alias files")
		os.Exit(1)
	}
	fileWriter := filesaver.New(als)
	server := server.New(log, fileWriter, host, uint16(port), savePath, alsManager)

	go func() {
		if err := server.Init(ctx); err != nil {
			log.Error("failed init server", "err", err)
			os.Exit(1)
		}
	}()
	log.Info("starting TCP server", "addr", fmt.Sprintf("%s:%d", host, uint16(port)))

	gracefulStop(log, cancel)
}

func gracefulStop(log *slog.Logger, cancel func()) {

	var sig os.Signal
	sysStop := make(chan os.Signal, 1)
	signal.Notify(sysStop, syscall.SIGTERM, syscall.SIGINT)

	sig = <-sysStop
	cancel()
	log.Info("application fully stopped", slog.String("SIGNAL", sig.String()))
}

// createFile creates file in path, if it null creates in user home path
func createFile(filename, path string) (string, error) {
	// using home directory if path is null
	if path == "" {
		currentUser, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("failed get target user: %w", err)
		}
		path = currentUser.HomeDir
	}

	filePath := filepath.Join(path, filename)

	// creating file if it not exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {		
		file, err := os.Create(filePath)
		if err != nil {
			return "", fmt.Errorf("failed create file: %w", err)
		}
		defer file.Close()
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