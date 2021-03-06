#include <unistd.h>
#include <stdio.h>
#include <sys/ptrace.h>
#include <sys/wait.h>
#include <sys/types.h>
#include <unistd.h>
#include <stdlib.h>
#include <stdio.h>
#include <errno.h>
#include <string.h>
#include <signal.h>
#include <time.h>


volatile sig_atomic_t usr_interrupt = 0;

void
synch_signal (int sig) {
  usr_interrupt = 1;
}

int main() {
  printf("starting\n");

  struct sigaction usr_action;
  sigset_t block_mask;

  sigfillset (&block_mask);
  usr_action.sa_handler = synch_signal;
  usr_action.sa_mask = block_mask;
  sigaction (SIGUSR1, &usr_action, NULL);

  sigset_t mask, oldmask;

  sigemptyset (&mask);
  sigaddset (&mask, SIGUSR1);

  sigprocmask (SIG_BLOCK, &mask, &oldmask);
  while (!usr_interrupt)
    sigsuspend (&oldmask);
  sigprocmask (SIG_UNBLOCK, &mask, NULL);

  printf("I have been USR sigg'd\n");

  FILE *fp;
  fp = fopen("/tmp/count.txt", "w+");

  struct timespec wait;
  wait.tv_sec = 0;
  wait.tv_nsec = 10000L;

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
