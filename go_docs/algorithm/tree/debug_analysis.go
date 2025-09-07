package tree

// 问题分析：你的代码中 getDisNode 函数有逻辑错误
// 让我们对比正确版本和错误版本

// 你的错误版本：
func getDisNodeWrong(root *TreeNode, target *TreeNode) int {
	if root == nil {
		return -1
	}
	if root == target {
		return 0
	}
	d := getDisNodeWrong(root.Left, target)
	if d == -1 {
		// 错误：这里直接返回了 getDisNodeWrong(root.Right, target) + 1
		// 但是如果右子树也没找到（返回-1），那么 -1 + 1 = 0，这是错误的
		return getDisNodeWrong(root.Right, target) + 1
	}
	// 这里也有问题，应该检查 d 是否为 -1
	return d + 1
}

// 正确版本：
func getDisNodeCorrect(root *TreeNode, target *TreeNode) int {
	if root == nil {
		return -1
	}
	if root == target {
		return 0
	}

	// 先在左子树查找
	d := getDisNodeCorrect(root.Left, target)
	if d == -1 {
		// 左子树没找到，再在右子树查找
		d = getDisNodeCorrect(root.Right, target)
	}
	
	// 关键：只有找到了（d != -1）才加1
	if d != -1 {
		return d + 1
	}
	return -1
}

// 修正后的完整解决方案：
func getDisFixed(root *TreeNode) int {
	if root == nil {
		return 0
	}

	var Max *TreeNode
	var Min *TreeNode

	// 找最大最小值节点
	var getMaxMin func(root *TreeNode)
	getMaxMin = func(root *TreeNode) {
		if root == nil {
			return
		}
		if Max == nil || Max.Val < root.Val {
			Max = root
		}
		if Min == nil || Min.Val > root.Val {
			Min = root
		}
		getMaxMin(root.Left)
		getMaxMin(root.Right)
	}

	// 找最近公共祖先
	var getLca func(root *TreeNode) *TreeNode
	getLca = func(root *TreeNode) *TreeNode {
		if root == nil {
			return root
		}
		if root == Max || root == Min {
			return root
		}
		l := getLca(root.Left)
		r := getLca(root.Right)
		if l != nil && r != nil {
			return root
		}
		if l != nil {
			return l
		}
		if r != nil {
			return r
		}
		return nil
	}

	// 正确的距离计算函数
	var getDisNode func(root *TreeNode, target *TreeNode) int
	getDisNode = func(root *TreeNode, target *TreeNode) int {
		if root == nil {
			return -1
		}
		if root == target {
			return 0
		}
		
		// 先在左子树查找
		d := getDisNode(root.Left, target)
		if d == -1 {
			// 左子树没找到，再在右子树查找
			d = getDisNode(root.Right, target)
		}
		
		// 关键修正：只有找到了才加1
		if d != -1 {
			return d + 1
		}
		return -1
	}

	getMaxMin(root)
	if Max == nil || Min == nil {
		return 0
	}
	
	lca := getLca(root)
	if lca == nil {
		return 0
	}
	
	a := getDisNode(lca, Max)
	b := getDisNode(lca, Min)
	
	// 确保距离有效
	if a == -1 || b == -1 {
		return 0
	}
	
	return a + b
}

/*
问题总结：
1. 你的 getDisNode 函数在右子树查找时，不管是否找到都加了1
2. 正确做法是：只有确实找到目标节点（返回值不是-1）才应该加1
3. 这导致在某些情况下返回错误的距离值

修正要点：
- 先尝试左子树，如果没找到（返回-1）再尝试右子树
- 只有在找到目标节点时（d != -1）才返回 d + 1
- 如果两个子树都没找到，返回-1表示未找到
*/