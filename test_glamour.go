package main

import (
	"fmt"
	"strings"
	"github.com/charmbracelet/glamour"
)

func main() {
	r, _ := glamour.NewTermRenderer(glamour.WithStandardStyle("dark"), glamour.WithWordWrap(100))
	out, _ := r.Render("This is the distilled state of Alpha")
	fmt.Printf("%q\n", out)
	fmt.Println(strings.Contains(out, "This is the distilled state of Alpha"))
}
