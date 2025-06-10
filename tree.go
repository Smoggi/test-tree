package go_ip_trees

import (
	"fmt"
	"iter"
	"net"
	"sort"
	"strconv"
	"strings"
)

type IPv4TreeNode struct {
	Key       string
	PrefixLen int
	Size      int
	Parent    *IPv4TreeNode
	IsLast    bool
	Info      map[string]interface{}
	Children  [2]*IPv4TreeNode
	Prefix    string
}

func NewIPv4TreeNode(key string, prefixLen int, size int, parent *IPv4TreeNode, isLast bool, info map[string]interface{}) *IPv4TreeNode {
	node := &IPv4TreeNode{
		Key:       key,
		PrefixLen: prefixLen,
		Size:      size,
		Parent:    parent,
		IsLast:    isLast,
		Info:      info,
	}
	if parent != nil {
		parent.NewChild(key, node)
		node.Prefix = parent.Prefix + key
	}
	return node
}

func (n *IPv4TreeNode) Child(key string) *IPv4TreeNode {
	i, _ := strconv.Atoi(key)
	return n.Children[i]
}

func (n *IPv4TreeNode) NewChild(key string, child *IPv4TreeNode) {
	i, _ := strconv.Atoi(key)
	n.Children[i] = child
}

func (n *IPv4TreeNode) Update(prefixLen int, size int) {
	if prefixLen > n.PrefixLen {
		n.IsLast = true
	}
	n.Size += size
}

func (n *IPv4TreeNode) Fullness() float64 {
	if n.PrefixLen == 32 {
		return 1.0
	}
	if n.PrefixLen == 31 {
		return float64(n.Size) / 2.0
	}
	return float64(n.Size) / float64(uint32(1)<<uint(32-n.PrefixLen))
}

func (n *IPv4TreeNode) TrueLastNode() bool {
	return n.Children[0] == nil && n.Children[1] == nil
}

func (n *IPv4TreeNode) Aggregate(fullness float64) {
	if n.Fullness() >= fullness && n.TrueLastNode() {
		n.IsLast = true
	}
}

func (n *IPv4TreeNode) Int() int64 {
	s := n.Prefix + strings.Repeat("0", 32-len(n.Prefix))
	val, err := strconv.ParseInt(s, 2, 64)
	if err != nil {
		return 0
	}
	return val
}

func (n *IPv4TreeNode) String() string {
	if n.PrefixLen > 0 {
		ip := GetIPv4FromBinaryString(n.Prefix + strings.Repeat("0", 32-n.PrefixLen))
		return fmt.Sprintf("%s/%d", ip.String(), n.PrefixLen)
	}
	return "root"
}

func (n *IPv4TreeNode) Sizeof() int {
	if n.IsLast {
		if n.Size > 0 {
			return n.Size
		}
		return 1 << uint(32-n.PrefixLen)
	}
	size := 0
	for _, child := range n.Children {
		if child != nil {
			size += child.Sizeof()
		}
	}
	return size
}

func (n *IPv4TreeNode) isRoot() bool {
	return n.String() == "root"
}

func (n *IPv4TreeNode) NetworkAddress() net.IP {
	s := n.String()
	parts := strings.Split(s, "/")
	return net.ParseIP(parts[0])
}

// IteratorIPv4Tree для IPv4Tree
func (n *IPv4TreeNode) IteratorIPv4Tree() iter.Seq[IPv4TreeNode] {
	return func(yield func(IPv4TreeNode) bool) {
		if !n.IsLast {
			for _, v := range n.Children {
				if !yield(*v) {
					return
				}
			}
		}
	}
}

// RecLastChildren рекурсивная функция для получения всех последних дочерних узлов взято из SpecialTree
func (n *IPv4TreeNode) RecLastChildren() []*IPv4TreeNode {
	var result []*IPv4TreeNode

	if n.IsLast {
		result = append(result, n)
	}

	for _, child := range n.Children {
		if child != nil {
			result = append(result, child.RecLastChildren()...)
		}
	}

	return result
}

func (n *IPv4TreeNode) GetInnerLastNodes() []*IPv4TreeNode {
	var result []*IPv4TreeNode

	for _, child := range n.Children {
		if child != nil && !child.IsLast {
			result = append(result, child.GetInnerLastNodes()...)
		}
	}

	result = append(result, n.RecLastChildren()...)
	return result
}

// -----------------------------
// IPv4Tree
// -----------------------------

type IPv4Tree struct {
	Root     *IPv4TreeNode
	nodes    int
	nodesMap map[string]*IPv4TreeNode
}

func NewIPv4Tree() *IPv4Tree {
	return &IPv4Tree{
		Root:     NewIPv4TreeNode("", 0, 0, nil, false, nil),
		nodes:    1,
		nodesMap: make(map[string]*IPv4TreeNode),
	}
}

func (t *IPv4Tree) insertNode(prev *IPv4TreeNode, key string, size int, isLast bool) *IPv4TreeNode {
	node := NewIPv4TreeNode(key, prev.PrefixLen+1, size, prev, isLast, nil)
	t.nodes++
	return node
}

// Add inserts another IPv4Tree into this tree.
func (t *IPv4Tree) Add(other *IPv4Tree) *IPv4Tree {
	for _, node := range other.Subnets() {
		if node.IsLast {
			ipStr := node.Key
			if err := t.Insert(ipStr); err != nil {
				return nil
			}
		}
	}
	return t
}

func (t *IPv4Tree) Sizeof(ip string) int {
	_, network, err := net.ParseCIDR(ip)
	if err != nil {
		panic(err)
	}
	node := t.Root
	binIP, _ := GetBinaryPathFromIPv4Addr(network)
	prefixLen, _ := network.Mask.Size()
	for _, n := range binIP {
		node = node.Child(string(n))
		if node == nil {
			return 0
		}
		if node.PrefixLen == prefixLen {
			break
		}
	}
	return node.Sizeof()
}

func (t *IPv4Tree) Subnets() []*IPv4TreeNode {
	var result []*IPv4TreeNode
	var dfs func(node *IPv4TreeNode)
	dfs = func(node *IPv4TreeNode) {
		if node == nil {
			return
		}
		if node.IsLast {
			result = append(result, node)
			return
		}
		for _, child := range node.Children {
			dfs(child)
		}
	}
	dfs(t.Root)

	// Optional: sort by IP order
	sort.Slice(result, func(i, j int) bool {
		return result[i].Int() < result[j].Int()
	})
	return result
}

func (t *IPv4Tree) Delete(ip string) error {
	_, netw, err := net.ParseCIDR(ip)
	if err != nil {
		return err
	}
	if !t.Intree(netw.String()) {
		return nil
	}
	size := t.Sizeof(netw.String())
	node := t.Root
	prev := node
	inLast := false
	prefixLen, _ := netw.Mask.Size()
	invKey := "0"

	binIP, _ := GetBinaryPathFromIPv4Addr(netw)
	for _, bit := range binIP {
		prev = node
		node = prev.Child(string(bit))
		if bit == 0 {
			invKey = "1"
		} else {
			invKey = "0"
		}
		invNode := prev.Child(invKey)
		if prev.IsLast && !inLast {
			inLast = true
		}
		if node == nil {
			node = t.insertNode(prev, string(bit), PrefixSize(prev.PrefixLen+1), false)
		}
		if invNode == nil {
			invNode = t.insertNode(prev, invKey, PrefixSize(prev.PrefixLen+1), false)
			invNode.IsLast = inLast
		}
		prev.IsLast = false
		if node.PrefixLen == prefixLen {
			return nil
		}
	}
	if prev.TrueLastNode() {
		prev.IsLast = true
	}

	for n := range prev.IteratorIPv4Tree() { // так ли ?
		n.Update(-1, size)
		n = *n.Parent
	}
	return nil
}

func (t *IPv4Tree) Intree(ip string) bool {
	_, network, _ := net.ParseCIDR(ip)

	if network == nil {
		return false
	}
	for _, n := range t.Subnets() { // так ли ?
		if n.String() == network.String() {
			return true
		}
	}

	binIP, _ := GetBinaryPathFromIPv4Addr(network)
	prefixLen, _ := network.Mask.Size()
	node := t.Root
	for _, bit := range binIP {
		prev := node
		node = prev.Child(string(bit))
		if node == nil {
			return false
		}
		if node.IsLast || node.PrefixLen == prefixLen {
			return true
		}
	}
	return true
}

func (t *IPv4Tree) Supernet(ip string) *IPv4TreeNode {
	_, netw, _ := net.ParseCIDR(ip)
	node := t.Root
	prefixLen, _ := netw.Mask.Size()
	binIP, _ := GetBinaryPathFromIPv4Addr(netw)

	for _, bit := range binIP {
		prev := node
		node = prev.Child(string(bit))
		if node == nil {
			return nil
		}
		if node.IsLast || node.PrefixLen == prefixLen {
			break
		}
	}
	return node
}

// Public InsertNode mb not used
func (t *IPv4Tree) PubInsertNode(newNode *IPv4TreeNode) error {
	_, network, _ := net.ParseCIDR(newNode.String())

	for _, n := range t.Subnets() { // так ли ?
		if n.String() == network.String() {
			return nil
		}
	}

	var prev *IPv4TreeNode
	var excess int
	size := newNode.Size
	node := t.Root
	t.Root.Update(-1, size)
	wasInsert := false

	prefixLen, _ := network.Mask.Size()
	binIP, _ := GetBinaryPathFromIPv4Addr(network)

	for _, bit := range binIP {
		prev = node
		node = prev.Child(string(bit))
		if node == nil {
			node = t.insertNode(prev, string(bit), size, false)
			wasInsert = true
		} else {
			node.Update(node.PrefixLen, size)
		}
		if node.PrefixLen == prefixLen {
			newNode.Parent = prev
			node = newNode
		}
		break
	}

	//prev.NewChild(bit, newNode) // последний символ ?
	if !wasInsert {
		if node.PrefixLen != prefixLen {
			excess = size
		} else {
			excess = node.Size - size
		}
		for n := range node.IteratorIPv4Tree() { // так ли ?
			n.Update(-1, excess)
			n = *n.Parent
		}
	} else {
		t.nodesMap[network.String()] = node
	}
	return nil
}

func (t *IPv4Tree) Insert(ip string) error {
	_, network, _ := net.ParseCIDR(ip)

	for _, n := range t.Subnets() { // так ли ?
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
		if node.IsLast {
			break
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
		for n := range node.IteratorIPv4Tree() { // так ли ?
			n.Update(-1, excess)
			n = *n.Parent
		}
	} else {
		t.nodesMap[network.String()] = node
	}
	return nil
}

func (t *IPv4Tree) Aggregate(fullness float64) {
	for _, node := range t.Subnets() { // так ли ?
		node.Aggregate(fullness)
	}
}

/*
func (t *IPv4Tree) Search(prefix string) *IPv4TreeNode {
	_, netw, err := net.ParseCIDR(prefix)
	if err != nil {
		return nil
	}
	binPrefix, _ := GetBinaryPathFromIPv4Addr(netw)
	node := t.Root
	ones, _ := netw.Mask.Size()
	for i := 0; i < ones; i++ {
		node = node.Child(string(binPrefix[i]))
		if node == nil {
			return nil
		}
	}
	return node
}

func (t *IPv4Tree) AggregateAll(fullness float64) {
	var dfs func(node *IPv4TreeNode)
	dfs = func(node *IPv4TreeNode) {
		if node == nil {
			return
		}
		for _, child := range node.Children {
			dfs(child)
		}
		node.Aggregate(fullness)
	}
	dfs(t.Root)
}

func (t *IPv4Tree) Length() int {
	return len(t.Subnets())
}

func (t *IPv4Tree) String() string {
	subnets := t.Subnets()
	lines := make([]string, len(subnets))
	for i, node := range subnets {
		lines[i] = node.String()
	}
	return strings.Join(lines, "\n")
}
*/
