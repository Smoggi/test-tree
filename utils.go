package go_ip_trees

import (
	"fmt"
	"math"
	"math/big"
	"net"
	"strconv"
	"strings"
)

// GetBinaryPathFromIPv4Addr returns a 32-bit binary representation of an IPv4 address or network.
func GetBinaryPathFromIPv4Addr(ip interface{}) (string, error) {
	switch v := ip.(type) {
	case net.IP:
		return fmt.Sprintf("%032b", big.NewInt(0).SetBytes(v.To4()).Uint64()), nil
	case *net.IPNet:
		return fmt.Sprintf("%032b", big.NewInt(0).SetBytes(v.IP.To4()).Uint64()), nil
	default:
		return "", fmt.Errorf("bad type %T", v)
	}
}

// GetIPv4FromBinaryString reverts binary string to IPv4 address.
func GetIPv4FromBinaryString(s string) net.IP {
	i, _ := strconv.ParseUint(s, 2, 32)
	return net.IPv4(byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
}

// IPv4SpaceSplit splits IPv4 space into 2^logPartsNum subnets.
func IPv4SpaceSplit(logPartsNum int) []*net.IPNet {
	nets := make([]*net.IPNet, 0)
	repeat := 1 << logPartsNum

	for i := 0; i < repeat; i++ {
		bin := fmt.Sprintf("%0*b", logPartsNum, i)
		fullbin := "0" + bin + strings.Repeat("0", 32-logPartsNum)
		ip := GetIPv4FromBinaryString(fullbin)
		_, netw, _ := net.ParseCIDR(fmt.Sprintf("%s/%d", ip.String(), logPartsNum))
		nets = append(nets, netw)
	}
	return nets
}

func PrefixSize(n int) int {
	return 1 << (32 - n)
}

func CalculateTotalHosts(ipNet *net.IPNet) int {
	// Calculate the number of host bits
	prefixLength, totalBits := ipNet.Mask.Size()
	hostBits := totalBits - prefixLength

	// Calculate the total hosts based on the number of host bits
	totalHosts := int(math.Pow(2, float64(hostBits)))

	return totalHosts
}

func TreeSub(treeA SpecialTree, treeB SpecialTree) (*SpecialTree, error) {
	var sa []string
	for _, node := range treeA.Subnets() {
		if node.IsLast {
			sa = append(sa, node.String())
		}
	}
	var sb []string
	for _, node := range treeB.Subnets() {
		if node.IsLast {
			sb = append(sb, node.String())
		}
	}
	sa = difference(sa, sb)
	res := NewSpecialTree()

	for _, cidr := range sa {
		res.Insert(cidr, nil)
	}
	return res, nil
}

func difference(a, b []string) []string {
	m := make(map[string]bool)
	for _, v := range b {
		m[v] = true
	}
	var res []string
	for _, v := range a {
		if !m[v] {
			res = append(res, v)
		}
	}
	return res
}

func TreeMerge(treeA SpecialTree, treeB SpecialTree) (SpecialTree, error) {
	for _, node := range treeB.Subnets() {
		if node.IsLast {
			treeA.Insert(node.String(), nil)
		}
	}
	return treeA, nil
}

func TreeToList(tree SpecialTree) ([]net.IPNet, error) {
	var res []net.IPNet
	for _, node := range tree.Subnets() {
		if node.IsLast {
			_, network, _ := net.ParseCIDR(node.String())
			res = append(res, *network)
		}
	}
	return res, nil
}
