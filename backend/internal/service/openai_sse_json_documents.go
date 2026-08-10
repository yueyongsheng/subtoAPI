package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
)

const (
	maxOpenAIConcatenatedJSONDocuments = 16
	maxOpenAIConcatenatedJSONBytes     = 16 * 1024 * 1024
)

// splitOpenAIConcatenatedJSONDocuments recognizes the narrow corruption shape
// produced when multiple complete Responses events arrive in one transport
// message. Other malformed payloads are left untouched for normal error paths.
func splitOpenAIConcatenatedJSONDocuments(payload []byte) ([][]byte, bool) {
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 || len(payload) > maxOpenAIConcatenatedJSONBytes || json.Valid(payload) {
		return nil, false
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	documents := make([][]byte, 0, 2)
	for {
		var raw json.RawMessage
		err := decoder.Decode(&raw)
		if err != nil {
			if err == io.EOF && len(documents) > 1 {
				return documents, true
			}
			return nil, false
		}
		raw = bytes.TrimSpace(raw)
		var envelope struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, false
		}
		eventType := strings.TrimSpace(envelope.Type)
		if eventType == "" || strings.ContainsAny(eventType, "\r\n") {
			return nil, false
		}
		if len(documents) == maxOpenAIConcatenatedJSONDocuments {
			return nil, false
		}
		documents = append(documents, raw)
	}
}

type openAISSEJSONDocumentScanner struct {
	scanner *bufio.Scanner
	pending []string
	current string
}

// errOpenAIInvalidSSEData marks a frame whose data payload was truncated or
// otherwise malformed. The frame must be discarded before it reaches the
// downstream decoder; appending a later response.failed event cannot repair a
// malformed JSON data line that has already been sent.
var errOpenAIInvalidSSEData = errors.New("invalid OpenAI SSE data payload")

type openAISSELineScanner interface {
	Scan() bool
	Text() string
	Err() error
}

// validatedOpenAISSELineScanner keeps one complete SSE frame in memory before
// exposing its lines. This preserves the existing line-oriented callers while
// preventing a final partial data line from being written to the client.
type validatedOpenAISSELineScanner struct {
	source      openAISSELineScanner
	pending     []string
	current     string
	frame       []string
	deferredErr error
	frameOpen   atomic.Bool
}

func newValidatedOpenAISSELineScanner(source openAISSELineScanner) *validatedOpenAISSELineScanner {
	return &validatedOpenAISSELineScanner{source: source}
}

func (s *validatedOpenAISSELineScanner) Scan() bool {
	if s == nil {
		return false
	}
	for {
		if len(s.pending) > 0 {
			s.current = s.pending[0]
			s.pending = s.pending[1:]
			return true
		}
		if s.deferredErr != nil {
			return false
		}
		if s.source == nil {
			s.deferredErr = errors.New("nil OpenAI SSE scanner")
			return false
		}

		if s.source.Scan() {
			line := s.source.Text()
			s.frame = append(s.frame, line)
			if line != "" {
				s.frameOpen.Store(true)
				continue
			}
			s.frameOpen.Store(false)
			if err := validateOpenAISSEFrame(s.frame); err != nil {
				s.frame = nil
				s.deferredErr = err
				return false
			}
			s.pending = append(s.pending, s.frame...)
			s.frame = nil
			continue
		}

		// Scanner can return a final token without a trailing newline. Preserve a
		// complete JSON frame and synthesize its missing blank-line boundary so a
		// later failure event cannot merge into the same SSE event. Malformed data
		// is discarded before it reaches the downstream decoder.
		sourceErr := s.source.Err()
		if len(s.frame) > 0 {
			if err := validateOpenAISSEFrame(s.frame); err != nil {
				s.frame = nil
				s.frameOpen.Store(false)
				if sourceErr != nil {
					s.deferredErr = fmt.Errorf("%w: %w", err, sourceErr)
				} else {
					s.deferredErr = err
				}
				return false
			}
			s.pending = append(s.pending, s.frame...)
			s.pending = append(s.pending, "")
			s.frame = nil
			s.frameOpen.Store(false)
			if sourceErr != nil {
				s.deferredErr = sourceErr
			}
			continue
		}
		s.deferredErr = sourceErr
		return false
	}
}

func (s *validatedOpenAISSELineScanner) Text() string {
	if s == nil {
		return ""
	}
	return s.current
}

func (s *validatedOpenAISSELineScanner) Err() error {
	if s == nil {
		return errors.New("nil OpenAI SSE scanner")
	}
	return s.deferredErr
}

func (s *validatedOpenAISSELineScanner) HasOpenFrame() bool {
	return s != nil && s.frameOpen.Load()
}

func validateOpenAISSEFrame(lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	dataLines := make([]string, 0, 1)
	for _, line := range lines {
		if data, ok := extractOpenAISSEDataLine(line); ok {
			dataLines = append(dataLines, data)
		}
	}
	if len(dataLines) == 0 {
		return nil
	}
	data := strings.Join(dataLines, "\n")
	if strings.TrimSpace(data) == "[DONE]" || json.Valid([]byte(data)) {
		return nil
	}
	return errOpenAIInvalidSSEData
}

func newOpenAISSEJSONDocumentScanner(scanner *bufio.Scanner) *openAISSEJSONDocumentScanner {
	return &openAISSEJSONDocumentScanner{scanner: scanner}
}

func (s *openAISSEJSONDocumentScanner) Scan() bool {
	if len(s.pending) > 0 {
		s.current = s.pending[0]
		s.pending = s.pending[1:]
		return true
	}
	if s.scanner == nil || !s.scanner.Scan() {
		return false
	}

	line := s.scanner.Text()
	data, ok := extractOpenAISSEDataLine(line)
	if !ok {
		s.current = line
		return true
	}
	if len(data) > maxOpenAIConcatenatedJSONBytes {
		s.current = line
		return true
	}
	documents, repaired := splitOpenAIConcatenatedJSONDocuments([]byte(data))
	if !repaired {
		s.current = line
		return true
	}

	expanded := make([]string, 0, len(documents)*3)
	for i, document := range documents {
		if i > 0 {
			var envelope struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(document, &envelope)
			expanded = append(expanded, "event: "+strings.TrimSpace(envelope.Type))
		}
		expanded = append(expanded, "data: "+string(document), "")
	}
	s.current = expanded[0]
	s.pending = expanded[1:]
	return true
}

func (s *openAISSEJSONDocumentScanner) Text() string {
	return s.current
}

func (s *openAISSEJSONDocumentScanner) Err() error {
	if s.scanner == nil {
		return nil
	}
	return s.scanner.Err()
}
