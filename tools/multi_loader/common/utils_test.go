package common

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vhive-serverless/loader/tools/multi_loader/types"
)

func TestNextProduct(t *testing.T) {
	ints := []int{2, 1}
	nextProduct := NextCProduct(ints)
	expectedArrs := [][]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {2, 0}, {2, 1}}
	curI := 0

	for {
		product := nextProduct()
		if len(product) == 0 {
			if curI != len(expectedArrs) {
				t.Fatalf("Expected %d products, got %d", len(expectedArrs), curI)
			}
			break
		}
		if len(product) != len(ints) {
			t.Fatalf("Expected product length %d, got %d", len(ints), len(product))
		}
		for i, v := range product {
			if v != expectedArrs[curI][i] {
				t.Fatalf("Expected %v, got %v", expectedArrs[curI], product)
			}
		}
		curI++
	}
}

func TestSplitPath(t *testing.T) {
	assert.Equal(t, []string{"file.txt"}, SplitPath("file.txt"), "Expected ['file.txt'] for single file")
	assert.Equal(t, []string{"home", "user", "docs", "file.txt"}, SplitPath(filepath.Join("home", "user", "docs", "file.txt")), "Expected full path split")
}

func TestSweepOptionsToPostfix(t *testing.T) {
	t.Run("Test Post Fix Naming Util", func(t *testing.T) {
		result := SweepOptionsToPostfix(
			[]types.SweepOptions{
				{Field: "PreScript", Values: []interface{}{"PreValue_1", "PreValue_2"}},
				{Field: "CPULimit", Values: []interface{}{"1vCPU", "2vCPU", "4vCPU"}},
				{Field: "ExperimentDuration", Values: []interface{}{"10", "20", "30"}},
				{Field: "PostScript", Values: []interface{}{"PostValue_1", "PostValue_2", "PostValue_3"}},
			},
			[]int{1, 2, 0, 2},
		)
		assert.Equal(t, "_PreScript_PreValue_2_CPULimit_4vCPU_ExperimentDuration_10_PostScript_PostValue_3", result, "Unexpected postfix result")
	})
}
