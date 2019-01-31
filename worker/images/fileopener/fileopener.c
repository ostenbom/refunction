#include <unistd.h>
#include <stdlib.h>
#include <stdio.h>
#include <time.h>

int main() {
  struct timespec *wait = malloc(sizeof(struct timespec));
  wait->tv_sec = 0;
  wait->tv_nsec = 5000000L; // 5ms

  int count = 0;
  while(1) {
    FILE *fp = malloc(sizeof(FILE));
    char filename[50];
    sprintf(filename, "%d.txt", count);
    fp = fopen(filename, "w+");

    printf("at: %i\n", count);
    fprintf(fp, "at: %i\n", count);
    fflush(fp);

    count++;
    nanosleep(wait, NULL);
  }

}
