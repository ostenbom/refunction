#include <stdio.h>
#include <unistd.h>

int main() {
  fprintf(stderr, "error!\n");
  sleep(10);
}
