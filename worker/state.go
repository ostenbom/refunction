package worker

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
)

type Memory struct {
	name          string
	startOffset   int64
	endOffset     int64
	processOffset int64
	permissions   string
	majorDevice   int
	minorDevice   int
	iNode         int
}

type State struct {
	registers       syscall.PtraceRegs
	memoryLocations []*Memory
}

func NewMemory(memoryData []string) (*Memory, error) {
	var name string
	if len(memoryData) >= 6 {
		name = memoryData[5]
	} else {
		name = ""
	}

	// Offset format 55b4969c1000-55b4969c2000
	offsets := strings.Split(memoryData[0], "-")
	// Device format fc:00
	devices := strings.Split(memoryData[3], ":")

	startOffset, err := strconv.ParseInt(offsets[0], 16, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse offset int: %s", err)
	}

	endOffset, err := strconv.ParseInt(offsets[1], 16, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse offset int: %s", err)
	}

	processOffset, err := strconv.ParseInt(memoryData[2], 16, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse offset int: %s", err)
	}

	majorDevice, err := strconv.ParseInt(devices[0], 16, 0)
	if err != nil {
		return nil, fmt.Errorf("could not parse device int: %s", err)
	}

	minorDevice, err := strconv.ParseInt(devices[1], 16, 0)
	if err != nil {
		return nil, fmt.Errorf("could not parse device int: %s", err)
	}

	iNode, err := strconv.Atoi(memoryData[4])
	if err != nil {
		return nil, fmt.Errorf("could not parse iNode int: %s", err)
	}

	return &Memory{
		name:          name,
		startOffset:   startOffset,
		endOffset:     endOffset,
		processOffset: processOffset,
		permissions:   memoryData[1],
		majorDevice:   int(majorDevice),
		minorDevice:   int(minorDevice),
		iNode:         iNode,
	}, nil
}
