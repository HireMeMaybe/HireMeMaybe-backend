package model

// MigrateAble is array of model instance, use for migrating database
var MigrateAble []interface{}

func init() {
	MigrateAble = append(
		MigrateAble,
		&User{},
		&CPSKUser{},
		&Company{},
		&File{},
		&JobPost{},
		&Application{},
		&AplicationAnswer{},
	)
}
