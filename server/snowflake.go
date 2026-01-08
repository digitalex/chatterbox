package main

import (
	"errors"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	nodeBits        = 10
	stepBits        = 12
	nodeMax         = -1 ^ (-1 << nodeBits)
	stepMask        = -1 ^ (-1 << stepBits)
	timeShift       = nodeBits + stepBits
	nodeShift       = stepBits
	epoch           = 1704067200000 // 2024-01-01 00:00:00 UTC
)

// Node struct holds the configuration for the Snowflake generator
type Node struct {
	mu        sync.Mutex
	timestamp int64
	nodeID    int64
	step      int64
}

// Global generator instance
var globalNode *Node

func init() {
	// Initialize with Node ID from env, or default to 1.
	nodeID := int64(1)
	if val := os.Getenv("SNOWFLAKE_NODE_ID"); val != "" {
		if id, err := strconv.ParseInt(val, 10, 64); err == nil {
			nodeID = id
		}
	}

	node, err := NewNode(nodeID)
	if err != nil {
		// Fallback to 1 if configured ID is invalid (e.g. too large)
		// In production, we might want to panic, but for resilience we fallback.
		// However, let's panic to alert misconfiguration.
		panic(err)
	}
	globalNode = node
}

// NewNode creates a new Snowflake node
func NewNode(nodeID int64) (*Node, error) {
	if nodeID < 0 || nodeID > nodeMax {
		return nil, errors.New("node ID too large")
	}
	return &Node{
		timestamp: 0,
		nodeID:    nodeID,
		step:      0,
	}, nil
}

// Generate creates a new unique ID
func (n *Node) Generate() int64 {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := time.Now().UnixMilli()

	if now < n.timestamp {
		// Clock moved backwards, refuse to generate id
		// In production, we might wait or error. Here we wait.
		for now <= n.timestamp {
			now = time.Now().UnixMilli()
		}
	}

	if n.timestamp == now {
		n.step = (n.step + 1) & stepMask
		if n.step == 0 {
			// Sequence exhausted, wait for next millisecond
			for now <= n.timestamp {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		n.step = 0
	}

	n.timestamp = now

	id := ((now - epoch) << timeShift) |
		(n.nodeID << nodeShift) |
		(n.step)

	return id
}

// GenerateID generates a new ID using the global node
func GenerateID() int64 {
	return globalNode.Generate()
}
