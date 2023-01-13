package internal

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var testCasesByVersion = []struct {
	name     string
	dlds     []Download
	expected []Download
}{
	{
		name: "rc",
		dlds: []Download{
			{
				Version: "1.19.1rc1",
			},
			{
				Version: "1.19.1",
			},
			{
				Version: "1.20rc3",
			},
			{
				Version: "1.3",
			},
		},
		expected: []Download{
			{
				Version: "1.20rc3",
			},
			{
				Version: "1.19.1",
			},
			{
				Version: "1.19.1rc1",
			},
			{
				Version: "1.3",
			},
		},
	},
}

func TestByVersion(t *testing.T) {
	for i := range testCasesByVersion {
		i := i
		t.Run(testCasesByVersion[i].name, func(t *testing.T) {
			sort.Sort(ByVersion(testCasesByVersion[i].dlds))
			diff := cmp.Diff(testCasesByVersion[i].dlds, testCasesByVersion[i].expected)
			if diff != "" {
				t.Logf("mismatch in expectation: \n\n%s", diff)
				t.Logf("%q", testCasesByVersion[i].dlds)
				t.Fail()
			}
		})
	}
}
