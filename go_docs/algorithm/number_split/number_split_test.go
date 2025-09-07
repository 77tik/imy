package number_split

import (
	"testing"
)

func TestCountEvenSumSplits(t *testing.T) {
	tests := []struct {
		n        int
		expected int
		desc     string
	}{
		{0, 0, "零的情况"},
		{1, 0, "1无法分成两个正数"},
		{2, 1, "2可以分成1+1"},
		{3, 0, "3是奇数，无法分成两个和为偶数的数"},
		{4, 2, "4可以分成1+3或2+2"},
		{5, 0, "5是奇数，无法分成两个和为偶数的数"},
		{6, 3, "6可以分成1+5,2+4,3+3"},
		{7, 0, "7是奇数，无法分成两个和为偶数的数"},
		{8, 4, "8可以分成1+7,2+6,3+5,4+4"},
		{10, 5, "10有5种分法"},
		{100, 50, "100有50种分法"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := CountEvenSumSplits(tt.n)
			if result != tt.expected {
				t.Errorf("CountEvenSumSplits(%d) = %d, expected %d", tt.n, result, tt.expected)
			}
			
			// 测试简洁版本
			resultSimple := CountEvenSumSplitsSimple(tt.n)
			if resultSimple != tt.expected {
				t.Errorf("CountEvenSumSplitsSimple(%d) = %d, expected %d", tt.n, resultSimple, tt.expected)
			}
		})
	}
}

func TestCountEvenSumSplitsWithDetails(t *testing.T) {
	tests := []struct {
		n               int
		expectedCount   int
		expectedSplits  [][]int
	}{
		{4, 2, [][]int{{1, 3}, {2, 2}}},
		{6, 3, [][]int{{1, 5}, {2, 4}, {3, 3}}},
		{8, 4, [][]int{{1, 7}, {2, 6}, {3, 5}, {4, 4}}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			count, splits := CountEvenSumSplitsWithDetails(tt.n)
			if count != tt.expectedCount {
				t.Errorf("Count = %d, expected %d", count, tt.expectedCount)
			}
			if len(splits) != len(tt.expectedSplits) {
				t.Errorf("Splits length = %d, expected %d", len(splits), len(tt.expectedSplits))
				return
			}
			for i, split := range splits {
				if len(split) != 2 || split[0] != tt.expectedSplits[i][0] || split[1] != tt.expectedSplits[i][1] {
					t.Errorf("Split[%d] = %v, expected %v", i, split, tt.expectedSplits[i])
				}
				// 验证和确实为偶数
				if (split[0]+split[1])%2 != 0 {
					t.Errorf("Split %v sum is not even", split)
				}
			}
		})
	}
}

// 验证数学公式的正确性
func TestMathFormula(t *testing.T) {
	for n := 1; n <= 20; n++ {
		// 用详细版本计算实际结果
		actual, _ := CountEvenSumSplitsWithDetails(n)
		// 用公式计算
		formula := CountEvenSumSplitsSimple(n)
		
		if actual != formula {
			t.Errorf("n=%d: actual=%d, formula=%d", n, actual, formula)
		}
	}
}

// 基准测试
func BenchmarkCountEvenSumSplits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CountEvenSumSplits(1000)
	}
}

func BenchmarkCountEvenSumSplitsSimple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CountEvenSumSplitsSimple(1000)
	}
}

func BenchmarkCountEvenSumSplitsWithDetails(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CountEvenSumSplitsWithDetails(100) // 较小的数字，因为这个版本较慢
	}
}