package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/haruyama480/gincontextleak"
)

func main() {
	singlechecker.Main(gincontextleak.Analyzer)
}
