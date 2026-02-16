package remote_http

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/_/mcp"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/sku_json_fmt"
	"code.linenisgreat.com/dodder/go/src/lima/type_blobs"
	"code.linenisgreat.com/dodder/go/src/november/queries"
)

func (server *Server) handleMCP(request Request) (response Response) {
	response.Headers().Set("Content-Type", "application/json")

	var mcpRequest mcp.Request

	decoder := json.NewDecoder(request.Body)

	if err := decoder.Decode(&mcpRequest); err != nil {
		response.MCPError(
			http.StatusBadRequest,
			nil, -32700, "Parse error", nil,
		)

		return response
	}

	if mcpRequest.JSONRPC != "2.0" {
		response.MCPError(
			http.StatusBadRequest,
			mcpRequest.ID,
			-32600,
			"Invalid Request",
			nil,
		)

		return response
	}

	mcpResponse := mcp.Response{
		JSONRPC: "2.0",
		ID:      mcpRequest.ID,
	}

	switch mcpRequest.Method {
	case "initialize":
		mcpResponse.Result = mcp.InitializeResponse{
			ProtocolVersion: "2024-11-05",
			Capabilities: mcp.ServerCapabilities{
				Resources: &mcp.ResourcesCapability{
					Subscribe:   false,
					ListChanged: true,
				},
			},
			ServerInfo: mcp.ServerInfo{
				Name:    "dodder",
				Version: "1.0.0",
			},
		}

	case "resources/list":
		resources := server.getMCPResources()
		mcpResponse.Result = mcp.ResourcesListResult{Resources: resources}

	case "resources/read":
		var params mcp.ResourcesReadParams

		if mcpRequest.Params != nil {
			paramsBytes, err := json.Marshal(mcpRequest.Params)
			if err != nil {
				mcpResponse.Error = &mcp.Error{
					Code:    -32602,
					Message: "Invalid params",
				}
				break
			}
			if err := json.Unmarshal(paramsBytes, &params); err != nil {
				mcpResponse.Error = &mcp.Error{
					Code:    -32602,
					Message: "Invalid params",
				}
				break
			}
		}

		contents, err := server.readMCPResource(params.URI)
		if err != nil {
			mcpResponse.Error = &mcp.Error{
				Code:    -32602,
				Message: fmt.Sprintf("Failed to read resource: %v", err),
			}
		} else {
			mcpResponse.Result = mcp.ResourcesReadResult{Contents: contents}
		}

	default:
		mcpResponse.Error = &mcp.Error{
			Code:    -32601,
			Message: "Method not found",
		}
	}

	responseBytes, err := json.Marshal(mcpResponse)
	if err != nil {
		response.MCPError(
			http.StatusInternalServerError,
			mcpRequest.ID,
			-32603,
			"Internal error",
			nil,
		)

		return response
	}

	response.StatusCode = http.StatusOK
	response.Body = ohio.NopCloser(bytes.NewReader(responseBytes))

	return response
}

func (server *Server) getMCPResources() []mcp.Resource {
	resources := []mcp.Resource{
		{
			URI:         "dodder:///types",
			Name:        "Objects",
			Description: "list of all available object types",
			MimeType:    "application/json",
		},
		{
			URI:         "dodder:///word-index",
			Name:        "Objects that contain words",
			Description: "A list of indexes corresponding to words, where the objects in the index contain the word somewhere in their blob or metadata",
			MimeType:    "application/json",
		},
		// {
		// 	URI:         "dodder:///tags",
		// 	Name:        "Objects",
		// 	Description: "list of all available tags",
		// 	MimeType:    "application/json",
		// },
	}

	return resources
}

func (server *Server) readMCPResource(
	uriString string,
) ([]mcp.ResourceContent, error) {
	uri, err := url.ParseRequestURI(uriString)
	if err != nil {
		return nil, err
	}

	if uri.Scheme != "dodder" {
		err = errors.BadRequestf(
			"expected scheme %q but got %q",
			"dodder",
			uri.Scheme,
		)
		return nil, err
	}

	if uri.Host != "" {
		err = errors.BadRequestf(
			"expected empty host but got %q",
			uri.Host,
		)
		return nil, err
	}

	if strings.HasPrefix(uri.Path, "/objects") {
		return server.readMCPResourceObjects(uri)
	} else if strings.HasPrefix(uri.Path, "/types") {
		return server.readMCPResourceTypes(uri)
	} else if strings.HasPrefix(uri.Path, "/blobs") {
		return server.readMCPResourceBlobs(uri)
	} else {
		return nil, errors.BadRequestf("resource not found: %q", uriString)
	}
}

func (server *Server) readMCPResourceTypes(
	uri *url.URL,
) ([]mcp.ResourceContent, error) {
	repo := server.Repo

	var queryGroup *queries.Query

	{
		var err error

		if queryGroup, err = repo.MakeExternalQueryGroup(
			queries.BuilderOptions(),
			sku.ExternalQueryOptions{},
			":t",
		); err != nil {
			return nil, errors.Wrap(err)
		}
	}

	results := make([]mcp.ResourceContent, 0)

	var lock sync.Mutex

	if err := repo.GetStore().QueryTransacted(
		queryGroup,
		func(object *sku.Transacted) (err error) {
			lock.Lock()
			defer lock.Unlock()

			objectResources, err := server.readMCPResourceObject(object)
			if err != nil {
				err = errors.Wrap(err)
				return err
			}

			results = append(results, objectResources...)

			return err
		},
	); err != nil {
		return nil, errors.Wrap(err)
	}

	return results, nil
}

func (server *Server) readMCPResourceObjects(
	uri *url.URL,
) ([]mcp.ResourceContent, error) {
	repo := server.Repo

	objectIdString := strings.TrimPrefix(
		strings.TrimPrefix(uri.Path, "/"),
		"objects",
	)

	if len(objectIdString) > 1 {
		var objectId ids.ObjectId

		if err := objectId.Set(objectIdString); err != nil {
			return nil, errors.Wrap(err)
		}

		var object *sku.Transacted

		{
			var err error

			if object, err = repo.GetStore().ReadOneObjectId(
				&objectId,
			); err != nil {
				return nil, errors.Wrap(err)
			}
		}

		return server.readMCPResourceObject(object)
	}

	var queryGroup *queries.Query

	{
		var err error

		if queryGroup, err = repo.MakeExternalQueryGroup(
			queries.BuilderOptions(
			// query.BuilderOptionWorkspace{Env: repo.GetEnvWorkspace()},
			),
			sku.ExternalQueryOptions{},
			":t",
		); err != nil {
			return nil, errors.Wrap(err)
		}
	}

	var list *sku.HeapTransacted

	{
		var err error

		if list, err = repo.MakeInventoryList(queryGroup); err != nil {
			return nil, errors.Wrap(err)
		}
	}

	results := make([]mcp.ResourceContent, 0, list.Len())

	for object := range list.All() {
		objectResources, err := server.readMCPResourceObject(object)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		results = append(results, objectResources...)
	}

	return results, nil
}

func (server *Server) readMCPResourceObject(
	object *sku.Transacted,
) ([]mcp.ResourceContent, error) {
	repo := server.Repo

	var jsonRep sku_json_fmt.MCP

	if err := jsonRep.FromTransacted(
		object,
		nil,
	); err != nil {
		return nil, errors.Wrap(err)
	}

	var typeBlob type_blobs.Blob

	{
		var typeObject *sku.Transacted

		{
			var err error

			if typeObject, err = repo.GetStore().ReadTypeObject(
				object.GetMetadata().GetTypeLock(),
			); err != nil {
				if errors.IsErrNotFound(err) {
					err = nil
					goto SKIP_TYPE_BLOB
				} else {
					return nil, errors.Wrap(err)
				}
			}

		}

		{
			var err error
			var repool interfaces.FuncRepool

			if typeBlob, repool, _, err = repo.GetTypedBlobStore().Type.ParseTypedBlob(
				typeObject.GetType(),
				typeObject.GetBlobDigest(),
			); err != nil {
				return nil, errors.Wrap(err)
			}

			defer repool()
		}
	}

SKIP_TYPE_BLOB:
	var mimeType string

	if typeBlob != nil {
		mimeType = mime.TypeByExtension(typeBlob.GetFileExtension())
	}

	if mimeType == "" {
		jsonRep.RelatedURIs = append(
			jsonRep.RelatedURIs,
			fmt.Sprintf("dodder:///blobs/%s", jsonRep.BlobId),
		)
	} else {
		jsonRep.RelatedURIs = append(
			jsonRep.RelatedURIs,
			fmt.Sprintf("dodder:///blobs/%s/%s", jsonRep.BlobId, mimeType),
		)
	}

	var sb strings.Builder

	encoder := json.NewEncoder(&sb)

	if err := encoder.Encode(jsonRep); err != nil {
		return nil, errors.Wrap(err)
	}

	return []mcp.ResourceContent{{
		URI:      jsonRep.URI,
		MimeType: "application/json",
		Text:     sb.String(),
	}}, nil
}

func (server *Server) readMCPResourceBlobs(
	uri *url.URL,
) ([]mcp.ResourceContent, error) {
	pathComponents := strings.Split(strings.TrimPrefix(uri.Path, "/blobs"), "/")

	if len(pathComponents) == 0 {
		return nil, errors.BadRequestf("blob digest not provided")
	}

	blobDigestString := pathComponents[0]

	digest, repool := markl.FormatHashSha256.GetMarklIdForString(blobDigestString)

	defer repool()

	readCloser, err := server.Repo.GetEnvRepo().GetDefaultBlobStore().MakeBlobReader(
		digest,
	)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	var buffer bytes.Buffer

	if _, err := io.Copy(&buffer, readCloser); err != nil {
		return nil, errors.Wrap(err)
	}

	var mimeType string

	if len(pathComponents) > 1 {
		mimeType = pathComponents[1]
	}

	return []mcp.ResourceContent{{
		URI:      uri.String(),
		MimeType: mimeType,
		Blob:     base64.StdEncoding.EncodeToString(buffer.Bytes()),
	}}, nil
}
