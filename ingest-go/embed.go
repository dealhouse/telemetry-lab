package assets

import _ "embed"

//go:embed internal/db/schema.sql
var SchemaSQL string

//go:embed api/openapi.yaml
var OpenAPIYAML string
