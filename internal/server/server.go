package server

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"ipmsg/internal/domain/models"
	"log/slog"
	"net"
	"strings"
	"sync"

	"time"

	"github.com/faiface/beep/generators"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

type MsgSaver interface {
	SaveToFile(filename string, req *models.IPmsgRequest) error 
}

type IPMsgServer struct {
	Addr 		 string
	Saver 		 MsgSaver
	SaveFilePath string
	log    		 *slog.Logger
}

func New(log *slog.Logger, 
	saver MsgSaver, 
	host string, 
	port uint16, 
	savePath string,) *IPMsgServer {
	return &IPMsgServer{
		Saver: saver,
		Addr: fmt.Sprintf("%s:%d", host, port),
		SaveFilePath: savePath,
		log: log,
	}
}

func (ipms *IPMsgServer) Init(ctx context.Context) error {

	const op = "IPMsgServer.Init()"

	Addr := ipms.Addr

	l, err := net.Listen("tcp", Addr)
	if err != nil {
		return fmt.Errorf("%s: %w", Addr, err)
	}
	defer l.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn ,err := l.Accept()
			if err != nil {
				continue
			}
			go ipms.handleConn(conn)
		}
	}
}

var (
	sampleRate = beep.SampleRate(44100)
	initOnce   sync.Once
)

func PlayBeep(freq int, dur time.Duration) error {
	// Частота дискретизации (samples per second)
	sr := beep.SampleRate(44100)

	// Инициализация speaker (один раз)
	if err := speaker.Init(sr, sr.N(time.Second/10)); err != nil {
		return err
	}

	// Создаём генератор синусоиды
	tone, err := generators.SinTone(sr, freq)
	if err != nil {
		return err
	}

	// Ограничиваем длительность сигнала
	streamer := beep.Take(sr.N(dur), tone)

	// Проигрываем
	speaker.Play(streamer)

	return nil
}


func (ipms *IPMsgServer) handleConn(conn net.Conn) {
    defer conn.Close()

    conn.SetReadDeadline(time.Now().Add(1 * time.Minute))

	reader := bufio.NewReaderSize(conn, 1024)
    
    data, err := reader.ReadBytes('\x00')
    if err != nil {
        ipms.writeError(conn, "failed read request: "+err.Error())
        return
    }

    data = bytes.TrimSuffix(data, []byte{0})

    req, err := parceRequest(string(data))
    if err != nil {
        ipms.writeError(conn, "failed parse request: "+err.Error())
        return
    }

    if err := ipms.Saver.SaveToFile(ipms.SaveFilePath, req); err != nil {
        ipms.writeError(conn, "failed save message: "+err.Error())
        return
    }

    PlayBeep(1200, time.Millisecond * 20)

    writeSuc(conn)
}


func parceRequest(req string) (*models.IPmsgRequest, error) {
	var res models.IPmsgRequest

	parts := strings.SplitN(req, "\nmsg:", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid request format")
	}

	header := parts[0]
	res.Msg = parts[1]

	_, err := fmt.Sscanf(
		header,
		"ipmsg\nfrom:%s\nlen:%d\ndate:%d",
		&res.From,
		&res.Len,
		&res.Date,
	)
	if err != nil {
		return nil, err
	}

	return &res, nil
}


func writeSuc(conn net.Conn)  {
	r := models.IPResponse{Succes: true}
	conn.Write([]byte(r.DecodeToString()))
}

func (ipm *IPMsgServer) writeError(conn net.Conn, err string)  {

	ipm.log.Error(err)

	er := models.IPResponse{Succes: false, Error: &err}

	conn.Write([]byte(er.DecodeToString()))
	conn.Close()
}