#include <unistd.h>
#include <stdlib.h>
#include <stdio.h>
#include <time.h>

int main() {
  printf("starting\n");

  FILE *fp = malloc(sizeof(FILE));
  fp = fopen("count.txt", "w+");

  struct timespec *wait = malloc(sizeof(struct timespec));
  wait->tv_sec = 0;
  wait->tv_nsec = 50000000L; // 50ms

  static int count;

  while(1) {
    printf("at: %i\n", count);
    fprintf(fp, "at: %i\n", count);
    fflush(fp);
    count++;
    nanosleep(wait, NULL);
  }

  fclose(fp);
}
