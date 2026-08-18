package collector

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/2H-K/pulse/internal/models"
	"github.com/2H-K/pulse/internal/parser"
)

type Collector struct {
	parser      *parser.LogParser
	output      chan *models.LogEntry
	sources     map[string]bool
	mu          sync.RWMutex
	wg          sync.WaitGroup
	cancelChans map[string]chan struct{}
}

func NewCollector(bufferSize int) *Collector {
	return &Collector{
		parser:      parser.New(),
		output:      make(chan *models.LogEntry, bufferSize),
		sources:     make(map[string]bool),
		cancelChans: make(map[string]chan struct{}),
	}
}

func (c *Collector) Output() <-chan *models.LogEntry {
	return c.output
}

func (c *Collector) AddSource(name string, reader io.Reader) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.sources[name]; exists {
		return
	}

	c.sources[name] = true
	cancelCh := make(chan struct{})
	c.cancelChans[name] = cancelCh

	c.wg.Add(1)
	go c.readLoop(name, reader, cancelCh)
}

func (c *Collector) AddFileSource(name string, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file %s: %w", filePath, err)
	}

	c.AddSource(name, file)
	return nil
}

func (c *Collector) AddTCPSource(addr string) error {
	// Placeholder for TCP collector
	return fmt.Errorf("TCP source not yet implemented")
}

func (c *Collector) RemoveSource(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cancelCh, exists := c.cancelChans[name]; exists {
		close(cancelCh)
		delete(c.cancelChans, name)
		delete(c.sources, name)
	}
}

func (c *Collector) readLoop(name string, reader io.Reader, cancel chan struct{}) {
	defer c.wg.Done()

	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for {
		select {
		case <-cancel:
			return
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				select {
				case <-cancel:
					return
				case <-time.After(100 * time.Millisecond):
					continue
				}
			}

			select {
			case <-cancel:
				return
			case <-time.After(50 * time.Millisecond):
				continue
			}
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		entry := c.parser.Parse(line, name)
		if entry == nil {
			continue
		}

		if entry.ID == "" {
			entry.ID = generateLogID()
		}

		select {
		case c.output <- entry:
		case <-cancel:
			return
		}
	}
}

func (c *Collector) Close() {
	c.mu.Lock()
	for name, cancelCh := range c.cancelChans {
		close(cancelCh)
		delete(c.cancelChans, name)
	}
	c.mu.Unlock()

	c.wg.Wait()
	close(c.output)
}

func (c *Collector) ActiveSources() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	sources := make([]string, 0, len(c.sources))
	for s := range c.sources {
		sources = append(sources, s)
	}
	return sources
}

func generateLogID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "log_" + hex.EncodeToString(b)
}

func ParseAndClassify(line string, source string, p *parser.LogParser) *models.LogEntry {
	entry := p.Parse(line, source)
	if entry == nil {
		return nil
	}
	if entry.ID == "" {
		entry.ID = generateLogID()
	}
	return entry
}

func ExtractKeyValuePairs(line string) map[string]string {
	pairs := make(map[string]string)
	parts := strings.Fields(line)

	for _, part := range parts {
		if idx := strings.Index(part, "="); idx > 0 {
			key := part[:idx]
			val := strings.Trim(part[idx+1:], "\"'")
			pairs[key] = val
		}
	}

	return pairs
}
