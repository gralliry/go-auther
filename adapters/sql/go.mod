module github.com/gralliry/go-auther/adapters/sql

go 1.26

require (
	github.com/gralliry/go-auther v0.0.0
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.44 // indirect
	golang.org/x/text v0.20.0 // indirect
)

replace github.com/gralliry/go-auther => ../..
