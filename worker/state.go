package worker

import (
	"bufio"
	"fmt"
	"os"
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

func NewMemoryLocations(pid int) ([]*Memory, error) {
	var memoryLocations []*Memory

	maps, err := os.Open(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		return nil, fmt.Errorf("could not open maps file: %s", err)
	}
	defer maps.Close()

	scanner := bufio.NewScanner(maps)
	for scanner.Scan() {
		memoryLine := scanner.Text()
		memoryData := strings.Fields(memoryLine)

		var name string
		if len(memoryData) >= 6 {
			name = memoryData[5]
		} else {
			name = ""
		}

		// These are kernel owned and can be skipped
		if name == "[vvar]" || name == "[vdso]" || name == "[vsyscall]" {
			continue
		}

		memory, err := parseMemoryData(name, memoryData)
		if err != nil {
			return nil, err
		}

		memoryLocations = append(memoryLocations, memory)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not scan maps file: %s", err)
	}

	return memoryLocations, nil
}

func parseMemoryData(name string, memoryData []string) (*Memory, error) {
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
