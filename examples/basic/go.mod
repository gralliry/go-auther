module auther/examples/basic

go 1.26

require (
	auther v0.0.0
	auther/adapters/memory v0.0.0
)

replace (
	auther => ../../
	auther/adapters/memory => ../../adapters/memory
)
