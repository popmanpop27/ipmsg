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
)

func main()  {

	runtime.LockOSThread()

	beep.Init()
	defer beep.Close()

	const (
		defaultHost     = "0.0.0.0"
		defaultPort     = 6767
	)
	log := slog.Default()

	defaultSavePath, err := createFile("ipmsg.txt", "")
	if err != nil {
		log.Error("failed create storage file", "err", err)
		os.Exit(1)
	}

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

// createFile создаёт файл с указанным именем в указанной директории.
// Если path пустой, файл создаётся в домашней директории пользователя.
// Возвращает полный путь к созданному файлу или ошибку.
func createFile(filename, path string) (string, error) {
	// Если путь пустой, используем домашнюю директорию
	if path == "" {
		currentUser, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("не удалось получить текущего пользователя: %w", err)
		}
		path = currentUser.HomeDir
	}

	// Формируем полный путь к файлу
	filePath := filepath.Join(path, filename)

	// Создаём файл
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("не удалось создать файл: %w", err)
	}
	defer file.Close()

	return filePath, nil
}