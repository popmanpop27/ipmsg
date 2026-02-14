package main

import (
	"bytes"
	"ipmsg-gui/pkg/apperror"
	"ipmsg/pkg/models"
	"ipmsg/pkg/fileparser"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var messagesShowed = map[models.IPmsgRequest]struct{}{}

func showMessages(container *fyne.Container, filepath string) error {
	messages, err := fileparser.ParseFile(filepath)
	if err != nil {
		return err
	}

	for _, ms := range messages {
		if _, showed := messagesShowed[models.IPmsgRequest(ms)]; showed{
			continue
		}
		addMessage(container, ms.From, ms.Date, ms.Msg)
		messagesShowed[models.IPmsgRequest(ms)] = struct{}{}
	}

	return nil
}

func main() {
	log := slog.Default()

	a := app.New()
	w := a.NewWindow("ipmsg gui")
	w.Resize(fyne.NewSize(600, 400))

	appError := apperror.New(log, w)

	homeUs, err := os.UserHomeDir()
	if err != nil {
		appError.QError("failed get home dir", err)
		return
	}

	msgPath := filepath.Join(homeUs, "ipmsg.txt")

	/* -------- Message Area -------- */

	messageContainer := container.NewVBox()
	scroll := container.NewVScroll(messageContainer)
	scroll.SetMinSize(fyne.NewSize(600, 300))
	scroll.ScrollToTop()

	/* -------- Input Area -------- */

	input := widget.NewEntry()
	input.SetPlaceHolder("Type message...")

	sendBtn := widget.NewButton("Send", func() {
		text := input.Text
		if text == "" {
			return
		}

		out, err := runIPMsgWithInput(text)
		if err != nil {
			appError.QError("failed send message", err)
		}

		log.Info("ipmsg output", "out", out)

		if err = showMessages(messageContainer, msgPath); err != nil {
			appError.QError("failed show messages", err)
		}

		input.SetText("")
		scroll.ScrollToTop()
	})

	input.OnSubmitted = func(string) {
		sendBtn.OnTapped()
	}

	bottom := container.NewBorder(nil, nil, nil, sendBtn, input)

	/* -------- Layout -------- */

	content := container.NewBorder(
		nil,
		bottom,
		nil,
		nil,
		scroll,
	)

	w.SetContent(content)

	/* -------- Load messages -------- */

	if err := showMessages(messageContainer, msgPath); err != nil {
		appError.QError("failed show messages", err)
	}

	w.ShowAndRun()
	log.Info("running")
}

/* ---------- Message Block ---------- */

func addMessage(messageContainer *fyne.Container, from string, date int64, msg string) {
	// Create labels
	timeLabel := widget.NewLabel(
		time.Unix(date, 0).Format("2006-01-02 15:04:05")+" - "+from,
	)
	messageLabel := widget.NewLabel(msg)
	messageLabel.Wrapping = fyne.TextWrapWord

	// Create a vertical box with 2 widgets
	block := widget.NewCard("", "", container.NewVBox(timeLabel, messageLabel))

	// Add to the message container
	messageContainer.Add(block)
}

func runIPMsgWithInput(input string) (string, error) {
	cmd := exec.Command("ipmsg")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	// Send input to ipmsg
	if _, err := stdin.Write([]byte(input + "\n")); err != nil {
		return "", err
	}
	stdin.Close() // finish input

	err = cmd.Wait()
	if err != nil {
		return out.String(), err
	}

	return out.String(), nil
}