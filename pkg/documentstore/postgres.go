package documentstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"

	"github.com/getzep/zep/pkg/models"
	"github.com/uptrace/bun"
)

// NewPostgresDocumentStore returns a new PostgresDocumentStore. Use this to correctly initialize the store.
func NewPostgresDocumentStore(
	appState *models.AppState,
	client *bun.DB,
) (*PostgresDocumentStore, error) {
	if appState == nil {
		return nil, NewDocumentStorageError("nil appState received", nil)
	}

	pds := &PostgresDocumentStore{BaseDocumentStore[*bun.DB]{Client: client}}
	err := pds.OnStart(context.Background(), appState)
	if err != nil {
		return nil, NewDocumentStorageError("failed to run OnInit", err)
	}
	return pds, nil
}

// Force compiler to validate that PostgresDocumentStore implements the DocumentStore interface.
var _ models.DocumentStore[*bun.DB] = &PostgresDocumentStore{}

type PostgresDocumentStore struct {
	BaseDocumentStore[*bun.DB]
}

func (pds *PostgresDocumentStore) OnStart(
	_ context.Context,
	appState *models.AppState,
) error {
	err := initPostgresSetup(context.Background(), appState, pds.Client)
	if err != nil {
		return NewDocumentStorageError("failed to ensure postgres schema setup", err)
	}

	return nil
}

func (pds *PostgresDocumentStore) GetClient() *bun.DB {
	return pds.Client
}

// GetCollection retrieves a Collection for a given collectionID.
func (pds *PostgresDocumentStore) GetCollection(
	ctx context.Context,
	_ *models.AppState,
	collectionID string,
) (*models.Collection, error) {
	return getCollection(ctx, pds.Client, collectionID)
}

// PutCollection stores a Collection for a given collectionID.
func (pds *PostgresDocumentStore) PutCollection(
	ctx context.Context,
	_ *models.AppState,
	collection *models.Collection,
) (*models.Collection, error) {
	// TODO: pass the whole collection to putCollection
	return putCollection(ctx, pds.Client, collection.CollectionID, collection.Metadata, false)
}

// GetDocument retrieves a Document for a given collectionID and documentID.
func (pds *PostgresDocumentStore) GetDocument(
	ctx context.Context,
	_ *models.AppState,
	documentID string,
	collectionID string,
) (*models.Document, error) {
	return getDocument(ctx, pds.Client, collectionID, documentID)
}

// PutDocument creates or updates a Document for a given collectionID and documentID.
func (pds *PostgresDocumentStore) PutDocument(
	ctx context.Context,
	_ *models.AppState,
	documentID string,
	collectionID string,
	document *models.Document,
) error {
	// TODO: pass the whole document to putDocument
	_, err := putDocument(ctx, pds.Client, collectionID, document.DocumentID, document.Metadata, false)
	return err
}

func (pds *PostgresDocumentStore) Close() error {
	if pds.Client != nil {
		return pds.Client.Close()
	}
	return nil
}

// DeleteDocument deletes a document from the document store. This is a soft delete.
// TODO: A hard delete will be implemented as an out-of-band process or left to the implementer.
func (pds *PostgresDocumentStore) DeleteDocument(ctx context.Context, collectionID string, documentID string) error {
	return deleteDocument(ctx, pds.Client, collectionID, documentID)
}

func (pds *PostgresDocumentStore) PurgeDeleted(ctx context.Context) error {
	err := purgeDeleted(ctx, pds.Client)
	if err != nil {
		return NewDocumentStorageError("failed to purge deleted", err)
	}

	return nil
}

// acquireAdvisoryXactLock acquires a PostgreSQL advisory lock for the given key.
// Expects a transaction to be open in tx.
// `pg_advisory_xact_lock` will wait until the lock is available. The lock is released
// when the transaction is committed or rolled back.
func acquireAdvisoryXactLock(ctx context.Context, tx bun.Tx, key string) error {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	hash := hasher.Sum(nil)
	lockID := binary.BigEndian.Uint64(hash[:8])

	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(?)", lockID); err != nil {
		return NewDocumentStorageError("failed to acquire advisory lock", err)
	}

	return nil
}

// acquireAdvisoryLock acquires a PostgreSQL advisory lock for the given key.
// The lock needs to be released manually by calling releaseAdvisoryLock.
// Accepts a bun.IDB, which can be either a *bun.DB or *bun.Tx.
// Returns the lock ID.
func acquireAdvisoryLock(ctx context.Context, db bun.IDB, key string) (uint64, error) {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	hash := hasher.Sum(nil)
	lockID := binary.BigEndian.Uint64(hash[:8])

	if _, err := db.ExecContext(ctx, "SELECT pg_advisory_lock(?)", lockID); err != nil {
		return 0, NewDocumentStorageError("failed to acquire advisory lock", err)
	}

	return lockID, nil
}

// releaseAdvisoryLock releases a PostgreSQL advisory lock for the given key.
// Accepts a bun.IDB, which can be either a *bun.DB or *bun.Tx.
func releaseAdvisoryLock(ctx context.Context, db bun.IDB, lockID uint64) error {
	if _, err := db.ExecContext(ctx, "SELECT pg_advisory_unlock(?)", lockID); err != nil {
		return NewDocumentStorageError("failed to release advisory lock", err)
	}

	return nil
}

// rollbackOnError rolls back the transaction if an error is encountered.
// If the error is sql.ErrTxDone, the transaction has already been committed or rolled back
// and we ignore the error.
func rollbackOnError(tx bun.Tx) {
	if rollBackErr := tx.Rollback(); rollBackErr != nil && !errors.Is(rollBackErr, sql.ErrTxDone) {
		log.Error("failed to rollback transaction", rollBackErr)
	}
}
