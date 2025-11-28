package storefront

import "github.com/jackc/pgx/v5/pgtype"

// Helper function to parse UUIDs for test data
func mustParseUUID(s string) pgtype.UUID {
	var uuid pgtype.UUID
	if err := uuid.Scan(s); err != nil {
		panic(err)
	}
	return uuid
}

// Helper function to parse Numeric values for test data
func mustParseNumeric(s string) pgtype.Numeric {
	var num pgtype.Numeric
	if err := num.Scan(s); err != nil {
		panic(err)
	}
	return num
}
