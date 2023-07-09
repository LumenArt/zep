package documentstore

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/getzep/zep/pkg/llms"

	"github.com/getzep/zep/pkg/models"

	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/uptrace/bun"
)

type PgCollection struct {
	bun.BaseModel `bun:"table:collection,alias:c"`

	UUID         uuid.UUID              `bun:",pk,type:uuid,default:gen_random_uuid()"`
	CollectionID string                 `bun:",unique,notnull"`
	CreatedAt    time.Time              `bun:"type:timestamptz,nullzero,notnull,default:current_timestamp"`
	UpdatedAt    time.Time              `bun:"type:timestamptz,nullzero,notnull,default:current_timestamp"`
	DeletedAt    time.Time              `bun:"type:timestamptz,soft_delete,nullzero"`
	Role         string                 `bun:",notnull"` // future use for RBAC
	Metadata     map[string]interface{} `bun:"type:jsonb,nullzero,json_use_number"`
}

// BeforeCreateTable is a dummy method to ensure uniform interface across all table models - used in table creation iterator
func (s *PgCollection) BeforeCreateTable(
	_ context.Context,
	_ *bun.CreateTableQuery,
) error {
	return nil
}

type PgDocument struct {
	bun.BaseModel `bun:"table:document,alias:d"`

	UUID         uuid.UUID               `bun:",pk,type:uuid,default:gen_random_uuid()"`
	DocumentID   string                  `bun:",unique,notnull"`
	CollectionID string                  `bun:",notnull"`
	CreatedAt    time.Time               `bun:"type:timestamptz,nullzero,notnull,default:current_timestamp"`
	UpdatedAt    time.Time               `bun:"type:timestamptz,nullzero,notnull,default:current_timestamp"`
	DeletedAt    time.Time               `bun:"type:timestamptz,soft_delete,nullzero"`
	Role         string                  `bun:",notnull"` // future use for RBAC
	Summary      *PgDocumentSummaryStore `bun:"rel:has-one,join:document_uuid=uuid,on_delete:cascade"`
	Metadata     map[string]interface{}  `bun:"type:jsonb,nullzero,json_use_number"`
	Collection   *PgCollection           `bun:"rel:belongs-to,join:collection_id=collection_id,on_delete:cascade"`
}

// BeforeCreateTable is a dummy method to ensure uniform interface across all table models - used in table creation iterator
func (s *PgDocument) BeforeCreateTable(
	_ context.Context,
	_ *bun.CreateTableQuery,
) error {
	return nil
}

type PgDocumentChunk struct {
	bun.BaseModel `bun:"table:documentchunks,alias:c"`

	UUID         uuid.UUID              `bun:",pk,type:uuid,default:gen_random_uuid()"`
	Sequence     float32                `bun:"type:float4,notnull"` // sequence is used to order chunks, starting at 0. Not used for now.
	CreatedAt    time.Time              `bun:"type:timestamptz,notnull,default:current_timestamp"`
	UpdatedAt    time.Time              `bun:"type:timestamptz,nullzero,default:current_timestamp"`
	DeletedAt    time.Time              `bun:"type:timestamptz,soft_delete,nullzero"`
	DocumentID   string                 `bun:",notnull"`
	CollectionID string                 `bun:",notnull"`
	Content      string                 `bun:",notnull"`                            // contents of the chunk
	Metadata     map[string]interface{} `bun:"type:jsonb,nullzero,json_use_number"` // future use
	Document     *PgDocument            `bun:"rel:belongs-to,join:document_id=document_id,on_delete:cascade"`
}

func (s *PgDocumentChunk) BeforeCreateTable(
	_ context.Context,
	_ *bun.CreateTableQuery,
) error {
	return nil
}

// PgDocumentVectorStore stores the embeddings for documents.
// TODO: Vector dims from config
type PgDocumentVectorStore struct {
	bun.BaseModel `bun:"table:message_embedding,alias:dvs"`

	UUID          uuid.UUID        `bun:",pk,type:uuid,default:gen_random_uuid()"`
	CreatedAt     time.Time        `bun:"type:timestamptz,notnull,default:current_timestamp"`
	UpdatedAt     time.Time        `bun:"type:timestamptz,nullzero,default:current_timestamp"`
	DeletedAt     time.Time        `bun:"type:timestamptz,soft_delete,nullzero"`
	CollectionID  string           `bun:",notnull"`
	DocumentID    string           `bun:",notnull"`
	ChunkUUID     uuid.UUID        `bun:"type:uuid,notnull,unique"`
	Embedding     pgvector.Vector  `bun:"type:vector(1536)"`
	IsEmbedded    bool             `bun:"type:bool,notnull,default:false"`
	DocumentChunk *PgDocumentChunk `bun:"rel:belongs-to,join:chunk_uuid=uuid,on_delete:cascade"`
}

func (s *PgDocumentVectorStore) BeforeCreateTable(
	_ context.Context,
	_ *bun.CreateTableQuery,
) error {
	return nil
}

type PgDocumentSummaryStore struct {
	bun.BaseModel `bun:"table:summary,alias:dss"`

	UUID         uuid.UUID              `bun:",pk,type:uuid,default:gen_random_uuid()"`
	CreatedAt    time.Time              `bun:"type:timestamptz,notnull,default:current_timestamp"`
	UpdatedAt    time.Time              `bun:"type:timestamptz,nullzero,default:current_timestamp"`
	DeletedAt    time.Time              `bun:"type:timestamptz,soft_delete,nullzero"`
	CollectionID string                 `bun:",notnull"`
	DocumentID   string                 `bun:",notnull"`
	Content      string                 `bun:",nullzero"` // allow null as we might want to use Metadata without a summary
	Metadata     map[string]interface{} `bun:"type:jsonb,nullzero,json_use_number"`
	Collection   *PgCollection          `bun:"rel:belongs-to,join:collection_id=collection_id,on_delete:cascade"`
	Document     *PgDocument            `bun:"rel:belongs-to,join:document_id=document_id,on_delete:cascade"`
}

func (s *PgDocumentSummaryStore) BeforeCreateTable(
	_ context.Context,
	_ *bun.CreateTableQuery,
) error {
	return nil
}

// Create collection_id indexes after table creation
var _ bun.AfterCreateTableHook = (*PgCollection)(nil)
var _ bun.AfterCreateTableHook = (*PgDocument)(nil)
var _ bun.AfterCreateTableHook = (*PgDocumentChunk)(nil)
var _ bun.AfterCreateTableHook = (*PgDocumentVectorStore)(nil)
var _ bun.AfterCreateTableHook = (*PgDocumentSummaryStore)(nil)

func (*PgCollection) AfterCreateTable(
	ctx context.Context,
	query *bun.CreateTableQuery,
) error {
	_, err := query.DB().NewCreateIndex().
		Model((*PgCollection)(nil)).
		Index("collection_id_idx").
		Column("collection_id").
		Exec(ctx)
	return err
}

func (*PgDocument) AfterCreateTable(
	ctx context.Context,
	query *bun.CreateTableQuery,
) error {
	_, err := query.DB().NewCreateIndex().
		Model((*PgDocument)(nil)).
		Index("document_id_idx").
		Column("document_id").
		Exec(ctx)
	return err
}

func (*PgDocumentChunk) AfterCreateTable(
	ctx context.Context,
	query *bun.CreateTableQuery,
) error {
	_, err := query.DB().NewCreateIndex().
		Model((*PgDocumentChunk)(nil)).
		Index("documentchunk_id_idx").
		Column("documentchunk_id").
		Exec(ctx)
	return err
}

func (*PgDocumentVectorStore) AfterCreateTable(
	ctx context.Context,
	query *bun.CreateTableQuery,
) error {
	_, err := query.DB().NewCreateIndex().
		Model((*PgDocumentVectorStore)(nil)).
		Index("doc_vec_store_id_idx").
		Column("document_id"). // TODO: index on document_id or collection_id or both?
		Exec(ctx)
	return err
}

func (*PgDocumentSummaryStore) AfterCreateTable(
	ctx context.Context,
	query *bun.CreateTableQuery,
) error {
	_, err := query.DB().NewCreateIndex().
		Model((*PgDocumentSummaryStore)(nil)).
		Index("sumstore_collection_id_idx").
		Column("document_id"). // TODO: index on document_id or collection_id or both?
		Exec(ctx)
	return err
}

var tableList = []bun.BeforeCreateTableHook{
	&PgDocumentVectorStore{},
	&PgDocumentSummaryStore{},
	&PgDocumentChunk{},
	&PgDocument{},
	&PgCollection{},
}

// initPostgresSetup creates the db schema if it does not exist.
func initPostgresSetup(
	ctx context.Context,
	appState *models.AppState,
	db *bun.DB,
) error {
	_, err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("error creating pgvector extension: %w", err)
	}

	// iterate through tableList in reverse order to create tables with foreign keys first
	for i := len(tableList) - 1; i >= 0; i-- {
		schema := tableList[i]
		_, err := db.NewCreateTable().
			Model(schema).
			IfNotExists().
			WithForeignKeys().
			Exec(ctx)
		if err != nil {
			// bun still trying to create indexes despite IfNotExists flag
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			return fmt.Errorf("error creating table for doc schema %T: %w", schema, err)
		}
	}

	model, err := llms.GetMessageEmbeddingModel(appState)
	if err != nil {
		return fmt.Errorf("error getting message embedding model: %w", err)
	}
	if model.Dimensions != 1536 {
		err := migrateDocumentEmbeddingDims(ctx, db, model.Dimensions)
		if err != nil {
			return fmt.Errorf("error migrating message embedding dimensions: %w", err)
		}
	}

	return nil
}

// TODO: refactor
func migrateDocumentEmbeddingDims(
	ctx context.Context,
	db *bun.DB,
	dimensions int,
) error {
	_, err := db.NewDropColumn().Model((*PgDocumentVectorStore)(nil)).Column("embedding").Exec(ctx)
	if err != nil {
		return fmt.Errorf("error dropping column embedding: %w", err)
	}
	_, err = db.NewAddColumn().
		Model((*PgDocumentVectorStore)(nil)).
		ColumnExpr(fmt.Sprintf("embedding vector(%d)", dimensions)).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("error adding column Embedding: %w", err)
	}

	return nil
}

// InitPostgresConn creates a new bun.DB connection to a postgres database using the provided DSN.
// The connection is configured to pool connections based on the number of PROCs available.
func InitPostgresConn(dsn string) *bun.DB {
	maxOpenConns := 4 * runtime.GOMAXPROCS(0)
	if dsn == "" {
		log.Fatal("dsn may not be empty")
	}
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	sqldb.SetMaxOpenConns(maxOpenConns)
	sqldb.SetMaxIdleConns(maxOpenConns)
	db := bun.NewDB(sqldb, pgdialect.New())
	return db
}
