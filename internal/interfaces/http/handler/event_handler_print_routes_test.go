package handler_test

import (
	"fmt"
	"testing"
)

func TestPrintRoutes(t *testing.T) {
	app, _, _, _ := SetupTestApp()
	for _, route := range app.GetRoutes() {
		fmt.Printf("%s %s\n", route.Method, route.Path)
	}
}
