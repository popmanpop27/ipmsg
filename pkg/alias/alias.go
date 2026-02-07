package alias

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// package for saving aliases in file in format <address> <alias> (example: 192.168.1.1 alex)

type Alias struct {
	filePath string
}

func New(path string) *Alias {
	return &Alias{
		filePath: path,
	}
} 

var (
	ErrInvalidFormat error = errors.New("invalid file format")
)

func (a *Alias) GetNames() (map[string]string, error) {
	res := map[string]string{} // ip - name

	file, err := os.OpenFile(a.filePath, os.O_CREATE | os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		text := scanner.Text()

		parts := strings.Split(text, " ")

		if len(parts) != 2 {
			return nil, ErrInvalidFormat
		}

		res[parts[0]] = parts[1]
		res[parts[1]] = parts[0]
	}

	return res, nil
}

func (a *Alias) AddName(name string, address string) error {
	file, err := os.OpenFile(a.filePath, os.O_CREATE | os.O_WRONLY | os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(fmt.Sprintf("%s %s\n", address, name)); err != nil {
		return err
	}

	if _, err := file.WriteString(fmt.Sprintf("%s %s\n", name, address)); err != nil {
		return err
	}
	
	return nil
}