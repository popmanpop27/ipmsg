package server

import (
	"bufio"
	"bytes"
	"context"
	"fmt"

	"ipmsg/internal/beep"
	"ipmsg/pkg/models"
	"ipmsg/pkg/alias"
	"log/slog"
	"net"
	"strings"

	"time"
)

type MsgSaver interface {
	SaveToFile(filename string, req *models.IPmsgRequest, alSaver *alias.Alias) error 
}

type IPMsgServer struct {
	Addr 		 string
	Saver 		 MsgSaver
	SaveFilePath string
	log    		 *slog.Logger
	alias        *alias.Alias
}

func New(log *slog.Logger, 
	saver MsgSaver, 
	host string, 
	port uint16, 
	savePath string,
	alias *alias.Alias,
	) *IPMsgServer {
	return &IPMsgServer{
		Saver: saver,
		Addr: fmt.Sprintf("%s:%d", host, port),
		SaveFilePath: savePath,
		log: log,
		alias: alias,
	}
}

func (ipServer *IPMsgServer) Init(ctx context.Context) error {

	const op = "IPMsgServer.Init()"

	Addr := ipServer.Addr

	l, err := net.Listen("tcp", Addr)
	if err != nil {
		return fmt.Errorf("%s: %w", Addr, err)
	}
	
	go func() {
		<-ctx.Done()
		l.Close()
		ipServer.log.Warn("opening port")
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn ,err := l.Accept()
			if err != nil {
				continue
			}
			go ipServer.handleConn(conn, ctx)
		}
	}
}


func (ipServer *IPMsgServer) handleConn(conn net.Conn, ctx context.Context) {
    defer conn.Close()

    conn.SetReadDeadline(time.Now().Add(1 * time.Minute))

	go func() {
		<- ctx.Done()
		conn.Close()
	}()

	reader := bufio.NewReaderSize(conn, 1024)
    
    data, err := reader.ReadBytes('\x00')
    if err != nil {
        ipServer.writeError(conn, "failed read request: "+err.Error())
        return
    }

    data = bytes.TrimSuffix(data, []byte{0})

    req, err := ipServer.parseRequest(string(data))
    if err != nil {
        ipServer.writeError(conn, "failed parse request: "+err.Error())
        return
    }

    if err := ipServer.Saver.SaveToFile(ipServer.SaveFilePath, req, ipServer.alias); err != nil {
        ipServer.writeError(conn, "failed save message: "+err.Error())
        return
    }

    beep.Beep()

    writeSuc(conn)
}


func (ipServer *IPMsgServer) parseRequest(req string) (*models.IPmsgRequest, error) {
	var res models.IPmsgRequest

	parts := strings.SplitN(req, "\nmsg:", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid request format")
	}

	header := parts[0]
	res.Msg = parts[1]

	_, err := fmt.Sscanf(
		header,
		"ipmsg\nfrom:%s\nlen:%d\ndate:%d\nalias:%s",
		&res.From,
		&res.Len,
		&res.Date,
		&res.Alias,
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

func (ipServer *IPMsgServer) writeError(conn net.Conn, err string)  {

	ipServer.log.Error(err)

	er := models.IPResponse{Succes: false, Error: &err}

	conn.Write([]byte(er.DecodeToString()))
	conn.Close()
}

