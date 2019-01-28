#include <unistd.h>
#include <stdlib.h>
#include <stdio.h>
#include <time.h>

int main() {
  printf("starting\n");

  FILE *fp;
  fp = fopen("count.txt", "w+");

  struct timespec wait;
  wait.tv_sec = 0;
  wait.tv_nsec = 500000000L;

  int count = 0;
  while(1) {
    printf("at: %i\n", count);
    fprintf(fp, "at: %i\n", count);
    fflush(fp);
    count++;
    nanosleep(&wait, NULL);
  }

  fclose(fp);
}
