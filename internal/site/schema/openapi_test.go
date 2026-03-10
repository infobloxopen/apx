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

func TestExtractOpenAPI_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte(`{{{not yaml`), 0o644)

	_, err := ExtractOpenAPI(path)
	assert.Error(t, err)
}
