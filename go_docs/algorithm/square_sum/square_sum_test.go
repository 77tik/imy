package square_sum

import (
	"reflect"
	"testing"
)

func TestMinSquareSum(t *testing.T) {
	tests := []struct {
		n        int
		expected int
		desc     string
	}{
		{0, 0, "零的情况"},
		{1, 1, "1 = 1²"},
		{2, 2, "2 = 1² + 1²"},
		{3, 3, "3 = 1² + 1² + 1²"},
		{4, 1, "4 = 2²"},
		{5, 2, "5 = 2² + 1²"},
		{6, 3, "6 = 2² + 1² + 1²"},
		{7, 4, "7 = 2² + 1² + 1² + 1²"},
		{8, 2, "8 = 2² + 2²"},
		{9, 1, "9 = 3²"},
		{10, 2, "10 = 3² + 1²"},
		{11, 3, "11 = 3² + 1² + 1²"},
		{12, 3, "12 = 2² + 2² + 2²"},
		{13, 2, "13 = 3² + 2²"},
		{14, 3, "14 = 3² + 2² + 1²"},
		{15, 4, "15 = 3² + 2² + 1² + 1²"},
		{16, 1, "16 = 4²"},
		{17, 2, "17 = 4² + 1²"},
		{18, 2, "18 = 3² + 3²"},
		{19, 3, "19 = 3² + 3² + 1²"},
		{20, 2, "20 = 4² + 2²"},
		{25, 1, "25 = 5²"},
		{26, 2, "26 = 5² + 1²"},
		{27, 3, "27 = 3² + 3² + 3²"},
		{28, 4, "28需要4个平方数"},
		{43, 3, "43 = 6² + 2² + 1²"},
		{87, 4, "87需要4个平方数"},
		{100, 1, "100 = 10²"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// 测试动态规划版本
			result := MinSquareSum(tt.n)
			if result != tt.expected {
				t.Errorf("MinSquareSum(%d) = %d, expected %d", tt.n, result, tt.expected)
			}
			
			// 测试优化版本
			resultOpt := MinSquareSumOptimized(tt.n)
			if resultOpt != tt.expected {
				t.Errorf("MinSquareSumOptimized(%d) = %d, expected %d", tt.n, resultOpt, tt.expected)
			}
			
			// 对于小数字，测试暴力解法
			if tt.n <= 20 {
				resultBrute := BruteForce(tt.n)
				if resultBrute != tt.expected {
					t.Errorf("BruteForce(%d) = %d, expected %d", tt.n, resultBrute, tt.expected)
				}
				
				resultMemo := BruteForceWithMemo(tt.n)
				if resultMemo != tt.expected {
					t.Errorf("BruteForceWithMemo(%d) = %d, expected %d", tt.n, resultMemo, tt.expected)
				}
			}
		})
	}
}

func TestMinSquareSumWithPath(t *testing.T) {
	tests := []struct {
		n            int
		expectedCount int
		validPaths   [][]int // 可能的有效路径
	}{
		{4, 1, [][]int{{4}}},
		{5, 2, [][]int{{4, 1}}},
		{8, 2, [][]int{{4, 4}}},
		{9, 1, [][]int{{9}}},
		{10, 2, [][]int{{9, 1}}},
		{13, 2, [][]int{{9, 4}, {4, 9}}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			count, path := MinSquareSumWithPath(tt.n)
			if count != tt.expectedCount {
				t.Errorf("Count = %d, expected %d", count, tt.expectedCount)
			}
			
			// 验证路径的正确性
			sum := 0
			for _, square := range path {
				sum += square
				if !isSquare(square) {
					t.Errorf("Path contains non-square number: %d", square)
				}
			}
			if sum != tt.n {
				t.Errorf("Path sum = %d, expected %d. Path: %v", sum, tt.n, path)
			}
			
			// 检查路径是否为有效路径之一
			validPath := false
			for _, validP := range tt.validPaths {
				if reflect.DeepEqual(path, validP) {
					validPath = true
					break
				}
			}
			if !validPath && len(tt.validPaths) > 0 {
				t.Logf("Path %v is valid but not in expected paths %v", path, tt.validPaths)
			}
		})
	}
}

func TestIsSquare(t *testing.T) {
	tests := []struct {
		n        int
		expected bool
	}{
		{0, true},
		{1, true},
		{2, false},
		{3, false},
		{4, true},
		{5, false},
		{9, true},
		{10, false},
		{16, true},
		{25, true},
		{26, false},
		{100, true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := isSquare(tt.n)
			if result != tt.expected {
				t.Errorf("isSquare(%d) = %v, expected %v", tt.n, result, tt.expected)
			}
		})
	}
}

func TestIsFormOf4a8b7(t *testing.T) {
	tests := []struct {
		n        int
		expected bool
	}{
		{7, true},   // 4^0 * (8*0 + 7) = 7
		{15, true},  // 4^0 * (8*1 + 7) = 15
		{23, true},  // 4^0 * (8*2 + 7) = 23
		{28, true},  // 4^1 * (8*0 + 7) = 28
		{31, true},  // 4^0 * (8*3 + 7) = 31
		{60, true},  // 4^1 * (8*1 + 7) = 60
		{87, true},  // 4^0 * (8*10 + 7) = 87
		{112, true}, // 4^2 * (8*0 + 7) = 112
		{1, false},
		{2, false},
		{3, false},
		{4, false},
		{5, false},
		{6, false},
		{8, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := isFormOf4a8b7(tt.n)
			if result != tt.expected {
				t.Errorf("isFormOf4a8b7(%d) = %v, expected %v", tt.n, result, tt.expected)
			}
		})
	}
}

// 验证所有算法的一致性
func TestAlgorithmConsistency(t *testing.T) {
	for n := 1; n <= 50; n++ {
		dp := MinSquareSum(n)
		opt := MinSquareSumOptimized(n)
		memo := BruteForceWithMemo(n)
		
		if dp != opt {
			t.Errorf("n=%d: DP=%d, Optimized=%d", n, dp, opt)
		}
		if dp != memo {
			t.Errorf("n=%d: DP=%d, Memo=%d", n, dp, memo)
		}
		
		// 验证路径版本
		count, path := MinSquareSumWithPath(n)
		if count != dp {
			t.Errorf("n=%d: Path count=%d, DP=%d", n, count, dp)
		}
		
		// 验证路径的正确性
		sum := 0
		for _, square := range path {
			sum += square
			if !isSquare(square) {
				t.Errorf("n=%d: Path contains non-square: %d", n, square)
			}
		}
		if sum != n {
			t.Errorf("n=%d: Path sum=%d, expected=%d", n, sum, n)
		}
	}
}

// 基准测试
func BenchmarkMinSquareSum(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MinSquareSum(100)
	}
}

func BenchmarkMinSquareSumOptimized(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MinSquareSumOptimized(100)
	}
}

func BenchmarkBruteForceWithMemo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BruteForceWithMemo(50) // 较小的数字，因为递归较慢
	}
}

func BenchmarkMinSquareSumWithPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MinSquareSumWithPath(100)
	}
}

// 大数字测试
func TestLargeNumbers(t *testing.T) {
	tests := []int{1000, 5000, 10000}
	
	for _, n := range tests {
		t.Run("", func(t *testing.T) {
			dp := MinSquareSum(n)
			opt := MinSquareSumOptimized(n)
			
			if dp != opt {
				t.Errorf("n=%d: DP=%d, Optimized=%d", n, dp, opt)
			}
			
			// 结果应该在合理范围内（根据拉格朗日四平方定理，最多4个）
			if dp < 1 || dp > 4 {
				t.Errorf("n=%d: result=%d is out of range [1,4]", n, dp)
			}
		})
	}
}