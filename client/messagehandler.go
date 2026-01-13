package client

import (
	"fmt"
	"log/slog"

	"github.com/richinsley/comfy2go/graphapi"
)

// MessageHandlers defines optional callback functions for handling different message types
// from a QueueItem. All handlers are optional - only provide handlers for the messages you care about.
type MessageHandlers struct {
	// OnStarted is called when execution begins
	OnStarted func(*PromptMessageStarted)

	// OnExecuting is called when a node starts executing
	OnExecuting func(*PromptMessageExecuting)

	// OnProgress is called with progress updates during node execution
	OnProgress func(*PromptMessageProgress)

	// OnProgressState is called with detailed progress state for all nodes (new ComfyUI message)
	OnProgressState func(*PromptMessageProgressState)

	// OnData is called when output data is available
	OnData func(*PromptMessageData)

	// OnExecutionSuccess is called when execution completes successfully (new ComfyUI message)
	OnExecutionSuccess func(*PromptMessageExecutionSuccess)

	// OnStopped is called when execution stops (success, error, or interruption)
	OnStopped func(*PromptMessageStopped)

	// OnError is called if there was an exception during execution
	// This is called before OnStopped when an error occurs
	OnError func(*PromptMessageStoppedException)

	// OnComplete is called after the message loop exits, regardless of success or failure
	// Useful for cleanup operations
	OnComplete func()
}

// DefaultMessageHandlers returns MessageHandlers with sensible defaults:
// - Logs started, executing, and stopped messages
// - Logs errors
// - Does NOT include progress bars (add your own if needed)
func DefaultMessageHandlers() *MessageHandlers {
	return &MessageHandlers{
		OnStarted: func(msg *PromptMessageStarted) {
			slog.Info("Execution started", "prompt_id", msg.PromptID)
		},
		OnExecuting: func(msg *PromptMessageExecuting) {
			slog.Info("Executing node", "node_id", msg.NodeID, "title", msg.Title)
		},
		OnError: func(err *PromptMessageStoppedException) {
			slog.Error("Execution error",
				"node_id", err.NodeID,
				"node_type", err.NodeType,
				"error", err.ExceptionMessage,
			)
		},
		OnStopped: func(msg *PromptMessageStopped) {
			if msg.Exception == nil {
				slog.Info("Execution completed successfully")
			}
		},
	}
}

// WithStartedHandler adds a started handler (builder pattern)
func (h *MessageHandlers) WithStartedHandler(fn func(*PromptMessageStarted)) *MessageHandlers {
	h.OnStarted = fn
	return h
}

// WithExecutingHandler adds an executing handler (builder pattern)
func (h *MessageHandlers) WithExecutingHandler(fn func(*PromptMessageExecuting)) *MessageHandlers {
	h.OnExecuting = fn
	return h
}

// WithProgressHandler adds a progress handler (builder pattern)
func (h *MessageHandlers) WithProgressHandler(fn func(*PromptMessageProgress)) *MessageHandlers {
	h.OnProgress = fn
	return h
}

// WithProgressStateHandler adds a progress state handler (builder pattern)
func (h *MessageHandlers) WithProgressStateHandler(fn func(*PromptMessageProgressState)) *MessageHandlers {
	h.OnProgressState = fn
	return h
}

// WithDataHandler adds a data handler (builder pattern)
func (h *MessageHandlers) WithDataHandler(fn func(*PromptMessageData)) *MessageHandlers {
	h.OnData = fn
	return h
}

// WithExecutionSuccessHandler adds an execution success handler (builder pattern)
func (h *MessageHandlers) WithExecutionSuccessHandler(fn func(*PromptMessageExecutionSuccess)) *MessageHandlers {
	h.OnExecutionSuccess = fn
	return h
}

// WithStoppedHandler adds a stopped handler (builder pattern)
func (h *MessageHandlers) WithStoppedHandler(fn func(*PromptMessageStopped)) *MessageHandlers {
	h.OnStopped = fn
	return h
}

// WithErrorHandler adds an error handler (builder pattern)
func (h *MessageHandlers) WithErrorHandler(fn func(*PromptMessageStoppedException)) *MessageHandlers {
	h.OnError = fn
	return h
}

// WithCompleteHandler adds a complete handler (builder pattern)
func (h *MessageHandlers) WithCompleteHandler(fn func()) *MessageHandlers {
	h.OnComplete = fn
	return h
}

// ProcessMessages processes messages from the QueueItem using the provided handlers.
// This function blocks until execution stops or an error occurs.
// Returns an error if execution failed, nil if successful.
func (qi *QueueItem) ProcessMessages(handlers *MessageHandlers) error {
	if handlers == nil {
		handlers = &MessageHandlers{}
	}

	var executionError error

	// Ensure OnComplete is called when we exit
	if handlers.OnComplete != nil {
		defer handlers.OnComplete()
	}

	for {
		msg := <-qi.Messages

		switch msg.Type {
		case "started":
			if handlers.OnStarted != nil {
				handlers.OnStarted(msg.ToPromptMessageStarted())
			}

		case "executing":
			if handlers.OnExecuting != nil {
				handlers.OnExecuting(msg.ToPromptMessageExecuting())
			}

		case "progress":
			if handlers.OnProgress != nil {
				handlers.OnProgress(msg.ToPromptMessageProgress())
			}

		case "progress_state":
			if handlers.OnProgressState != nil {
				handlers.OnProgressState(msg.ToPromptMessageProgressState())
			}

		case "data":
			if handlers.OnData != nil {
				handlers.OnData(msg.ToPromptMessageData())
			}

		case "execution_success":
			if handlers.OnExecutionSuccess != nil {
				handlers.OnExecutionSuccess(msg.ToPromptMessageExecutionSuccess())
			}

		case "stopped":
			stopped := msg.ToPromptMessageStopped()

			// Handle error first if present
			if stopped.Exception != nil {
				if handlers.OnError != nil {
					handlers.OnError(stopped.Exception)
				}
				executionError = fmt.Errorf("execution failed: %s - %s",
					stopped.Exception.ExceptionType,
					stopped.Exception.ExceptionMessage)
			}

			// Then call stopped handler
			if handlers.OnStopped != nil {
				handlers.OnStopped(stopped)
			}

			return executionError

		default:
			slog.Warn("Unknown message type received", "type", msg.Type)
		}
	}
}

// QueuePromptAndProcess atomically queues a prompt and starts processing messages.
// This is the RECOMMENDED way to execute prompts as it avoids race conditions between
// queueing and message processing.
//
// This method:
// 1. Queues the prompt with ComfyUI
// 2. Immediately starts processing messages with the provided handlers
// 3. Blocks until execution completes or fails
// 4. Returns an error if execution failed, nil if successful
//
// Example:
//
//	err := client.QueuePromptAndProcess(graph,
//	    client.DefaultMessageHandlers().
//	        WithDataHandler(func(msg *client.PromptMessageData) {
//	            // handle output data
//	        }),
//	)
func (c *ComfyClient) QueuePromptAndProcess(graph *graphapi.Graph, handlers *MessageHandlers) error {
	// Queue the prompt
	item, err := c.QueuePrompt(graph)
	if err != nil {
		return fmt.Errorf("failed to queue prompt: %w", err)
	}

	// Immediately start processing messages (no race condition)
	return item.ProcessMessages(handlers)
}
