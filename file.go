package chatproxy

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/cixtor/readability"
)

func MessageFromFile(path string) (message string, tokenLen int, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	content := ""
	for scanner.Scan() {
		content += scanner.Text()
	}

	message = fmt.Sprintf("--%s--\n%s\n", path, content)
	tokenLen = guessTokens(message)
	return message, tokenLen, nil
}

func (c *ChatGPTClient) MessageFromFiles(path string) (string, error) {
	message := ""
	totalTokenLength := 0

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Ignore hidden files
		if filepath.Base(path)[0] == '.' {
			if info.IsDir() {
				return filepath.SkipDir // If it's a directory, skip it entirely
			}
			return nil // If it's a file, just skip this file
		}

		if !info.IsDir() { // check if it's a file and not a directory
			m, tl, err := MessageFromFile(path)
			if err != nil {
				return err
			}
			fmt.Fprintf(c.output, "Tokens: %d -> %s\n", tl, path)
			message += m
			totalTokenLength += tl
		}

		return nil
	})
	if err != nil {
		return "", err
	}
	fmt.Fprintf(c.output, "Estimated Total Tokens: %d\n", totalTokenLength)

	return message, nil
}

func MessageToFile(content string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	fmt.Fprintln(file, content)
	return nil
}

func guessTokens(input string) int {
	return len(input) / 2
}

func CreateAuditLog() (*os.File, error) {
	auditLogDir, err := getAuditLogDir()
	if err != nil {
		return nil, err
	}
	dateTimeString := time.Now().Format("2006-01-02_15:04:05")
	return os.Create(filepath.Join(auditLogDir, fmt.Sprintf("%s.log", dateTimeString)))
}

func getAuditLogDir() (string, error) {
	// Use XDG_STATE_HOME if available, otherwise fallback to default
	xdgStateHome := os.Getenv("XDG_STATE_HOME")
	if xdgStateHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		xdgStateHome = filepath.Join(home, ".local", "state")
	}

	// Create your application's specific directory for storing audit logs
	appAuditLogDir := filepath.Join(xdgStateHome, "chatproxy", "audit_logs")
	err := os.MkdirAll(appAuditLogDir, 0700)
	if err != nil {
		return "", err
	}

	return appAuditLogDir, nil
}

// inputOutput takes a path, checks if it is a file or URL, and returns the
// contents of the file or the text of the URL.
func (c *ChatGPTClient) inputOutput(path string) (msg string, err error) {
	_, err = os.Stat(path)
	if err == nil {
		msg, err = c.MessageFromFiles(path)
		if err != nil {
			return "", err
		}
	} else {
		_, err := url.ParseRequestURI(path)
		if err != nil {
			path = "https://" + path
		}
		resp, err := http.Get(path)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		r := readability.New()
		article, err := r.Parse(resp.Body, path)
		if err != nil {
			return "", err
		}
		msg = article.TextContent
	}
	return msg, nil
}
