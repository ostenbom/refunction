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

    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind(('', HOSTPORT))
    s.listen()
    while not functionLoaded:
        print("loading function", flush=True)
        functionLoaded = loadFunction(s)


    print("starting function server", flush=True)
    startFunctionServer(s)
    s.close()

def alertDone():
    os.kill(os.getpid(), signal.SIGUSR1)

def alertCheckpoint():
    os.kill(os.getpid(), signal.SIGUSR2)

def loadFunction(s):
    function = getFunctionJson(s)

    if "imports" in function:
        print("requested to import", function["imports"], flush=True)

    if "handler" not in function:
        print("no handler from function", flush=True)
        return False

    global handle
    exec(function["handler"], globals())
    return True

def startFunctionServer(s):
    while True:
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

        response_string = ''
        try:
            response_string = json.dumps(response)
        except ValueError as e:
            print("could not dump response to json", e, response, flush=True)
            conn.close()
            continue

        print("sending response:", response_string, flush=True)
        conn.send(response_string.encode("utf-8"))
        try:
            conn.shutdown(socket.SHUT_RDWR)
        except OSError as e:
            pass
        conn.close()

def getFunctionJson(s):
    print("waiting function json", flush=True)
    conn, addr = s.accept()
    print("accepting function json", flush=True)
    function_json = decodeSocketJson(conn)
    response = json.dumps({"success": True}).encode("utf-8")
    conn.send(response)
    try:
        conn.shutdown(socket.SHUT_RDWR)
    except OSError as e:
        pass
    conn.close()
    return function_json

# Takes an accepted connection, decodes until well-formed json
def decodeSocketJson(conn):
    CHUNK_SIZE = 1024

    total_data = []
    loaded_data = ''
    jsonDecoded = False
    while not jsonDecoded:
        data = conn.recv(CHUNK_SIZE)
        if not data:
            raise ValueError("connection did not send complete json")
        total_data.append(data)
        try:
            decoded = [x.decode("utf-8") for x in total_data]
            loaded_data = json.loads(''.join(decoded))
            jsonDecoded = True
        except json.JSONDecodeError as e:
            continue

    return loaded_data

main()
