package model

// The File struct represents a file with an ID, content stored as bytes, and an extension.
// @property {int} ID - The `ID` property in the `File` struct is an integer field that is marked as
// the primary key in the database using the `gorm:"primaryKey"` tag. This means that the `ID` field
// uniquely identifies each record in the database table for the `File` struct.
// @property {[]byte} Content - The `Content` property in the `File` struct represents the actual data
// of the file, stored as a byte slice. This property will hold the binary data of the file, such as
// text, images, or any other type of file content.
// @property {string} Extension - The `Extension` property in the `File` struct represents the file
// extension of the file. This could be something like ".txt", ".jpg", ".pdf", etc. It is used to
// identify the type of file and determine how it should be handled or processed.
type File struct {
	ID        int `gorm:"primaryKey"`
	Content   []byte
	Extension string
}
