// Package fixtures holds synthetic contract fixtures for cross-repo decoder tests.
package fixtures

import _ "embed"

//go:embed start_preparation_fixtures.json
var StartPreparationJSON string
