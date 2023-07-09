package documentstore

import (
	"context"
	"fmt"

	"github.com/getzep/zep/internal"
	"github.com/getzep/zep/pkg/models"
)

var log = internal.GetLogger()

// BaseDocumentStore is the base implementation of a DocumentStore. Client is the underlying datastore client, such as a
// database connection. The extractorObservers slice is used to store all registered Extractors.
type BaseDocumentStore[T any] struct {
	Client             T
	extractorObservers []models.Extractor
}

// Attach registers an Extractor to the MemoryStore
func (s *BaseDocumentStore[T]) Attach(observer models.Extractor) {
	s.extractorObservers = append(s.extractorObservers, observer)
}

// NotifyExtractors notifies all registered Extractors of a new MessageEvent
func (s *BaseDocumentStore[T]) NotifyExtractors(
	ctx context.Context,
	appState *models.AppState,
	eventData *models.MessageEvent,
) {
	for _, observer := range s.extractorObservers {
		go func(obs models.Extractor) {
			err := obs.Notify(ctx, appState, eventData)
			if err != nil {
				log.Errorf("BaseDocumentStore NotifyExtractors failed: %v", err)
			}
		}(observer)
	}
}

type DocumentStorageError struct {
	message       string
	originalError error
}

func (e *DocumentStorageError) Error() string {
	return fmt.Sprintf("doc storage error: %s (original error: %v)", e.message, e.originalError)
}

func NewDocumentStorageError(message string, originalError error) *DocumentStorageError {
	return &DocumentStorageError{message: message, originalError: originalError}
}

// remove?
func checkLastNParms(lastNTokens int, lastNMessages int) error {
	if lastNTokens > 0 {
		return NewDocumentStorageError("not implemented", nil)
	}

	if lastNMessages > 0 && lastNTokens > 0 {
		return NewDocumentStorageError("cannot specify both lastNMessages and lastNTokens", nil)
	}

	if lastNMessages < 0 || lastNTokens < 0 {
		return NewDocumentStorageError("cannot specify negative lastNMessages or lastNTokens", nil)
	}
	return nil
}
