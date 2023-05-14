module github.com/Dutchy-/spacetrader-go/main

go 1.19

replace github.com/Dutchy-/spacetrader-go/client v0.0.0 => ../client

require (
	github.com/Dutchy-/spacetrader-go/client v0.0.0
	github.com/juliangruber/go-intersect v1.1.0
	golang.org/x/time v0.3.0
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/deepmap/oapi-codegen v1.12.4 // indirect
	github.com/google/uuid v1.3.0 // indirect
)
