package main

//go:generate buf generate proto -o proto
//go:generate swagger generate client -f proto/service.swagger.json -t test/integration/clihttp
