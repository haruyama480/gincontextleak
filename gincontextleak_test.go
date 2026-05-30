package gincontextleak_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/haruyama480/gincontextleak"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, gincontextleak.Analyzer, "a")
}

func TestFixes(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, gincontextleak.Analyzer, "a")
}
