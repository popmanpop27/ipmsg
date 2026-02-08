package filesaver

import (
	"bufio"
	"fmt"
	"ipmsg/internal/domain/models"
	"ipmsg/pkg/alias"
	"os"
	"time"
)


type FileSaver struct {
}

func New(al map[string]string) *FileSaver {
	return &FileSaver{
	}
}

func (fs *FileSaver) SaveToFile(filename string, req *models.IPmsgRequest, alSaver *alias.Alias) error {
	const op = "filesaver.SaveToFile"

	if req.Alias != "" {
		if err := alSaver.AddName(req.Alias, req.From); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(filename, os.O_CREATE| os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer file.Close()

	buff := bufio.NewScanner(file)

	buff.Scan()
	if len(buff.Text()) == 0 {
		if err := writeTableHeaders(file); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	from := req.From

	aliases, err := alSaver.GetNames()
	if err != nil {
		return err
	}

	if name, exists := aliases[req.From]; exists {
		from = fmt.Sprintf("%s(%s)", name, req.From)
	}

	_, err = fmt.Fprintf(
		file,
		"%-20s | %-30s | %6d\n%s\n\n",
		time.Unix(req.Date, 0).Format(time.DateTime),
		from,
		req.Len,
		req.Msg,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func writeTableHeaders(w *os.File) error {
	headers := fmt.Sprintf(
		"%-20s | %-30s | %6s\n%s\n",
		"TIME",
		"FROM",
		"LEN",
		"---------------------------------------------------------------",
	)

	_, err := w.WriteString(headers)
	return err
}
