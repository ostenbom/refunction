import signal, os, time

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

while not activated:
    signal.pause()

f = open("/tmp/count.txt", "a")
count = 0
while True:
    f.write("at: " + str(count) + "\n")
    f.flush()
    print("at:", count)
    count += 1
    time.sleep(0.05)

