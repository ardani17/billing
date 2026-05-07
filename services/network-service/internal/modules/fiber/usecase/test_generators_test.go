package usecase

import "pgregory.net/rapid"

func uuidGen() *rapid.Generator[string] {
	return rapid.StringMatching(
		`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`,
	)
}
