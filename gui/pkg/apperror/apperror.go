package apperror

import (
	"context"
	"log/slog"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

type Handler struct {
	logger *slog.Logger
	ctx    context.Context
	window fyne.Window
}

func New(logger *slog.Logger, w fyne.Window) *Handler {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}

	return &Handler{
		logger: logger,
		ctx:    context.Background(),
		window: w,
	}
}

func (h *Handler) QError(msg string, err error) {
	if err == nil {
		return
	}

	h.logger.ErrorContext(h.ctx, msg,
		slog.String("error", err.Error()),
	)

	dialog.ShowError(err, h.window)
}
