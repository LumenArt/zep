package documentstore

import (
	"context"
	"database/sql"
	"fmt"

	"dario.cat/mergo"

	"github.com/getzep/zep/pkg/models"
	"github.com/jinzhu/copier"
	"github.com/uptrace/bun"
)

// putCollection stores a new collection or updates an existing collection with new metadata.
func putDocument(
	ctx context.Context,
	db *bun.DB,
	documentID string,
	collectionID string,
	metadata map[string]interface{},
	isPrivileged bool,
) (*models.Document, error) {
	if collectionID == "" {
		return nil, NewDocumentStorageError("collectionID cannot be empty", nil)
	} else if documentID == "" {
		return nil, NewDocumentStorageError("documentID cannot be empty", nil)
	}

	// We're not going to run this in a transaction as we don't necessarily
	// need to roll back the session creation if the message metadata upsert fails.
	document := PgDocument{DocumentID: documentID, CollectionID: collectionID}
	_, err := db.NewInsert().
		Model(&document).
		Column("document_id").
		Column("collection_id").
		On("CONFLICT (document_id, collection_id) DO UPDATE"). // we'll do an upsert
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, NewDocumentStorageError("failed to put document", err)
	}

	// If the collection is deleted, return an error
	if !document.DeletedAt.IsZero() {
		return nil, NewDocumentStorageError(fmt.Sprintf("document %s is deleted", documentID), nil)
	}

	// remove the top-level `system` key from the metadata if the caller is not privileged
	if !isPrivileged {
		delete(metadata, "system")
	}

	// update the session metadata and return the session
	return putDocumentMetadata(ctx, db, documentID, collectionID, metadata)
}

// putCollectionMetadata updates the metadata for a collection. The metadata map is merged
// with the existing metadata map, creating keys and values if they don't exist.
func putDocumentMetadata(ctx context.Context,
	db *bun.DB,
	documentID string,
	collectionID string,
	metadata map[string]interface{}) (*models.Document, error) {
	// Acquire a lock for this DocumentID. This is to prevent concurrent updates
	// to the document metadata.
	lockID, err := acquireAdvisoryLock(ctx, db, documentID)
	if err != nil {
		return nil, NewDocumentStorageError("failed to acquire advisory lock", err)
	}
	defer func(ctx context.Context, db bun.IDB, lockID uint64) {
		err := releaseAdvisoryLock(ctx, db, lockID)
		if err != nil {
			log.Error(ctx, "failed to release advisory lock", err)
		}
	}(ctx, db, lockID)

	dbDocument := &PgDocument{}
	err = db.NewSelect().
		Model(dbDocument).
		Where("document_id = ?", collectionID).
		Scan(ctx)
	if err != nil {
		return nil, NewDocumentStorageError("failed to get session", err)
	}

	// merge the existing metadata with the new metadata
	dbMetadata := dbDocument.Metadata
	if err := mergo.Merge(&dbMetadata, metadata, mergo.WithOverride); err != nil {
		return nil, NewDocumentStorageError("failed to merge metadata", err)
	}

	// put the document metadata, returning the updated document
	_, err = db.NewUpdate().
		Model(dbDocument).
		Set("metadata = ?", dbMetadata).
		Where("document_id = ?", collectionID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, NewDocumentStorageError("failed to update session metadata", err)
	}

	document := &models.Document{}
	err = copier.Copy(document, dbDocument)
	if err != nil {
		return nil, NewDocumentStorageError("Unable to copy session", err)
	}

	return document, nil
}

// getCollection retrieves a document collection from the document store.
func getDocument(
	ctx context.Context,
	db *bun.DB,
	documentID string,
	collectionID string,
) (*models.Document, error) {
	document := PgDocument{}
	err := db.NewSelect().Model(&document).Where("document_id = ? AND collection_id = ?", documentID, collectionID).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, NewDocumentStorageError("failed to get document", err)
	}

	retDocument := models.Document{}
	err = copier.Copy(&retDocument, &document)
	if err != nil {
		return nil, NewDocumentStorageError("failed to copy collection", err)
	}

	return &retDocument, nil
}

// deleteDocument deletes a document from the document store. This is a soft delete.
func deleteDocument(
	ctx context.Context,
	db *bun.DB,
	documentID string,
	collectionID string,
) error {
	return deleteDocument(ctx, db, documentID, collectionID)
}

// purgeDeleted deletes all deleted documents from the document store.
func purgeDeleted(
	ctx context.Context,
	db *bun.DB,
) error {
	return purgeDeleted(ctx, db)
}
