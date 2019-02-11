import time
start_time = time.time()

import os
import signal
import socket
import select

prevMask = signal.pthread_sigmask(signal.SIG_BLOCK, [])
block = set(signal.Signals) - {signal.SIGUSR1}
signal.pthread_sigmask(signal.SIG_BLOCK, list(block))

activated = False
def activate(signum, frame):
    global activated
    global prevMask
    activated = True
    signal.pthread_sigmask(signal.SIG_SETMASK, prevMask)

signal.signal(signal.SIGUSR1, activate)

s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.bind(('', 5000))
s.listen(1)

ready_time = time.time()
print("ready time", abs(start_time - ready_time), "at", time.time(), flush=True)
while not activated:
    readready, _, _ = select.select([s], [], [], 0.001)
    if len(readready):
        conn, addr = s.accept()
        data = conn.recv(20)
        if data:
            conn.send(data)
        conn.close()

s.close()

f = open("/tmp/count.txt", "a")
count = 0
while True:
    f.write("at: " + str(count) + "\n" )
    f.flush()
    print("at:", count, flush=True)
    count += 1
    time.sleep(0.05)

