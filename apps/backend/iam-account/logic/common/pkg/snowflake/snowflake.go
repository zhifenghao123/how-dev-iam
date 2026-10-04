// Package snowflake 提供最小可用的雪花 ID 生成器。
//
// 布局（63 位有效，最高位符号位固定 0）：
//
//	timestamp(41) | node(10) | sequence(12)
//
// - timestamp：毫秒时间戳相对 epoch 的偏移
// - node：节点 ID，单进程内固定，从配置读入或使用默认值 1
// - sequence：同一毫秒内的自增序列，溢出时忙等到下一毫秒
//
// 单节点单进程内并发安全。多节点部署时请在配置中为每个进程分配不同 node id。
package snowflake

import (
	"errors"
	"sync"
	"time"
)

const (
	epoch          int64 = 1_704_067_200_000 // 2024-01-01 00:00:00 UTC (ms)
	nodeBits             = 10
	sequenceBits         = 12
	maxNode              = -1 ^ (-1 << nodeBits)     // 1023
	maxSequence          = -1 ^ (-1 << sequenceBits) // 4095
	timestampShift       = nodeBits + sequenceBits
	nodeShift            = sequenceBits
)

// Node 是一个雪花 ID 生成器。
type Node struct {
	mu       sync.Mutex
	node     int64
	lastTS   int64
	sequence int64
}

// NewNode 构造节点；node ∈ [0, 1023]。
func NewNode(node int64) (*Node, error) {
	if node < 0 || node > maxNode {
		return nil, errors.New("snowflake: node id out of range")
	}
	return &Node{node: node}, nil
}

// MustNewNode 构造节点，非法时 panic。
func MustNewNode(node int64) *Node {
	n, err := NewNode(node)
	if err != nil {
		panic(err)
	}
	return n
}

// Generate 生成一个雪花 ID。
func (n *Node) Generate() int64 {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now().UnixMilli()
	if now == n.lastTS {
		n.sequence = (n.sequence + 1) & maxSequence
		if n.sequence == 0 {
			// 序列耗尽，忙等到下一毫秒。
			for now <= n.lastTS {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		n.sequence = 0
	}
	n.lastTS = now

	return ((now - epoch) << timestampShift) | (n.node << nodeShift) | n.sequence
}

// ===== 全局默认节点（供 service 层直接使用）=====

var defaultNode = MustNewNode(1)

// Next 使用全局默认节点生成 ID。
func Next() int64 {
	return defaultNode.Generate()
}
