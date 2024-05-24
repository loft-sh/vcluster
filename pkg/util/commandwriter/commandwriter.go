package commandwriter

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/loft-sh/log/scanner"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"github.com/loft-sh/vcluster/pkg/util/ringbuffer"
	"k8s.io/klog/v2"
)

type CommandWriter interface {
	Close()
	CloseAndWait(ctx context.Context, err error)
	Writer() io.Writer
}

func NewCommandWriter(component string, useRingBuffer bool) (CommandWriter, error) {
	if useRingBuffer {
		writer, err := NewRingBufferWriter(component)
		if err != nil {
			return nil, fmt.Errorf("creating pipe writer: %w", err)
		}

		return writer, nil
	}

	writer, err := NewPipeWriter(component)
	if err != nil {
		return nil, fmt.Errorf("creating pipe writer: %w", err)
	}

	return writer, nil
}

func NewRingBufferWriter(component string) (CommandWriter, error) {
	return &ringBufferWriter{
		component: component,
		buffer:    ringbuffer.NewBuffer(20 * 1024),
	}, nil
}

type ringBufferWriter struct {
	buffer *ringbuffer.Buffer

	component string
}

func (r *ringBufferWriter) Close() {}

func (r *ringBufferWriter) CloseAndWait(ctx context.Context, err error) {
	// regular stop case
	if err != nil && err.Error() != "signal: killed" {
		out, _ := io.ReadAll(r.buffer)
		klog.FromContext(ctx).Info("error running " + r.component + ":\n" + string(out))
	}
}

func (r *ringBufferWriter) Writer() io.Writer {
	return r.buffer
}

func NewPipeWriter(component string) (CommandWriter, error) {
	writer := &commandWriter{
		done: make(chan struct{}),

		component: component,
	}

	err := writer.Start()
	if err != nil {
		return nil, err
	}

	return writer, nil
}

type commandWriter struct {
	done chan struct{}

	reader io.ReadCloser
	writer io.WriteCloser

	component string
}

func (c *commandWriter) Writer() io.Writer {
	return c.writer
}

func (c *commandWriter) Close() {
	_ = c.writer.Close()
}

func (c *commandWriter) CloseAndWait(_ context.Context, _ error) {
	c.Close()
	<-c.done
}

func (c *commandWriter) Start() error {
	// create os pipe
	var err error
	c.reader, c.writer, err = os.Pipe()
	if err != nil {
		return err
	}

	// start func
	c.done = make(chan struct{})
	go func() {
		defer close(c.done)

		// make sure we scan the output correctly
		scan := scanner.NewScanner(c.reader)
		for scan.Scan() {
			line := scan.Text()
			if len(line) == 0 {
				continue
			}

			// print to our logs
			args := []interface{}{"component", c.component}
			loghelper.PrintKlogLine(line, args)
		}
	}()

	return nil
}
