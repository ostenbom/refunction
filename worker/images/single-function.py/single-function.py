import time
start_time = time.time()
import os
import json
import signal
import socket
import importlib

HOSTPORT = 5000

prevMask = signal.pthread_sigmask(signal.SIG_BLOCK, [])
block = set(signal.Signals) - {signal.SIGUSR1, signal.SIGUSR2}
signal.pthread_sigmask(signal.SIG_BLOCK, list(block))

activated = False
def activate(signum, frame):
    global activated
    global prevMask
    activated = True
    signal.pthread_sigmask(signal.SIG_SETMASK, prevMask)

def nothing(signum, frame):
    pass

signal.signal(signal.SIGUSR1, activate)
signal.signal(signal.SIGUSR2, nothing)

ready_time = time.time()
print("ready time", abs(start_time - ready_time), "at", time.time(), flush=True)
while not activated:
    os.kill(os.getpid(), signal.SIGUSR2)
    time.sleep(0.01)

signal.signal(signal.SIGUSR1, nothing)
signal.signal(signal.SIGUSR2, nothing)
print("activated", flush=True)
# At this point the container is traced and ready to go.

def main():
    alertCheckpoint()

    handle({"success": True})

    alertDone()

def handle(req):
    print("handling request", req, flush=True)
    return req

def alertDone():
    os.kill(os.getpid(), signal.SIGUSR1)

def alertCheckpoint():
    os.kill(os.getpid(), signal.SIGUSR2)

main()
