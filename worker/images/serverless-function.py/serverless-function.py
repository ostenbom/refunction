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

    functionLoaded = False

    while not functionLoaded:
        print("loading function", flush=True)
        functionLoaded = loadFunction()


    print("starting function server", flush=True)
    startFunctionServer()

def alertDone():
    os.kill(os.getpid(), signal.SIGUSR1)

def alertCheckpoint():
    os.kill(os.getpid(), signal.SIGUSR2)

def loadFunction():
    function = getFunctionJson()

    if "imports" in function:
        print("requested to import", function["imports"], flush=True)

    if "handler" not in function:
        print("no handler from function", flush=True)
        return False

    global handle
    exec(function["handler"], globals())
    return True

def startFunctionServer():
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind(('', HOSTPORT))
    s.listen()
    while True:
        conn, addr = s.accept()
        total_data = []
        while True:
            data = conn.recv(1024)
            if not data:
                break
            total_data.append(data)

        decoded_data = [x.decode("utf-8") for x in total_data]

        request = ''
        try:
            request = json.loads(''.join(decoded_data))
        except ValueError as e:
            print("could not load request to json", e, request, flush=True)
            conn.close()
            continue

        print("received request:", request, flush=True)
        response = handle(request)

        response_string = ''
        try:
            response_string = json.dumps(response)
        except ValueError as e:
            print("could not dump response to json", e, response, flush=True)
            conn.close()
            continue

        print("sending response:", response_string, flush=True)
        conn.sendall(response_string.encode("utf-8"))
        conn.close()

def getFunctionJson():
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind(('', HOSTPORT))
    s.listen()
    data = getSocketJson(s)
    s.close()
    return data

def getSocketJson(s):
    print("waiting function json", flush=True)
    conn, addr = s.accept()
    print("accepting function json", flush=True)
    total_data = []
    while True:
        data = conn.recv(1024)
        if not data:
            break
        total_data.append(data)
    conn.close()
    decoded_data = [x.decode("utf-8") for x in total_data]
    return json.loads(''.join(decoded_data))

main()
