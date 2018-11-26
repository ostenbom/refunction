#include <sys/types.h>
#include <sys/ptrace.h>
#include <unistd.h>
#include <stdio.h>

int main(int charc, char **argv) {
  ptrace(PTRACE_TRACEME);
  while (1) {
    printf("sleeping for 5\n");
    sleep(5);
  }
}
