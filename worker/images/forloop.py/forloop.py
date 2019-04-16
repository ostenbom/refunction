import time
import os
import signal
import socket
import select

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

while not activated:
    print("sending signal")
    os.kill(os.getpid(), signal.SIGUSR2)
    time.sleep(0.01)

f = open("/tmp/count.txt", "a")
count = 0
while True:
    f.write("at: " + str(count) + "\n" )
    f.flush()
    print("at:", count, flush=True)
    count += 1
    time.sleep(0.02)

