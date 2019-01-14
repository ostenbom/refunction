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

volatile sig_atomic_t usr_interrupt = 0;

void
synch_signal (int sig) {
  usr_interrupt = 1;
}

int main() {
  printf("starting\n");

  int action_success;
  struct sigaction chld_action;

  action_success = sigaction(SIGCHLD, NULL, &chld_action);
  chld_action.sa_flags |= SA_NOCLDWAIT;
  action_success = sigaction(SIGCHLD, &chld_action, NULL);

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
  fp = fopen("count.txt", "w+");

  int count = 0;
  while(1) {
    printf("at: %i\n", count);
    fprintf(fp, "at: %i\n", count);
    fflush(fp);
    count++;
    sleep(1);
  }

  fclose(fp);
}
