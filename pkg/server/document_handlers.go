package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/go-chi/chi/v5"

	"github.com/getzep/zep/pkg/models"
)

//TODO: Move interfaces to server package

// CreateCollectionHandler godoc
//
//	@Summary		Creates a new DocumentCollection
//	@Description	If a collection with the same name already exists, an error will be returned.
//	@Tags			collection
//	@Accept			json
//	@Produce		json
//	@Param			collectionName	path		string						true	"Name of the Document Collection"
//	@Param			collection		body		models.DocumentCollection	true	"Document Collection"
//	@Success		200				{object}	string						"OK"
//	@Failure		400				{object}	APIError					"Bad Request"
//	@Failure		404				{object}	APIError					"Not Found"
//	@Failure		500				{object}	APIError					"Internal Server Error"
//	@Router			/api/v1/collection [post]
func CreateCollectionHandler(appState *models.AppState) http.HandlerFunc {
	store := appState.DocumentStore
	return func(w http.ResponseWriter, r *http.Request) {
		collectionName := strings.ToLower(chi.URLParam(r, "collectionName"))
		if collectionName == "" {
			renderError(w, errors.New("collectionName is required"), http.StatusBadRequest)
			return
		}

		var collection models.DocumentCollection
		err := json.NewDecoder(r.Body).Decode(&collection)
		if err != nil {
			renderError(w, err, http.StatusBadRequest)
			return
		}

		err = store.CreateCollection(r.Context(), &collection)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				renderError(w, err, http.StatusNotFound)
				return
			}
			renderError(w, err, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(OKResponse))
		if err != nil {
			renderError(w, err, http.StatusInternalServerError)
			return
		}
	}
}

// UpdateCollectionHandler godoc
//
//	@Summary	Updates a DocumentCollection
//	@Tags		collection
//	@Accept		json
//	@Produce	json
//	@Param		collectionName	path		string						true	"Name of the Document Collection"
//	@Param		collection		body		models.DocumentCollection	true	"Document Collection"
//	@Success	200				{object}	string						"OK"
//	@Failure	400				{object}	APIError					"Bad Request"
//	@Failure	404				{object}	APIError					"Not Found"
//	@Failure	500				{object}	APIError					"Internal Server Error"
//	@Router		/api/v1/collection/{collectionName} [patch]
func UpdateCollectionHandler(appState *models.AppState) http.HandlerFunc {
	store := appState.DocumentStore
	return func(w http.ResponseWriter, r *http.Request) {
		collectionName := strings.ToLower(chi.URLParam(r, "collectionName"))
		if collectionName == "" {
			renderError(w, errors.New("collectionName is required"), http.StatusBadRequest)
			return
		}
		var collection models.DocumentCollection
		if err := json.NewDecoder(r.Body).Decode(&collection); err != nil {
			renderError(w, err, http.StatusBadRequest)
			return
		}

		err := store.UpdateCollection(r.Context(), &collection)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				renderError(w, err, http.StatusNotFound)
				return
			}
			renderError(w, err, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("OK"))
		if err != nil {
			renderError(w, err, http.StatusInternalServerError)
			return
		}
	}
}

// DeleteCollectionHandler godoc
//
//	@Summary		Deletes a DocumentCollection
//	@Description	If a collection with the same name already exists, it will be overwritten.
//	@Tags			collection
//	@Accept			json
//	@Produce		json
//	@Param			collectionName	path		string		true	"Name of the Document Collection"
//	@Success		200				{object}	string		"OK"
//	@Failure		400				{object}	APIError	"Bad Request"
//	@Failure		404				{object}	APIError	"Not Found"
//	@Failure		500				{object}	APIError	"Internal Server Error"
//	@Router			/api/v1/collection/{collectionName} [delete]
func DeleteCollectionHandler(appState *models.AppState) http.HandlerFunc {
	store := appState.DocumentStore
	return func(w http.ResponseWriter, r *http.Request) {
		collectionName := strings.ToLower(chi.URLParam(r, "collectionName"))
		if collectionName == "" {
			renderError(w, errors.New("collectionName is required"), http.StatusBadRequest)
			return
		}

		err := store.DeleteCollection(r.Context(), collectionName)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				renderError(w, err, http.StatusNotFound)
				return
			}
			renderError(w, err, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("OK"))
		if err != nil {
			renderError(w, err, http.StatusInternalServerError)
			return
		}
	}
}

// GetCollectionListHandler godoc
//
//	@Summary		Gets a list of DocumentCollections
//	@Description	Returns a list of all DocumentCollections.
//	@Tags			collection
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		[]models.DocumentCollection	"OK"
//	@Failure		500	{object}	APIError					"Internal Server Error"
//	@Router			/api/v1/collection [get]
func GetCollectionListHandler(appState *models.AppState) http.HandlerFunc {
	store := appState.DocumentStore
	return func(w http.ResponseWriter, r *http.Request) {
		collections, err := store.GetCollectionList(r.Context())
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				renderError(w, err, http.StatusNotFound)
				return
			}
			renderError(w, err, http.StatusInternalServerError)
			return
		}

		if err := encodeJSON(w, collections); err != nil {
			renderError(w, err, http.StatusInternalServerError)
			return
		}
	}
}

// GetCollectionHandler godoc
//
//	@Summary		Gets a DocumentCollection
//	@Description	Returns a DocumentCollection if it exists.
//	@Tags			collection
//	@Accept			json
//	@Produce		json
//	@Param			collectionName	path		string						true	"Name of the Document Collection"
//	@Success		200				{object}	models.DocumentCollection	"OK"
//	@Failure		400				{object}	APIError					"Bad Request"
//	@Failure		404				{object}	APIError					"Not Found"
//	@Failure		500				{object}	APIError					"Internal Server Error"
//	@Router			/api/v1/collection/{collectionName} [get]
func GetCollectionHandler(appState *models.AppState) http.HandlerFunc {
	store := appState.DocumentStore
	return func(w http.ResponseWriter, r *http.Request) {
		collectionName := strings.ToLower(chi.URLParam(r, "collectionName"))
		if collectionName == "" {
			renderError(w, errors.New("collectionName is required"), http.StatusBadRequest)
			return
		}

		collection, err := store.GetCollection(r.Context(), collectionName)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				renderError(w, err, http.StatusNotFound)
				return
			}
			renderError(w, err, http.StatusInternalServerError)
			return
		}

		if err := encodeJSON(w, collection); err != nil {
			renderError(w, err, http.StatusInternalServerError)
			return
		}
	}
}

// CreateDocumentsHandler godoc
//
//	@Summary		Creates Multiple Documents in a DocumentCollection
//	@Description	Creates Documents in a specified DocumentCollection and returns their UUIDs.
//	@Tags			document
//	@Accept			json
//	@Produce		json
//	@Param			collectionName	path		string				true	"Name of the Document Collection"
//	@Param			documents		body		[]models.Document	true	"Array of Documents to be created"
//	@Success		200				{array}		uuid.UUID			"OK"
//	@Failure		400				{object}	APIError			"Bad Request"
//	@Failure		500				{object}	APIError			"Internal Server Error"
//	@Router			/api/v1/collection/{collectionName}/document [post]
func CreateDocumentsHandler(appState *models.AppState) http.HandlerFunc {
	store := appState.DocumentStore
	return func(w http.ResponseWriter, r *http.Request) {
		collectionName := strings.ToLower(chi.URLParam(r, "collectionName"))
		if collectionName == "" {
			renderError(w, errors.New("collectionName is required"), http.StatusBadRequest)
			return
		}

		var documents []models.DocumentInterface
		if err := json.NewDecoder(r.Body).Decode(&documents); err != nil {
			renderError(w, err, http.StatusBadRequest)
			return
		}

		uuids, err := store.CreateDocuments(r.Context(), collectionName, documents)
		if err != nil {
			renderError(w, err, http.StatusInternalServerError)
			return
		}

		if err := encodeJSON(w, uuids); err != nil {
			renderError(w, err, http.StatusInternalServerError)
			return
		}
	}
}

// UpdateDocumentHandler godoc
//
//	@Summary	Updates a Document in a DocumentCollection by UUID
//	@Tags		document
//	@Accept		json
//	@Produce	json
//	@Param		collectionName	path		string			true	"Name of the Document Collection"
//	@Param		documentUUID	path		string			true	"UUID of the Document to be updated"
//	@Param		document		body		models.Document	true	"Document to be updated"
//	@Success	200				{object}	string			"OK"
//	@Failure	400				{object}	APIError		"Bad Request"
//	@Failure	404				{object}	APIError		"Not Found"
//	@Failure	500				{object}	APIError		"Internal Server Error"
//	@Router		/api/v1/collection/{collectionName}/document/uuid/{documentUUID} [patch]
func UpdateDocumentHandler(appState *models.AppState) http.HandlerFunc {
	store := appState.DocumentStore
	return func(w http.ResponseWriter, r *http.Request) {
		collectionName := strings.ToLower(chi.URLParam(r, "collectionName"))
		documentUUID := parseUUIDFromURL(r, w, "documentUUID")

		if collectionName == "" {
			renderError(w, errors.New("collectionName is required"), http.StatusBadRequest)
			return
		}
		if documentUUID == uuid.Nil {
			renderError(w, errors.New("documentUUID is required"), http.StatusBadRequest)
			return
		}

		var document models.Document
		if err := json.NewDecoder(r.Body).Decode(&document); err != nil {
			renderError(w, err, http.StatusBadRequest)
			return
		}

		document.UUID = documentUUID
		documents := []models.DocumentInterface{&document}
		err := store.UpdateDocuments(r.Context(), collectionName, documents)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				renderError(w, err, http.StatusNotFound)
				return
			}
			renderError(w, err, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("OK"))
		if err != nil {
			renderError(w, err, http.StatusInternalServerError)
			return
		}
	}
}

// UpdateDocumentsBatchHandler godoc
//
//	@Summary		Batch Updates Documents in a DocumentCollection
//	@Description	Updates Documents in a specified DocumentCollection.
//	@Tags			document
//	@Accept			json
//	@Produce		json
//	@Param			collectionName	path		string				true	"Name of the Document Collection"
//	@Param			documents		body		[]models.Document	true	"Array of Documents to be updated"
//	@Success		200				{object}	string				"OK"
//	@Failure		400				{object}	APIError			"Bad Request"
//	@Failure		500				{object}	APIError			"Internal Server Error"
//	@Router			/api/v1/collection/{collectionName}/document/batchUpdate [patch]
func UpdateDocumentsBatchHandler(appState *models.AppState) http.HandlerFunc {
	store := appState.DocumentStore
	return func(w http.ResponseWriter, r *http.Request) {
		collectionName := strings.ToLower(chi.URLParam(r, "collectionName"))
		if collectionName == "" {
			renderError(w, errors.New("collectionName is required"), http.StatusBadRequest)
			return
		}

		var documents []models.DocumentInterface
		if err := json.NewDecoder(r.Body).Decode(&documents); err != nil {
			renderError(w, err, http.StatusBadRequest)
			return
		}

		err := store.UpdateDocuments(r.Context(), collectionName, documents)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				renderError(w, err, http.StatusNotFound)
				return
			}
			renderError(w, err, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("OK"))
		if err != nil {
			renderError(w, err, http.StatusInternalServerError)
			return
		}
	}
}

// GetDocumentHandler godoc
//
//	@Summary		Gets a Document from a DocumentCollection by UUID
//	@Description	Returns specified Document from a DocumentCollection.
//	@Tags			document
//	@Accept			json
//	@Produce		json
//	@Param			collectionName	path		string			true	"Name of the Document Collection"
//	@Param			documentUUID	path		string			true	"UUID of the Document to be updated"
//	@Success		200				{object}	models.Document	"OK"
//	@Failure		400				{object}	APIError		"Bad Request"
//	@Failure		500				{object}	APIError		"Internal Server Error"
//	@Router			/api/v1/collection/{collectionName}/document/uuid/{documentUUID} [get]
func GetDocumentHandler(appState *models.AppState) http.HandlerFunc {
	store := appState.DocumentStore
	return func(w http.ResponseWriter, r *http.Request) {
		collectionName := strings.ToLower(chi.URLParam(r, "collectionName"))
		documentUUID := parseUUIDFromURL(r, w, "documentUUID")

		if collectionName == "" {
			renderError(w, errors.New("collectionName is required"), http.StatusBadRequest)
			return
		}
		if documentUUID == uuid.Nil {
			renderError(w, errors.New("documentUUID is required"), http.StatusBadRequest)
			return
		}

		uuids := []uuid.UUID{documentUUID}
		documents, err := store.GetDocuments(
			r.Context(),
			collectionName,
			uuids,
			nil,
		)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				renderError(w, err, http.StatusNotFound)
				return
			}
			renderError(w, err, http.StatusInternalServerError)
			return
		}

		if err := encodeJSON(w, documents); err != nil {
			renderError(w, err, http.StatusInternalServerError)
			return
		}
	}
}

// GetDocumentsBatchHandler godoc
//
//	@Summary		Batch Gets Documents from a DocumentCollection
//	@Description	Returns Documents from a DocumentCollection specified by UUID or ID.
//	@Tags			document
//	@Accept			json
//	@Produce		json
//	@Param			collectionName	path		string				true	"Name of the Document Collection"
//	@Param			documentRequest	body		documentRequest		true	"UUIDs and IDs of the Documents to be fetched"
//	@Success		200				{array}		[]models.Document	"OK"
//	@Failure		400				{object}	APIError			"Bad Request"
//	@Failure		500				{object}	APIError			"Internal Server Error"
//	@Router			/api/v1/collection/{collectionName}/document/batchGet [post]
func GetDocumentsBatchHandler(appState *models.AppState) http.HandlerFunc {
	store := appState.DocumentStore
	return func(w http.ResponseWriter, r *http.Request) {
		collectionName := strings.ToLower(chi.URLParam(r, "collectionName"))
		if collectionName == "" {
			renderError(w, errors.New("collectionName is required"), http.StatusBadRequest)
			return
		}

		var docRequest documentRequest
		if err := json.NewDecoder(r.Body).Decode(&docRequest); err != nil {
			renderError(w, err, http.StatusBadRequest)
			return
		}

		documents, err := store.GetDocuments(
			r.Context(),
			collectionName,
			docRequest.UUIDs,
			docRequest.DocumentIDs,
		)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				renderError(w, err, http.StatusNotFound)
				return
			}
			renderError(w, err, http.StatusInternalServerError)
			return
		}

		if err := encodeJSON(w, documents); err != nil {
			renderError(w, err, http.StatusInternalServerError)
			return
		}
	}
}

// DeleteDocumentHandler godoc
//
//	@Summary		Delete Document from a DocumentCollection by UUID
//	@Description	Delete specified Document from a DocumentCollection.
//
//	@Tags			document
//
//	@Accept			json
//	@Produce		json
//	@Param			collectionName	path		string		true	"Name of the Document Collection"
//	@Param			documentUUID	path		string		true	"UUID of the Document to be deleted"
//	@Success		200				{object}	string		"OK"
//	@Failure		400				{object}	APIError	"Bad Request"
//	@Failure		404				{object}	APIError	"Document Not Found"
//	@Failure		500				{object}	APIError	"Internal Server Error"
//	@Router			/api/v1/collection/{collectionName}/document/uuid/{documentUUID} [delete]
func DeleteDocumentHandler(appState *models.AppState) http.HandlerFunc {
	store := appState.DocumentStore
	return func(w http.ResponseWriter, r *http.Request) {
		collectionName := strings.ToLower(chi.URLParam(r, "collectionName"))
		documentUUID := parseUUIDFromURL(r, w, "documentUUID")

		if collectionName == "" {
			renderError(w, errors.New("collectionName is required"), http.StatusBadRequest)
			return
		}

		if documentUUID == uuid.Nil {
			renderError(w, errors.New("documentUUID is required"), http.StatusBadRequest)
			return
		}

		uuids := []uuid.UUID{documentUUID}
		err := store.DeleteDocuments(r.Context(), collectionName, uuids)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				renderError(w, err, http.StatusNotFound)
				return
			}
			renderError(w, err, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("OK"))
		if err != nil {
			renderError(w, err, http.StatusInternalServerError)
			return
		}
	}
}

// DeleteDocumentsBatchHandler godoc
//
//	@Summary		Batch Deletes Documents from a DocumentCollection by UUID
//	@Description	Deletes specified Documents from a DocumentCollection.
//
//	@Tags			document
//
//	@Accept			json
//	@Produce		json
//	@Param			collectionName	path		string		true	"Name of the Document Collection"
//	@Param			documentUUIDs	body		[]uuid.UUID	true	"UUIDs of the Documents to be deleted"
//	@Success		200				{object}	string		"OK"
//	@Failure		400				{object}	APIError	"Bad Request"
//	@Failure		500				{object}	APIError	"Internal Server Error"
//	@Router			/api/v1/collection/{collectionName}/document/batchDelete [post]
func DeleteDocumentsBatchHandler(appState *models.AppState) http.HandlerFunc {
	store := appState.DocumentStore
	return func(w http.ResponseWriter, r *http.Request) {
		collectionName := strings.ToLower(chi.URLParam(r, "collectionName"))
		if collectionName == "" {
			renderError(w, errors.New("collectionName is required"), http.StatusBadRequest)
			return
		}

		var documentUUIDs []uuid.UUID
		if err := json.NewDecoder(r.Body).Decode(&documentUUIDs); err != nil {
			renderError(w, err, http.StatusBadRequest)
			return
		}

		err := store.DeleteDocuments(r.Context(), collectionName, documentUUIDs)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				renderError(w, err, http.StatusNotFound)
				return
			}
			renderError(w, err, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("OK"))
		if err != nil {
			renderError(w, err, http.StatusInternalServerError)
			return
		}
	}
}

// documentRequest is a struct for the request body of GetDocumentsBatchHandler
type documentRequest struct {
	UUIDs       []uuid.UUID `json:"uuids"`
	DocumentIDs []string    `json:"documentIDs"`
}

func parseUUIDFromURL(r *http.Request, w http.ResponseWriter, paramName string) uuid.UUID {
	uuidStr := chi.URLParam(r, paramName)
	documentUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		renderError(
			w,
			fmt.Errorf("unable to parse document UUID: %w", err),
			http.StatusBadRequest,
		)
		return uuid.Nil
	}
	return documentUUID
}
