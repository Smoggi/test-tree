package go_ip_trees

import (
	"net"
	"sort"
)

type SpecialNode struct {
	IPv4TreeNode
}

type SpecialTree struct {
	Root     *IPv4TreeNode
	nodes    int
	nodesMap map[string]*IPv4TreeNode
}

func NewSpecialTree() *SpecialTree {
	return &SpecialTree{
		Root:     NewIPv4TreeNode("", 0, 0, nil, false, nil),
		nodes:    1,
		nodesMap: make(map[string]*IPv4TreeNode),
	}
}

func (t *SpecialTree) insertNode(prev *IPv4TreeNode, key string, size int, isLast bool) *IPv4TreeNode {
	node := NewIPv4TreeNode(key, prev.PrefixLen+1, size, prev, isLast, nil)
	t.nodes++
	return node
}

func (t *SpecialTree) Insert(cidr string, info map[string]interface{}) {
	_, network, _ := net.ParseCIDR(cidr)

	prefixLen, _ := network.Mask.Size()
	binIP, _ := GetBinaryPathFromIPv4Addr(network)

	var prev *IPv4TreeNode
	size := CalculateTotalHosts(network)
	node := t.Root

	for _, bit := range binIP {
		prev = node
		node = prev.Child(string(bit))
		if node == nil {
			node = t.insertNode(prev, string(bit), size, false) //info ?
		} else {
			node.Update(node.PrefixLen, size)
		}

		if node.PrefixLen == prefixLen {
			break
		}
	}

	node.IsLast = true
}

func (t *SpecialTree) Supernet(cidr string) *IPv4TreeNode {
	_, network, _ := net.ParseCIDR(cidr)

	prefixLen, _ := network.Mask.Size()
	node := t.Root
	var lastNodes []*IPv4TreeNode
	path, _ := GetBinaryPathFromIPv4Addr(network)
	for _, bit := range path {
		prev := node
		node = prev.Child(string(bit))
		if node == nil {
			break
		}

		if node.IsLast {
			lastNodes = append(lastNodes, node)
		}
		if node.PrefixLen == prefixLen {
			lastNodes = append(lastNodes, node)
			break
		}
	}

	if len(lastNodes) == 0 {
		return nil
	}
	return lastNodes[len(lastNodes)-1]
}

func (t *SpecialTree) Subnets() []*IPv4TreeNode {
	var result []*IPv4TreeNode
	var dfs func(node *IPv4TreeNode)
	dfs = func(node *IPv4TreeNode) {
		if node == nil {
			return
		}

		for _, child := range node.Children {
			dfs(child)
		}
		result = append(result, node)
		return
	}
	dfs(t.Root)

	// Optional: sort by IP order
	sort.Slice(result, func(i, j int) bool {
		return result[i].Int() < result[j].Int()
	})
	return result
}
