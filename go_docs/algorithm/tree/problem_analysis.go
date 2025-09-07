package tree

// 重新分析你的代码问题
// 让我们逐步对比你的代码和正确实现

/*
你的原始代码分析：

func getDis(root *TreeNode) int {
    if root == nil {
        return 0
    }

    var Max *TreeNode
    var Min *TreeNode

    // 1. getMaxMin函数 - 这部分看起来是正确的
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

    // 2. getLca函数 - 这部分也是正确的
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

    // 3. getDisNode函数 - 问题在这里！
    var getDisNode func(root *TreeNode, target *TreeNode) int
    getDisNode = func(root *TreeNode, target *TreeNode) int {
        if root == nil {
            return -1
        }
        if root == target {
            return 0
        }
        d := getDisNode(root.Left, target)
        if d == -1 {
            // 错误1：这里直接返回了右子树结果+1，没有检查右子树是否找到
            return getDisNode(root.Right, target) + 1
        }
        // 错误2：这里没有检查d是否为-1就直接+1
        return d + 1
    }

    getMaxMin(root)
    lca := getLca(root)
    if Max==nil || Min == nil {
        return 0
    }
    a := getDisNode(lca, Max)
    b := getDisNode(lca, Min)
    return a + b
}
*/

// 问题分析：
// 1. 你的getDisNode函数有两个严重错误
// 2. 当在右子树查找时，不管是否找到都加了1
// 3. 这会导致距离计算错误

// 让我们看看可能的其他问题：

// 问题1：题目可能要求的是"叶子节点"中的最大最小值
// 你的代码找的是所有节点中的最大最小值
func getMaxMinLeafOnly(root *TreeNode) (*TreeNode, *TreeNode) {
    var maxLeaf, minLeaf *TreeNode
    
    var dfs func(*TreeNode)
    dfs = func(node *TreeNode) {
        if node == nil {
            return
        }
        
        // 检查是否为叶子节点
        if node.Left == nil && node.Right == nil {
            if maxLeaf == nil || node.Val > maxLeaf.Val {
                maxLeaf = node
            }
            if minLeaf == nil || node.Val < minLeaf.Val {
                minLeaf = node
            }
        }
        
        dfs(node.Left)
        dfs(node.Right)
    }
    
    dfs(root)
    return maxLeaf, minLeaf
}

// 问题2：可能需要处理特殊情况
func getDisCorrectVersion(root *TreeNode) int {
    if root == nil {
        return 0
    }
    
    // 如果题目要求叶子节点，使用这个函数
    maxNode, minNode := getMaxMinLeafOnly(root)
    
    // 如果最大最小值是同一个节点，距离为0
    if maxNode == minNode {
        return 0
    }
    
    if maxNode == nil || minNode == nil {
        return 0
    }
    
    // 找LCA
    lca := findLCA(root, maxNode, minNode)
    if lca == nil {
        return 0
    }
    
    // 计算距离
    distToMax := getDistance(lca, maxNode)
    distToMin := getDistance(lca, minNode)
    
    if distToMax == -1 || distToMin == -1 {
        return 0
    }
    
    return distToMax + distToMin
}

// 正确的距离计算函数
func getDistance(root *TreeNode, target *TreeNode) int {
    if root == nil {
        return -1
    }
    if root == target {
        return 0
    }
    
    // 先在左子树查找
    leftDist := getDistance(root.Left, target)
    if leftDist != -1 {
        return leftDist + 1
    }
    
    // 左子树没找到，在右子树查找
    rightDist := getDistance(root.Right, target)
    if rightDist != -1 {
        return rightDist + 1
    }
    
    // 都没找到
    return -1
}

// 正确的LCA函数
func findLCA(root *TreeNode, p *TreeNode, q *TreeNode) *TreeNode {
    if root == nil {
        return nil
    }
    
    if root == p || root == q {
        return root
    }
    
    left := findLCA(root.Left, p, q)
    right := findLCA(root.Right, p, q)
    
    if left != nil && right != nil {
        return root
    }
    
    if left != nil {
        return left
    }
    
    return right
}

/*
可能的问题总结：
1. 题目可能要求的是"叶子节点"中的最大最小值，而不是所有节点
2. getDisNode函数逻辑错误（这是主要问题）
3. 边界情况处理不当
4. 可能需要考虑最大最小值是同一个节点的情况

建议：
1. 确认题目是否要求叶子节点
2. 修正getDisNode函数的逻辑
3. 添加更多边界情况检查
*/