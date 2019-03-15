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
#include "Python.h"
#include "cJSON.h"

#define JSON_BUFFER_MAX_SIZE 4096

volatile sig_atomic_t usr_interrupt = 0;

void
synch_signal (int sig) {
  usr_interrupt = 1;
}

void nothing(int sig) {
}

void log_line(char*);
void error_line(char*);
cJSON* recv_json();

void activate();

int main(int argc, char *argv[]) {
  char pidstring[10];
  sprintf(pidstring, "%d", getpid());
  log_line(pidstring);

  activate();
  log_line("activated");

  wchar_t *program = Py_DecodeLocale(argv[0], NULL);
  if (program == NULL) {
      fprintf(stderr, "Fatal error: cannot decode argv[0]\n");
      exit(1);
  }
  Py_SetProgramName(program);  /* optional but recommended */
  Py_Initialize();
  log_line("python started");

  /* Alert checkpoint */
  raise(SIGUSR1);
  log_line("post checkpoint");
  cJSON *function_json = recv_json();
  cJSON *handler = cJSON_GetObjectItemCaseSensitive(function_json, "handler");
  char *handler_string = cJSON_Print(handler);
  log_line(handler_string);

  PyRun_SimpleString("from time import time,ctime\n"
                     "print('Today is', ctime(time()))\n");
  Py_Finalize();
  PyMem_RawFree(program);
  return 0;

  /* printf("done\n"); */
  /* // Done */
  /* raise(SIGUSR2); */
  /* while (1) { */
  /*   raise(SIGUSR1); */
  /* } */
}

void error_line(char* errorline) {
  char *log_string = NULL;
  cJSON *log = cJSON_CreateObject();

  cJSON_AddStringToObject(log, "type", "error");
  cJSON_AddStringToObject(log, "data", errorline);

  log_string = cJSON_Print(log);
  printf("%s\n", log_string);
  free(log_string);
  fflush(stdout);
}

void log_line(char* logline) {
  char *log_string = NULL;
  cJSON *log = cJSON_CreateObject();

  cJSON_AddStringToObject(log, "type", "log");
  cJSON_AddStringToObject(log, "data", logline);

  log_string = cJSON_Print(log);
  printf("%s\n", log_string);
  free(log_string);
  fflush(stdout);
}

cJSON* recv_json() {
  char* buffer = NULL;
  size_t buffsize = JSON_BUFFER_MAX_SIZE;
  /* size_t readchars; */

  buffer = (char*)malloc(buffsize * sizeof(char));
  if (buffer == NULL) {
    error_line("could not allocate recv_json buffer");
    return NULL;
  }

  getline(&buffer, &buffsize, stdin);

  cJSON *parsed_json = cJSON_Parse(buffer);


  return parsed_json;
}

void activate() {
  struct sigaction usr_action1;
  struct sigaction usr_action2;

  sigemptyset(&usr_action1.sa_mask);
  sigemptyset(&usr_action2.sa_mask);
  usr_action1.sa_handler = synch_signal;
  usr_action2.sa_handler = nothing;
  usr_action1.sa_flags = SA_NODEFER;
  usr_action2.sa_flags = SA_NODEFER;

  sigaction (SIGUSR1, &usr_action1, NULL);
  sigaction (SIGUSR2, &usr_action2, NULL);

  sigset_t mask, oldmask;

  struct timespec signalwait;
  signalwait.tv_sec = 0;
  signalwait.tv_nsec = 1000000L; // 1ms

  sigfillset(&mask);
  sigdelset(&mask, SIGUSR1);
  sigdelset(&mask, SIGUSR2);

  sigprocmask (SIG_SETMASK, &mask, &oldmask);
  while (!usr_interrupt) {
    raise(SIGUSR2);
    nanosleep(&signalwait, NULL);
  }
  sigprocmask (SIG_SETMASK, &oldmask, NULL);
}
