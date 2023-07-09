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
func putCollection(
	ctx context.Context,
	db *bun.DB,
	collectionID string,
	metadata map[string]interface{},
	isPrivileged bool,
) (*models.Collection, error) {
	if collectionID == "" {
		return nil, NewDocumentStorageError("collectionID cannot be empty", nil)
	}

	// We're not going to run this in a transaction as we don't necessarily
	// need to roll back the session creation if the message metadata upsert fails.
	collection := PgCollection{CollectionID: collectionID}
	_, err := db.NewInsert().
		Model(&collection).
		Column("collection_id").
		On("CONFLICT (collection_id) DO UPDATE"). // we'll do an upsert
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, NewDocumentStorageError("failed to put collection", err)
	}

	// If the collection is deleted, return an error
	if !collection.DeletedAt.IsZero() {
		return nil, NewDocumentStorageError(fmt.Sprintf("collection %s is deleted", collectionID), nil)
	}

	// remove the top-level `system` key from the metadata if the caller is not privileged
	if !isPrivileged {
		delete(metadata, "system")
	}

	// update the session metadata and return the session
	return putCollectionMetadata(ctx, db, collectionID, metadata)
}

// putCollectionMetadata updates the metadata for a collection. The metadata map is merged
// with the existing metadata map, creating keys and values if they don't exist.
func putCollectionMetadata(ctx context.Context,
	db *bun.DB,
	collectionID string,
	metadata map[string]interface{}) (*models.Collection, error) {
	// Acquire a lock for this CollectionID. This is to prevent concurrent updates
	// to the collection metadata.
	lockID, err := acquireAdvisoryLock(ctx, db, collectionID)
	if err != nil {
		return nil, NewDocumentStorageError("failed to acquire advisory lock", err)
	}
	defer func(ctx context.Context, db bun.IDB, lockID uint64) {
		err := releaseAdvisoryLock(ctx, db, lockID)
		if err != nil {
			log.Error(ctx, "failed to release advisory lock", err)
		}
	}(ctx, db, lockID)

	dbCollection := &PgCollection{}
	err = db.NewSelect().
		Model(dbCollection).
		Where("collection_id = ?", collectionID).
		Scan(ctx)
	if err != nil {
		return nil, NewDocumentStorageError("failed to get session", err)
	}

	// merge the existing metadata with the new metadata
	dbMetadata := dbCollection.Metadata
	if err := mergo.Merge(&dbMetadata, metadata, mergo.WithOverride); err != nil {
		return nil, NewDocumentStorageError("failed to merge metadata", err)
	}

	// put the collection metadata, returning the updated collection
	_, err = db.NewUpdate().
		Model(dbCollection).
		Set("metadata = ?", dbMetadata).
		Where("collection_id = ?", collectionID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, NewDocumentStorageError("failed to update session metadata", err)
	}

	collection := &models.Collection{}
	err = copier.Copy(collection, dbCollection)
	if err != nil {
		return nil, NewDocumentStorageError("Unable to copy session", err)
	}

	return collection, nil
}

// getCollection retrieves a document collection from the document store.
func getCollection(
	ctx context.Context,
	db *bun.DB,
	collectionID string,
) (*models.Collection, error) {
	collection := PgCollection{}
	err := db.NewSelect().Model(&collection).Where("collection_id = ?", collectionID).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, NewDocumentStorageError("failed to get collection", err)
	}

	retCollection := models.Collection{}
	err = copier.Copy(&retCollection, &collection)
	if err != nil {
		return nil, NewDocumentStorageError("failed to copy collection", err)
	}

	return &retCollection, nil
}
