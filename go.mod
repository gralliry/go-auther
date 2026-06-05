module github.com/gralliry/go-auther

go 1.26

require github.com/gralliry/go-auther/adapter/empty v0.0.0

require github.com/bwmarrin/snowflake v0.3.0

replace github.com/gralliry/go-auther/adapter/empty => ./adapter/driver/empty
