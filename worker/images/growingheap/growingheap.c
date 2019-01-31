#include <time.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

int main() {
  struct timespec wait;
  wait.tv_sec = 0;
  wait.tv_nsec = 50000000L; // 0.05s

  char *str;
  while (1) {
    printf("at: %p\n", sbrk(0));

    // Trigger heap changes
    str = malloc(10000 * sizeof(char));
    str[0] = 'm';

    nanosleep(&wait, NULL);
  }
}
