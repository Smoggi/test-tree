package go_ip_trees

import (
	"net"
)

type CIDRTree struct {
	Root     *IPv4TreeNode
	nodes    int
	nodesMap map[string]*IPv4TreeNode
}

func NewCIDRTree() *CIDRTree {
	return &CIDRTree{
		Root:     NewIPv4TreeNode("", 0, 0, nil, false, nil),
		nodes:    1,
		nodesMap: make(map[string]*IPv4TreeNode),
	}
}

func (t *CIDRTree) insertNode(prev *IPv4TreeNode, key string, size int, isLast bool) *IPv4TreeNode {
	node := NewIPv4TreeNode(key, prev.PrefixLen+1, size, prev, isLast, nil)
	t.nodes++
	return node
}

func (t *CIDRTree) Insert(cidr string, attrs map[string]interface{}) error {
	_, network, _ := net.ParseCIDR(cidr)

	for n := range t.Root.IteratorIPv4Tree() { // так ли ?
		if n.String() == network.String() {
			return nil
		}
	}

	var prev *IPv4TreeNode
	var excess int
	size := CalculateTotalHosts(network)
	node := t.Root
	t.Root.Update(-1, size)
	wasInsert := false

	prefixLen, _ := network.Mask.Size()
	binIP, _ := GetBinaryPathFromIPv4Addr(network)

	for _, bit := range binIP {
		prev = node
		node = prev.Child(string(bit))
		if node == nil {
			node = t.insertNode(prev, string(bit), size, false) //info ?
			wasInsert = true
		} else {
			node.Update(node.PrefixLen, size)
		}

		if node.PrefixLen == prefixLen {
			break
		}
	}

	node.IsLast = true
	if !wasInsert {
		if node.PrefixLen != prefixLen {
			excess = size
		} else {
			excess = node.Size - size
		}
		for n := range node.IteratorIPv4Tree() {
			n.Update(-1, -excess)
			n = *n.Parent
		}
	} else {
		t.nodesMap[network.String()] = node
	}

	return nil
}

func (t *CIDRTree) Supernet(cidr string) *IPv4TreeNode {
	_, netw, _ := net.ParseCIDR(cidr)

	prefixLen, _ := netw.Mask.Size()
	node := t.Root
	var lastNodes []*IPv4TreeNode
	path, _ := GetBinaryPathFromIPv4Addr(netw)
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
