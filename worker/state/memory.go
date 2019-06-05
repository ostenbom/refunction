package state

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

type Memory struct {
	name          string
	startOffset   int64
	endOffset     int64
	processOffset int64
	readable      bool
	writable      bool
	executable    bool
	shared        bool
	majorDevice   int
	minorDevice   int
	iNode         int
	content       []byte
}

func newMemoryLocations(pid int) ([]*Memory, error) {
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

func (s *State) SaveWritablePages() error {
	memoryFile, err := os.OpenFile(fmt.Sprintf("/proc/%d/mem", s.pid), os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("could not open /proc/pid/mem: %s", err)
	}
	defer memoryFile.Close()

	for _, memory := range s.memoryLocations {
		if !memory.writable {
			continue
		}

		numBytes := memory.endOffset - memory.startOffset
		memory.content = make([]byte, numBytes)

		read, err := memoryFile.ReadAt(memory.content, memory.startOffset)
		if err != nil || int64(read) != numBytes {
			return fmt.Errorf("could not read /proc/pid/mem data: %s", err)
		}
	}

	return nil
}

func (s *State) MemorySize() int {
	total := 0
	for _, m := range s.memoryLocations {
		total += len(m.content)
	}

	return total
}

func (s *State) ProgramBreakChanged() (bool, error) {
	newMemory, err := newMemoryLocations(s.pid)
	if err != nil {
		return true, fmt.Errorf("could not get new memory on program break changed check: %s", err)
	}

	beforeHeap, err := s.getMemory("[heap]")
	if err != nil {
		return true, err
	}

	var afterHeap *Memory
	for i := range newMemory {
		if newMemory[i].name == "[heap]" {
			afterHeap = newMemory[i]
		}
	}
	if afterHeap == nil {
		// Should be impossible at this point
		return true, fmt.Errorf("no heap in new memory on break change check")
	}

	if beforeHeap.endOffset != afterHeap.endOffset {
		return true, nil
	}

	return false, nil
}

func (s *State) MemoryChanged() (bool, error) {
	newMemory, err := newMemoryLocations(s.pid)
	if err != nil {
		return true, fmt.Errorf("could not get new memory on memory changed check: %s", err)
	}
	if len(s.memoryLocations) != len(newMemory) {
		return true, nil
	}

	for i := range newMemory {
		newMem := newMemory[i]
		oldMem := s.memoryLocations[i]
		if newMem.name != oldMem.name {
			return true, nil
		}
		if newMem.startOffset != oldMem.startOffset {
			return true, nil
		}
		if newMem.endOffset != oldMem.endOffset {
			return true, nil
		}
		if newMem.processOffset != oldMem.processOffset {
			return true, nil
		}
		if newMem.readable != oldMem.readable {
			return true, nil
		}
		if newMem.writable != oldMem.writable {
			return true, nil
		}
		if newMem.executable != oldMem.executable {
			return true, nil
		}
		if newMem.shared != oldMem.shared {
			return true, nil
		}
		if newMem.majorDevice != oldMem.majorDevice {
			return true, nil
		}
		if newMem.minorDevice != oldMem.minorDevice {
			return true, nil
		}
		if newMem.iNode != oldMem.iNode {
			return true, nil
		}
	}

	return false, nil
}

func (s *State) NumMemoryLocationsChanged() (bool, error) {
	newMemory, err := newMemoryLocations(s.pid)
	if err != nil {
		return true, fmt.Errorf("could not get new memory on memory locations changed check: %s", err)
	}
	if len(s.memoryLocations) != len(newMemory) {
		return true, nil
	}
	return false, nil
}

func (s *State) RestoreDirtyPages() error {

	memoryFile, err := os.OpenFile(fmt.Sprintf("/proc/%d/mem", s.pid), os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("could not open /proc/pid/mem: %s", err)
	}
	defer memoryFile.Close()

	var wg sync.WaitGroup

	for _, memory := range s.memoryLocations {
		if !memory.writable {
			continue
		}

		wg.Add(1)
		go func(memory *Memory) {
			defer wg.Done()
			err := s.singleMemoryRestoreDirtyPages(memory, memoryFile)
			if err != nil {
				log.Error(err)
			}
		}(memory)
	}

	wg.Wait()

	return nil
}

func (s *State) singleMemoryRestoreDirtyPages(memory *Memory, memoryFile *os.File) error {
	// 64-bit entries
	pagemapEntrySize := 8
	pageSize := int64(os.Getpagesize())

	startPage := memory.startOffset / pageSize
	endPage := memory.endOffset / pageSize
	numPages := endPage - startPage
	pagemapStartOffset := startPage * int64(pagemapEntrySize)

	var dirty int
	var wg sync.WaitGroup
	parallelism := int64(8)
	var batchSize int64 = numPages / parallelism
	var remainder int64 = numPages % parallelism
	for i := int64(0); i < parallelism; i++ {
		wg.Add(1)
		startIndex := i * batchSize
		endIndex := startIndex + batchSize
		if i == parallelism-1 {
			endIndex = endIndex + remainder
		}
		go func() {
			pagemap, err := os.OpenFile(fmt.Sprintf("/proc/%d/pagemap", s.pid), os.O_RDONLY, os.ModePerm)
			if err != nil {
				log.Errorf("could not open pid %d pagemap: %s", s.pid, err)
			}
			defer pagemap.Close()

			_, err = pagemap.Seek(pagemapStartOffset+(startIndex*int64(pagemapEntrySize)), 0)
			if err != nil {
				log.Errorf("could not seek pid %d pagemap: %s", s.pid, err)
			}

			var sectionOffset = startIndex * pageSize
			var currentPageOffset = memory.startOffset + sectionOffset
			var currentByteNum = int(sectionOffset)
			entryBytes := make([]byte, pagemapEntrySize)
			for i := startIndex; i < endIndex; i++ {

				read, err := pagemap.Read(entryBytes)
				if err != nil || read != pagemapEntrySize {
					log.Errorf("could not read pid %d pagemap: %s", s.pid, err)
				}

				// 55th bit is soft/dirty bit. Arch is little-endian
				dirtySet := entryBytes[6] >> 7
				if dirtySet == byte(1) {
					dirty++
					thisPage := memory.content[currentByteNum : currentByteNum+int(pageSize)]
					read, err = memoryFile.WriteAt(thisPage, currentPageOffset)
					if err != nil || int64(read) != pageSize {
						log.Errorf("could not read pid /proc/%d/map: %s", s.pid, err)
					}
					// spew.Dump(thisPage)
				}

				currentPageOffset += pageSize
				currentByteNum += int(pageSize)
			}

			wg.Done()
		}()

	}

	wg.Wait()

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
		readable:      memoryData[1][0] == 'r',
		writable:      memoryData[1][1] == 'w',
		executable:    memoryData[1][2] == 'x',
		shared:        memoryData[1][3] == 's',
		majorDevice:   int(majorDevice),
		minorDevice:   int(minorDevice),
		iNode:         iNode,
	}, nil
}
