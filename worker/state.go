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
	content       []byte
}

type State struct {
	pid             int
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

		// TODO: These are kernel owned and can be skipped?
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

func (s *State) getMemory(memoryName string) (*Memory, error) {
	var memory *Memory
	for _, m := range s.memoryLocations {
		if m.name == memoryName {
			memory = m
			break
		}
	}
	if memory == nil {
		return nil, fmt.Errorf("no memory %s found", memoryName)
	}

	return memory, nil
}

func (s *State) SavePages(memoryName string) error {
	memory, err := s.getMemory(memoryName)
	if err != nil {
		return err
	}

	numBytes := memory.endOffset - memory.startOffset
	memory.content = make([]byte, numBytes)

	memoryFile, err := os.OpenFile(fmt.Sprintf("/proc/%d/mem", s.pid), os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("could not open /proc/pid/mem: %s", err)
	}
	defer memoryFile.Close()

	read, err := memoryFile.ReadAt(memory.content, memory.startOffset)
	if err != nil || int64(read) != numBytes {
		return fmt.Errorf("could not read /proc/pid/mem data: %s", err)
	}

	return nil
}

func (s *State) MemorySize(memoryName string) (int, error) {
	memory, err := s.getMemory(memoryName)
	if err != nil {
		return 0, err
	}

	return len(memory.content), nil
}

func (s *State) RestoreDirtyPages(memoryName string) error {
	memory, err := s.getMemory(memoryName)
	if err != nil {
		return err
	}

	pagemap, err := os.OpenFile(fmt.Sprintf("/proc/%d/pagemap", s.pid), os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("could not open pid %d pagemap: %s", s.pid, err)
	}
	defer pagemap.Close()

	memoryFile, err := os.OpenFile(fmt.Sprintf("/proc/%d/mem", s.pid), os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("could not open /proc/pid/mem: %s", err)
	}
	defer memoryFile.Close()

	// 64-bit entries
	pagemapEntrySize := 8
	pageSize := int64(os.Getpagesize())

	startPage := memory.startOffset / pageSize
	endPage := memory.endOffset / pageSize
	numPages := endPage - startPage
	pagemapStartOffset := startPage * int64(pagemapEntrySize)

	_, err = pagemap.Seek(pagemapStartOffset, 0)
	if err != nil {
		return fmt.Errorf("could not seek pid %d pagemap: %s", s.pid, err)
	}

	var dirty int
	var currentPageOffset = memory.startOffset
	var currentByteNum = 0
	entryBytes := make([]byte, pagemapEntrySize)
	for i := int64(0); i < numPages; i++ {

		read, err := pagemap.Read(entryBytes)
		if err != nil || read != pagemapEntrySize {
			return fmt.Errorf("could not read pid %d pagemap: %s", s.pid, err)
		}

		// 55th bit is soft/dirty bit. Arch is little-endian
		dirtySet := entryBytes[6] >> 7
		if dirtySet == byte(1) {
			dirty++
			thisPage := memory.content[currentByteNum : currentByteNum+int(pageSize)]
			read, err = memoryFile.WriteAt(thisPage, currentPageOffset)
			if err != nil || int64(read) != pageSize {
				return fmt.Errorf("could not read pid /proc/%d/map: %s", s.pid, err)
			}
			// spew.Dump(thisPage)
		}

		currentPageOffset += pageSize
		currentByteNum += int(pageSize)
	}

	return nil
}

func (s *State) CountDirtyPages(memoryName string) (int, error) {
	memory, err := s.getMemory(memoryName)
	if err != nil {
		return 0, err
	}

	pagemap, err := os.OpenFile(fmt.Sprintf("/proc/%d/pagemap", s.pid), os.O_RDONLY, os.ModePerm)
	if err != nil {
		return 0, fmt.Errorf("could not open pid %d pagemap: %s", s.pid, err)
	}
	defer pagemap.Close()

	// 64-bit entries
	pagemapEntrySize := 8
	pageSize := int64(os.Getpagesize())

	startPage := memory.startOffset / pageSize
	endPage := memory.endOffset / pageSize
	numPages := endPage - startPage
	pagemapStartOffset := startPage * int64(pagemapEntrySize)

	_, err = pagemap.Seek(pagemapStartOffset, 0)
	if err != nil {
		return 0, fmt.Errorf("could not seek pid %d pagemap: %s", s.pid, err)
	}

	var dirty int
	var currentPageOffset = memory.startOffset
	entryBytes := make([]byte, pagemapEntrySize)
	for i := int64(0); i < numPages; i++ {

		read, err := pagemap.Read(entryBytes)
		if err != nil || read != pagemapEntrySize {
			return 0, fmt.Errorf("could not read pid %d pagemap: %s", s.pid, err)
		}

		// 55th bit is soft/dirty bit. Arch is little-endian
		dirtySet := entryBytes[6] >> 7
		if dirtySet == byte(1) {
			dirty++
		}

		currentPageOffset += pageSize
	}

	return dirty, nil
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
