package model

var MigrateAble []interface{}

func init() {
	MigrateAble = append(
		MigrateAble,
		&User{},
		&CPSKUser{},
		&Company{},
		&File{},
	)
}
