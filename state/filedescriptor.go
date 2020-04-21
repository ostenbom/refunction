package state

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

type FileDescriptor struct {
	name   string
	link   string
	fdInfo string
}

func newFileDescriptors(pid int) ([]*FileDescriptor, error) {
	var fileDescriptors []*FileDescriptor

	fdDir, err := os.Open(fmt.Sprintf("/proc/%d/fd", pid))
	if err != nil {
		return nil, fmt.Errorf("could not open fd directory: %s", err)
	}
	defer fdDir.Close()

	fdInfoDir, err := os.Open(fmt.Sprintf("/proc/%d/fdinfo", pid))
	if err != nil {
		return nil, fmt.Errorf("could not open fd info directory: %s", err)
	}
	defer fdInfoDir.Close()

	fdFiles, err := fdDir.Readdirnames(0)
	if err != nil {
		return nil, fmt.Errorf("could not read fd directory: %s", err)
	}
	fdInfoFiles, err := fdInfoDir.Readdirnames(0)
	if err != nil {
		return nil, fmt.Errorf("could not read fd info directory: %s", err)
	}

	// Sanity check
	if len(fdFiles) != len(fdInfoFiles) {
		return nil, errors.New("not the same amount of fd and fdinfos")
	}

	for i := range fdFiles {
		if fdFiles[i] != fdInfoFiles[i] {
			return nil, errors.New("fd file names do not match")
		}
		var fileDescriptor FileDescriptor

		name := fdFiles[i]
		fileDescriptor.name = name

		link, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/%s", pid, name))
		if err != nil {
			return nil, fmt.Errorf("could not read fd link: %s", err)
		}
		fileDescriptor.link = link

		fdInfo, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/fdinfo/%s", pid, name))
		if err != nil {
			return nil, fmt.Errorf("could not open fdinfo file: %s", err)
		}
		fileDescriptor.fdInfo = string(fdInfo)

		fileDescriptors = append(fileDescriptors, &fileDescriptor)
	}

	return fileDescriptors, nil
}

func (s *State) GetFileDescriptors() []*FileDescriptor {
	return s.fileDescriptors
}

func (s *State) FdsChanged() (bool, error) {
	newDescriptors, err := newFileDescriptors(s.pid)
	if err != nil {
		return true, fmt.Errorf("could not get new descriptors on change check: %s", err)
	}

	if len(newDescriptors) != len(s.fileDescriptors) {
		return true, nil
	}

	return false, nil
}
