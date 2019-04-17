#include <unistd.h>
#include <stdio.h>
#include <sys/ptrace.h>
#include <sys/wait.h>
#include <sys/types.h>
#include <stropts.h>
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
volatile sig_atomic_t server_finish = 0;

void synch_signal(int sig) {
  usr_interrupt = 1;
}

void start_server() {
  server_finish = 0;
}

void end_server(int sig) {
  server_finish = 1;
}

void nothing(int sig) {

}

void log_line(char*);

int stdinHasInput()
{
  struct timeval tv;
  fd_set fds;
  tv.tv_sec = 0;
  tv.tv_usec = 10000; // 10ms

  FD_ZERO(&fds);
  FD_SET(STDIN_FILENO, &fds);
  int selectresp = select(1, &fds, NULL, NULL, &tv);
  if (selectresp < 0) {
    log_line("bad select response");
    return -1;
  }

  return (FD_ISSET(STDIN_FILENO, &fds));
}

void send_function_loaded();
void send_response(char*);
void error_line(char*);
cJSON* recv_json();

void activate();
void start_python(char*);
PyObject* import_module(char*);

int main(int argc, char *argv[]) {
  char pidstring[50];
  sprintf(pidstring, "Pid: %d", getpid());
  log_line(pidstring);

  activate();
  log_line("activated");

  start_python(argv[0]);
  log_line("python started");

  PyObject *json_module = import_module("json");
  log_line("imported json");

  // TODO: Do this before or after checkpoint??
  PyObject *phandle_func, *pvalue;
  PyObject *pglobal = PyDict_New();
  PyObject *handle_module = PyModule_New("handler");
  PyModule_AddStringConstant(handle_module, "__file__", "");
  PyObject *plocal = PyModule_GetDict(handle_module);
  PyObject *builtins = PyEval_GetBuiltins();
  PyDict_SetItemString(pglobal, "__builtins__", builtins);

  /* Alert checkpoint */
  raise(SIGUSR1);
  log_line("post checkpoint");

  log_line("starting function json load");
  /* Receive handler function string */
  cJSON *function_json = recv_json();
  cJSON *handler = cJSON_GetObjectItemCaseSensitive(function_json, "handler");
  char *handler_string = handler->valuestring;
  log_line(handler_string);

  /* Load handler function into module */
  pvalue = PyRun_String(handler_string, Py_file_input, pglobal, plocal);
  if (pvalue == NULL) {
      if (PyErr_Occurred()) {
        PyErr_Print();
      }
      fprintf(stderr, "Error: could not load handle function\n");
      exit(1);
  }
  Py_DECREF(pvalue);
  cJSON_Delete(function_json);

  phandle_func = PyObject_GetAttrString(handle_module, "handle");

  if (!phandle_func || !PyCallable_Check(phandle_func)) {
      if (PyErr_Occurred()) {
        PyErr_Print();
      }
      fprintf(stderr, "Error: obtain handle function from module\n");
      exit(1);
  }

  log_line("handle function successfully loaded");
  send_function_loaded();

  start_server();
  while (!server_finish) {
    if (stdinHasInput() <= 0) {
      continue;
    }

    cJSON *req_json = recv_json();
    // TODO: Check if json is type request
    cJSON *request_data = cJSON_GetObjectItemCaseSensitive(req_json, "data");
    char *request_data_string = request_data->valuestring;
    log_line(request_data_string);

    // json.loads(request)
    PyObject *pjson_request, *pjson_call_args, *pjson_loads, *prequest;
    pjson_request = PyUnicode_FromString(request_data_string);
    pjson_call_args = PyTuple_New(1);
    PyTuple_SetItem(pjson_call_args, 0, pjson_request);

    pjson_loads = PyObject_GetAttrString(json_module, "loads");
    prequest = PyObject_CallObject(pjson_loads, pjson_call_args);
    Py_XDECREF(pjson_loads);
    if (prequest == NULL) {
      log_line("failure when loading request json");
      PyErr_Print();
      fflush(stderr);
      exit(1);
    }

    log_line("json loaded");

    // Create args for handle func
    PyObject *phandle_args;
    phandle_args = PyTuple_New(1);
    PyTuple_SetItem(phandle_args, 0, prequest);

    // Call handle func
    PyObject *presponse;
    presponse = PyObject_CallObject(phandle_func, phandle_args);
    if (presponse == NULL) {
      log_line("failure in handle call");
      PyErr_Print();
      fflush(stderr);
      exit(1);
    }

    log_line("handle called");

    // json.dumps(response)
    PyObject *pjson_response, *pjson_dumps;
    pjson_dumps = PyObject_GetAttrString(json_module, "dumps");
    PyTuple_SetItem(pjson_call_args, 0, presponse);
    pjson_response = PyObject_CallObject(pjson_dumps, pjson_call_args);
    Py_XDECREF(pjson_dumps);

    log_line("json dumped");

    PyObject *ascii_response;
    ascii_response = PyUnicode_AsASCIIString(pjson_response);
    send_response(PyBytes_AsString(ascii_response));

    Py_DECREF(ascii_response);
    Py_DECREF(presponse);
    Py_DECREF(pjson_response);
    Py_DECREF(phandle_args);
    Py_DECREF(pjson_call_args);
    Py_DECREF(pjson_request);
  }

  log_line("finished server");

  // Done
  raise(SIGUSR2);
  struct timespec finishedwait;
  finishedwait.tv_sec = 0;
  finishedwait.tv_nsec = 10000000L; // 10ms
  while (1) {
    nanosleep(&finishedwait, NULL);
  }
  /* Py_Finalize(); */
  /* PyMem_RawFree(program); */
  /* return 0; */
}

void send_function_loaded() {
  char *log_string = NULL;
  cJSON *log = cJSON_CreateObject();
  cJSON_AddStringToObject(log, "type", "function_loaded");
  cJSON_AddStringToObject(log, "data", "");

  log_string = cJSON_PrintUnformatted(log);
  printf("%s\n", log_string);
  free(log_string);
  fflush(stdout);
  cJSON_Delete(log);
}

void send_response(char* response) {
  char *log_string = NULL;
  cJSON *log = cJSON_CreateObject();
  cJSON *jresponse = cJSON_Parse(response);

  cJSON_AddStringToObject(log, "type", "response");
  cJSON_AddItemToObject(log, "data", jresponse);

  log_string = cJSON_PrintUnformatted(log);
  printf("%s\n", log_string);
  free(log_string);
  fflush(stdout);
  cJSON_Delete(log);
}

void error_line(char* errorline) {
  char *log_string = NULL;
  cJSON *log = cJSON_CreateObject();

  cJSON_AddStringToObject(log, "type", "error");
  cJSON_AddStringToObject(log, "data", errorline);

  log_string = cJSON_PrintUnformatted(log);
  printf("%s\n", log_string);
  free(log_string);
  fflush(stdout);
  cJSON_Delete(log);
}

void log_line(char* logline) {
  char *log_string = NULL;
  cJSON *log = cJSON_CreateObject();

  cJSON_AddStringToObject(log, "type", "log");
  cJSON_AddStringToObject(log, "data", logline);

  log_string = cJSON_PrintUnformatted(log);
  printf("%s\n", log_string);
  free(log_string);
  fflush(stdout);
  cJSON_Delete(log);
}

void start_python(char* program_name) {
  wchar_t *program = Py_DecodeLocale(program_name, NULL);
  if (program == NULL) {
      fprintf(stderr, "Fatal error: cannot decode argv[0]\n");
      exit(1);
  }
  Py_SetProgramName(program);  /* optional but recommended */
  Py_Initialize();
}

PyObject* import_module(char* str_name) {
  PyObject *module, *module_name;
  module_name = PyUnicode_DecodeFSDefault(str_name);
  module = PyImport_Import(module_name);
  if (module == NULL) {
      fprintf(stderr, "Error: cannot import %s module\n", str_name);
      exit(1);
  }
  Py_DECREF(module_name);
  return module;
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
  free(buffer);
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

  struct sigaction stop_server_action;
  sigemptyset(&stop_server_action.sa_mask);
  stop_server_action.sa_handler = end_server;
  stop_server_action.sa_flags = SA_NODEFER;
  sigaction(SIGUSR2, &stop_server_action, NULL);
}
