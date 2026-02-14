package fileparser

import (
	"bufio"
	"ipmsg/pkg/models"
	"os"
	"strconv"
	"strings"
	"time"
)

func ParseFile(filename string) ([]models.IPmsgRequest, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []models.IPmsgRequest
	scanner := bufio.NewScanner(file)

	// Skip header (2 lines)
	for i := 0; i < 2 && scanner.Scan(); i++ {
	}

	for {
		if !scanner.Scan() {
			break
		}

		meta := strings.TrimSpace(scanner.Text())
		if meta == "" {
			continue
		}

		// Split metadata line
		parts := strings.Split(meta, "|")
		if len(parts) != 3 {
			continue
		}

		// TIME
		t, err := time.Parse(time.DateTime, strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, err
		}

		// FROM / ALIAS
		fromRaw := strings.TrimSpace(parts[1])
		from := fromRaw
		alias := ""

		if strings.Contains(fromRaw, "(") && strings.HasSuffix(fromRaw, ")") {
			i := strings.Index(fromRaw, "(")
			alias = fromRaw[:i]
			from = strings.TrimSuffix(fromRaw[i+1:], ")")
		}

		// LEN
		l, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			return nil, err
		}

		// Read message body
		var msgLines []string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" {
				break
			}
			msgLines = append(msgLines, line)
		}

		result = append(result, models.IPmsgRequest{
			From:  from,
			Alias: alias,
			Len:   l,
			Date:  t.Unix(),
			Msg:   strings.Join(msgLines, "\n"),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}