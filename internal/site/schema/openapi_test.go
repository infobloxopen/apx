package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractOpenAPI_BasicYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.yaml")
	os.WriteFile(path, []byte(`openapi: "3.0.3"
info:
  title: Pet Store
  version: "1.0.0"
  description: A sample pet store API
paths:
  /pets:
    get:
      summary: List all pets
      operationId: listPets
      parameters:
        - name: limit
          in: query
      responses:
        "200":
          description: A list of pets
    post:
      summary: Create a pet
      responses:
        "201":
          description: Created
  /pets/{petId}:
    get:
      summary: Get a pet by ID
      parameters:
        - name: petId
          in: path
      responses:
        "200":
          description: A pet
        "404":
          description: Not found
components:
  schemas:
    Pet:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: integer
          format: int64
          description: Pet unique identifier
        name:
          type: string
          description: Pet name
        tag:
          type: string
`), 0o644)

	spec, err := ExtractOpenAPI(path)
	require.NoError(t, err)

	assert.Equal(t, "Pet Store", spec.Title)
	assert.Equal(t, "1.0.0", spec.Version)
	assert.Equal(t, "A sample pet store API", spec.Description)

	// Paths.
	require.Len(t, spec.Paths, 2)
	assert.Equal(t, "/pets", spec.Paths[0].Path)
	require.Len(t, spec.Paths[0].Operations, 2)
	assert.Equal(t, "GET", spec.Paths[0].Operations[0].Method)
	assert.Equal(t, "List all pets", spec.Paths[0].Operations[0].Summary)
	assert.Equal(t, "listPets", spec.Paths[0].Operations[0].OperationID)
	assert.Contains(t, spec.Paths[0].Operations[0].Parameters, "query: limit")
	assert.Equal(t, "POST", spec.Paths[0].Operations[1].Method)

	assert.Equal(t, "/pets/{petId}", spec.Paths[1].Path)
	require.Len(t, spec.Paths[1].Operations, 1)
	assert.Contains(t, spec.Paths[1].Operations[0].Parameters, "path: petId")
	assert.Contains(t, spec.Paths[1].Operations[0].Responses, "200: A pet")
	assert.Contains(t, spec.Paths[1].Operations[0].Responses, "404: Not found")

	// Schemas.
	require.Len(t, spec.Schemas, 1)
	assert.Equal(t, "Pet", spec.Schemas[0].Name)
	assert.Equal(t, "object", spec.Schemas[0].Type)
	require.Len(t, spec.Schemas[0].Properties, 3)

	idProp := spec.Schemas[0].Properties[0]
	assert.Equal(t, "id", idProp.Name)
	assert.Equal(t, "int64", idProp.Type)
	assert.True(t, idProp.Required)
	assert.Equal(t, "Pet unique identifier", idProp.Description)

	nameProp := spec.Schemas[0].Properties[1]
	assert.Equal(t, "name", nameProp.Name)
	assert.True(t, nameProp.Required)

	tagProp := spec.Schemas[0].Properties[2]
	assert.Equal(t, "tag", tagProp.Name)
	assert.False(t, tagProp.Required)
}

func TestExtractOpenAPI_Swagger2(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.yaml")
	os.WriteFile(path, []byte(`swagger: "2.0"
info:
  title: Legacy API
  version: "2.0.0"
paths:
  /users:
    get:
      summary: List users
      responses:
        "200":
          description: OK
definitions:
  User:
    type: object
    properties:
      id:
        type: string
      email:
        type: string
`), 0o644)

	spec, err := ExtractOpenAPI(path)
	require.NoError(t, err)

	assert.Equal(t, "Legacy API", spec.Title)

	// Swagger 2.0 definitions should map to schemas.
	require.Len(t, spec.Schemas, 1)
	assert.Equal(t, "User", spec.Schemas[0].Name)
	require.Len(t, spec.Schemas[0].Properties, 2)
}

func TestExtractOpenAPI_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.json")
	os.WriteFile(path, []byte(`{
		"openapi": "3.0.0",
		"info": {"title": "JSON API", "version": "1.0"},
		"paths": {
			"/items": {
				"get": {
					"summary": "List items",
					"responses": {"200": {"description": "OK"}}
				}
			}
		}
	}`), 0o644)

	spec, err := ExtractOpenAPI(path)
	require.NoError(t, err)

	assert.Equal(t, "JSON API", spec.Title)
	require.Len(t, spec.Paths, 1)
	assert.Equal(t, "/items", spec.Paths[0].Path)
}

func TestExtractOpenAPI_RefTypes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.yaml")
	os.WriteFile(path, []byte(`openapi: "3.0.3"
info:
  title: Ref Test
  version: "1.0"
components:
  schemas:
    Order:
      type: object
      properties:
        items:
          type: array
          items:
            $ref: "#/components/schemas/LineItem"
        customer:
          $ref: "#/components/schemas/Customer"
    LineItem:
      type: object
      properties:
        name:
          type: string
    Customer:
      type: object
      properties:
        email:
          type: string
`), 0o644)

	spec, err := ExtractOpenAPI(path)
	require.NoError(t, err)

	require.Len(t, spec.Schemas, 3)
	// Sorted alphabetically: Customer(0), LineItem(1), Order(2).
	order := spec.Schemas[2]
	assert.Equal(t, "Order", order.Name)
	require.Len(t, order.Properties, 2)
	assert.Equal(t, "customer", order.Properties[0].Name)
	assert.Equal(t, "Customer", order.Properties[0].Type)
	assert.Equal(t, "items", order.Properties[1].Name)
	assert.Equal(t, "array<LineItem>", order.Properties[1].Type)
}

func TestExtractOpenAPI_RequestResponseBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.yaml")
	os.WriteFile(path, []byte(`openapi: "3.0.3"
info:
  title: Body Test
  version: "1.0"
paths:
  /users:
    get:
      summary: List users
      responses:
        "200":
          description: A list of users
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/User"
    post:
      summary: Create a user
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/CreateUserRequest"
      responses:
        "201":
          description: Created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
  /users/{id}:
    put:
      summary: Update a user
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/UpdateUserRequest"
      responses:
        "200":
          description: Updated
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
    delete:
      summary: Delete a user
      responses:
        "204":
          description: Deleted
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
    CreateUserRequest:
      type: object
      properties:
        name:
          type: string
    UpdateUserRequest:
      type: object
      properties:
        name:
          type: string
`), 0o644)

	spec, err := ExtractOpenAPI(path)
	require.NoError(t, err)

	require.Len(t, spec.Paths, 2)

	// GET /users — no request body, response body is array<User>.
	getOp := spec.Paths[0].Operations[0]
	assert.Equal(t, "GET", getOp.Method)
	assert.Empty(t, getOp.RequestBody)
	assert.Equal(t, "array<User>", getOp.ResponseBody)

	// POST /users — request body is CreateUserRequest, response body is User.
	postOp := spec.Paths[0].Operations[1]
	assert.Equal(t, "POST", postOp.Method)
	assert.Equal(t, "CreateUserRequest", postOp.RequestBody)
	assert.Equal(t, "User", postOp.ResponseBody)

	// PUT /users/{id} — request body is UpdateUserRequest, response body is User.
	putOp := spec.Paths[1].Operations[0]
	assert.Equal(t, "PUT", putOp.Method)
	assert.Equal(t, "UpdateUserRequest", putOp.RequestBody)
	assert.Equal(t, "User", putOp.ResponseBody)

	// DELETE /users/{id} — no request body, no response body (204 has no content).
	deleteOp := spec.Paths[1].Operations[1]
	assert.Equal(t, "DELETE", deleteOp.Method)
	assert.Empty(t, deleteOp.RequestBody)
	assert.Empty(t, deleteOp.ResponseBody)
}

func TestExtractOpenAPI_Swagger2Body(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.yaml")
	os.WriteFile(path, []byte(`swagger: "2.0"
info:
  title: Swagger Body Test
  version: "1.0"
paths:
  /items:
    post:
      summary: Create item
      parameters:
        - name: body
          in: body
          schema:
            $ref: "#/definitions/CreateItemRequest"
      responses:
        "200":
          description: Created
          schema:
            $ref: "#/definitions/Item"
    get:
      summary: List items
      responses:
        "200":
          description: OK
          schema:
            type: array
            items:
              $ref: "#/definitions/Item"
definitions:
  Item:
    type: object
    properties:
      id:
        type: string
  CreateItemRequest:
    type: object
    properties:
      name:
        type: string
`), 0o644)

	spec, err := ExtractOpenAPI(path)
	require.NoError(t, err)

	require.Len(t, spec.Paths, 1)

	// GET /items — no request body, response body is array<Item>.
	getOp := spec.Paths[0].Operations[0]
	assert.Equal(t, "GET", getOp.Method)
	assert.Empty(t, getOp.RequestBody)
	assert.Equal(t, "array<Item>", getOp.ResponseBody)

	// POST /items — request body is CreateItemRequest, response body is Item.
	postOp := spec.Paths[0].Operations[1]
	assert.Equal(t, "POST", postOp.Method)
	assert.Equal(t, "CreateItemRequest", postOp.RequestBody)
	assert.Equal(t, "Item", postOp.ResponseBody)
}

func TestExtractOpenAPI_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte(`{{{not yaml`), 0o644)

	_, err := ExtractOpenAPI(path)
	assert.Error(t, err)
}
