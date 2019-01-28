#include <time.h>
#include <stdio.h>
#include <stdlib.h>

int main() {
  struct timespec wait;
  wait.tv_sec = 0;
  wait.tv_nsec = 50000000L; // 0.05s

  int i = 0;
  char *str;
  while (1) {
    i++;
    printf("at: %i\n", i);

    // Trigger heap changes
    str = malloc(1000 * sizeof(char));
    str[0] = 'm';
    free(str);

    nanosleep(&wait, NULL);
  }
}
