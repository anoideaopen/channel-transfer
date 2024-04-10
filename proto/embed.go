package proto

import (
	_ "embed"
)

//go:embed service.swagger.json
var SwaggerJSON []byte
