import time
start_time = time.time()
import gc
gc.disable()
import os
import json
import select
import signal
import socket
import importlib

HOSTPORT = 5000

activated = False
def activate(signum, frame):
    global activated
    global prevMask
    activated = True
    signal.pthread_sigmask(signal.SIG_SETMASK, prevMask)

def nothing(signum, frame):
    pass

finished = False
def finish(signum, frame):
    global finished
    finished = True

prevMask = signal.pthread_sigmask(signal.SIG_BLOCK, [])
block = set(signal.Signals) - {signal.SIGUSR1, signal.SIGUSR2}
signal.pthread_sigmask(signal.SIG_BLOCK, list(block))

signal.signal(signal.SIGUSR1, activate)
signal.signal(signal.SIGUSR2, nothing)

ready_time = time.time()
print("ready time", abs(start_time - ready_time), "at", time.time(), flush=True)
while not activated:
    os.kill(os.getpid(), signal.SIGUSR2)
    time.sleep(0.01)

signal.signal(signal.SIGUSR1, nothing)
signal.signal(signal.SIGUSR2, finish)
print("activated", flush=True)
# At this point the container is traced and ready to go.

def main():
    alertCheckpoint()
    print("checkpoint taken", flush=True)

    functionLoaded = False
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    print("socket", flush=True)
    # s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    s.bind(('', HOSTPORT))
    print("bind", flush=True)
    s.listen()
    print("listen", flush=True)
    while not functionLoaded:
        print("loading function", id(loadFunction), id(s), flush=True)
        functionLoaded = loadFunction(s)

    print("starting function server", flush=True)
    startFunctionServer(s)
    s.close()
    alertDone()

    while True:
        alertCheckpoint()

def alertCheckpoint():
    os.kill(os.getpid(), signal.SIGUSR1)

def alertDone():
    print("alerting done", flush=True)
    os.kill(os.getpid(), signal.SIGUSR2)

def loadFunction(s):
    print("getting function json", id(getFunctionJson), id(s), flush=True)
    function = getFunctionJson(s)
    print("got function json", flush=True)

    if "imports" in function:
        print("requested to import", function["imports"], flush=True)

    if "handler" not in function:
        print("no handler from function", flush=True)
        return False

    print("loading handle function", function["handler"], flush=True)
    global handle
    exec(function["handler"], globals())
    print("execed handler", flush=True)
    return True

def startFunctionServer(s):
    global finished
    while not finished:
        readready, _, _ = select.select([s], [], [], 0.01)
        if len(readready):
            conn, addr = s.accept()
            request = ''
            try:
                request = decodeSocketJson(conn)
            except ValueError as e:
                print("could not get request from socket", e, flush=True)
                conn.close()
                continue

            print("received request:", request, flush=True)
            response = handle(request)

            print("sending response:", response, flush=True)
            conn.send(json.dumps({"type": "response","data": response}).encode("utf-8"))
            conn.recv(1)
            # try:
            #     conn.shutdown(socket.SHUT_RDWR)
            # except OSError as e:
            #     pass
            conn.close()

def getFunctionJson(s):
    print("waiting function json", id(s), flush=True)
    conn, addr = s.accept()
    print("accepting function json", flush=True)
    function_json = decodeSocketJson(conn)
    print("decoded json", flush=True)

    response = json.dumps({"success": True}).encode("utf-8")
    print("sending response", flush=True)
    conn.send(response)
    print("waiting close")
    conn.recv(1)
    # try:
    #     conn.shutdown(socket.SHUT_RDWR)
    # except OSError as e:
    #     pass
    conn.close()
    return function_json

# Takes an accepted connection, decodes until well-formed json
def decodeSocketJson(conn):
    print("decoding json", flush=True)
    CHUNK_SIZE = 1024

    total_data = []
    loaded_data = ''
    jsonDecoded = False
    while not jsonDecoded:
        print("continuing decoding json", flush=True)
        data = conn.recv(CHUNK_SIZE)
        print("conn recv", flush=True)
        if not data:
            raise ValueError("connection did not send complete json")
        print("appending data", flush=True)
        print(hex(id(total_data)))
        print(total_data)
        total_data.append(data)
        try:
            print("decoding", flush=True)
            decoded = [x.decode("utf-8") for x in total_data]
            loaded_data = json.loads(''.join(decoded))
            print("decoded", flush=True)
            jsonDecoded = True
        except json.JSONDecodeError as e:
            continue

    return loaded_data

main()
