#include <time.h>
#include <stdio.h>

int main() {
  struct timespec wait;
  wait.tv_sec = 0;
  wait.tv_nsec = 50000000L; // 0.05s or 50ms

  int i = 0;
  while (1) {
    i++;
    printf("at: %i\n", i);

    nanosleep(&wait, NULL);
  }
}
