package messagesorter

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const protocolInfoCloseTag = "</_ProtocolInfo>"

var (
	errMissingServiceID    = errors.New("messagesorter: service id tag not found")
	errMissingServiceName  = errors.New("messagesorter: service name tag not found")
	errMissingProtocolInfo = errors.New("messagesorter: protocol info closing tag not found")
)

// Result captures the structured data extracted from the raw message capture.
type Result struct {
	ServiceID   string
	ServiceName string
	Messages    []string
}

var recordBlockPattern = regexp.MustCompile(`(?s)<RECORD>.*?</RECORD>`) // non-greedy removal

// Process takes the raw capture content and extracts distinct, sorted message tags.
func Process(raw string) (Result, error) {
	serviceID, err := extractTag(raw, `<ServiceID TYPE="UBYT">`, `</ServiceID>`)
	if err != nil {
		return Result{}, errMissingServiceID
	}

	serviceName, err := extractTag(raw, `<ProtocolType TYPE="STR">`, `</ProtocolType>`)
	if err != nil {
		return Result{}, errMissingServiceName
	}

	body, err := stripProtocolInfo(raw)
	if err != nil {
		return Result{}, err
	}

	body = recordBlockPattern.ReplaceAllString(body, "")

	messages, err := collectMessages(body)
	if err != nil {
		return Result{}, err
	}

	return Result{ServiceID: serviceID, ServiceName: serviceName, Messages: messages}, nil
}

// ProcessFile loads the capture located at inputPath, extracts the sorted messages, and writes
// the numbered output to the provided output directory. The output path is returned.
func ProcessFile(inputPath, outputDir string) (string, Result, error) {
	contents, err := os.ReadFile(inputPath)
	if err != nil {
		return "", Result{}, fmt.Errorf("messagesorter: read input: %w", err)
	}

	result, err := Process(string(contents))
	if err != nil {
		return "", Result{}, err
	}

	if outputDir == "" {
		outputDir = filepath.Dir(inputPath)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", Result{}, fmt.Errorf("messagesorter: ensure output dir: %w", err)
	}

	outputPath := filepath.Join(outputDir, OutputFilename(result.ServiceID, result.ServiceName))
	if err := WriteNumbered(outputPath, result.Messages); err != nil {
		return "", Result{}, err
	}

	return outputPath, result, nil
}

// OutputFilename creates the canonical output filename for a processed capture.
func OutputFilename(serviceID, serviceName string) string {
	sanitizedServiceName := sanitizeFilenameSegment(serviceName)
	sanitizedServiceID := sanitizeFilenameSegment(serviceID)
	return fmt.Sprintf("%s_%s.txt", sanitizedServiceID, sanitizedServiceName)
}

func sanitizeFilenameSegment(v string) string {
	v = strings.TrimSpace(v)
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "'", "<", "(", ">", ")", "|", "-")
	sanitized := replacer.Replace(v)
	sanitized = strings.Map(func(r rune) rune {
		if r == 0 {
			return -1
		}
		return r
	}, sanitized)
	if sanitized == "" {
		return "unknown"
	}
	return sanitized
}

// WriteNumbered writes the supplied messages to the path with 1-based numbering.
func WriteNumbered(path string, messages []string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("messagesorter: create output: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for i, msg := range messages {
		if _, err := fmt.Fprintf(writer, "%d: %s\n", i+1, msg); err != nil {
			return fmt.Errorf("messagesorter: write output: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("messagesorter: flush output: %w", err)
	}
	return nil
}

func stripProtocolInfo(raw string) (string, error) {
	idx := strings.Index(raw, protocolInfoCloseTag)
	if idx == -1 {
		return "", errMissingProtocolInfo
	}
	return raw[idx+len(protocolInfoCloseTag):], nil
}

func extractTag(raw, openTag, closeTag string) (string, error) {
	start := strings.Index(raw, openTag)
	if start == -1 {
		return "", errors.New("tag not found")
	}
	start += len(openTag)
	end := strings.Index(raw[start:], closeTag)
	if end == -1 {
		return "", errors.New("tag not found")
	}
	return raw[start : start+end], nil
}

func collectMessages(body string) ([]string, error) {
	scanner := bufio.NewScanner(strings.NewReader(body))
	set := make(map[string]struct{})
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "</") {
			continue
		}
		if len(line) < 2 || line[0] != '<' || line[len(line)-1] != '>' {
			continue
		}
		// mimic original behaviour that assumes tags wrapped in angle brackets
		inner := line[1 : len(line)-1]
		if _, ok := set[inner]; !ok {
			set[inner] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("messagesorter: scan messages: %w", err)
	}

	result := make([]string, 0, len(set))
	for msg := range set {
		result = append(result, msg)
	}
	sort.Strings(result)
	return result, nil
}
