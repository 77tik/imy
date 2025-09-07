package number_split

// CountEvenSumSplits 计算将数字n分成两个和为偶数的数的分法数量
// 数学原理：两个数的和为偶数 ⟺ 两个数同为奇数或同为偶数
// 关键洞察：如果n是奇数，无法分成两个和为偶数的数（因为奇数无法表示为两个同奇偶性数的和）
func CountEvenSumSplits(n int) int {
	if n <= 0 {
		return 0
	}
	
	// 如果n是奇数，无法分成两个和为偶数的数
	if n%2 == 1 {
		return 0
	}
	
	// 如果n是偶数，可以分成两个奇数或两个偶数
	// 分法数就是 n/2（每种情况下的分法数）
	return n / 2
}

// CountEvenSumSplitsSimple 最简洁的实现
func CountEvenSumSplitsSimple(n int) int {
	if n <= 0 || n%2 == 1 {
		return 0
	}
	return n / 2
}

// CountEvenSumSplitsWithDetails 带详细说明的版本
func CountEvenSumSplitsWithDetails(n int) (int, [][]int) {
	if n <= 0 {
		return 0, nil
	}
	
	var splits [][]int
	count := 0
	
	// 遍历所有可能的第一个数
	for i := 1; i < n; i++ {
		j := n - i
		// 检查两个数的和是否为偶数
		if (i+j)%2 == 0 {
			// 避免重复计算 (i,j) 和 (j,i)
			if i <= j {
				splits = append(splits, []int{i, j})
				count++
			}
		}
	}
	
	return count, splits
}