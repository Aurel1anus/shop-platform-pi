package migrations

import _ "embed"

//go:embed 001_schema.sql
var Schema string

//go:embed 002_demo_seed.sql
var Seed string
