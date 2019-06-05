import json
import fileinput
from datetime import datetime

def start_function_server():
    send_data("started", "")

    stdin = fileinput.input()

    loaded = False
    while not loaded:
        try:
            function_string = receive_data_of_type("function", stdin)
            global main
            exec(function_string, globals())
            loaded = True
        except:
            send_data("function_loaded", False)
            continue

    send_data("function_loaded", True)

    while True:
        message_type, data = receive_data(stdin)
        if message_type == "request":
            log(f"received request: {data}")
            result = main(data)
            send_data("response", result)

    # Never finishes. Either killed or restored

def receive_data_of_type(data_type, stdin):
    for line in stdin:
        try:
            data = json.loads(line)
            if "type" not in data or "data" not in data:
                continue
            if data["type"] == data_type:
                return data["data"]
        except json.JSONDecodeError as e:
            continue

def receive_data(stdin):
    for line in stdin:
        try:
            data = json.loads(line)
            if "type" not in data or "data" not in data:
                continue
            return data["type"], data["data"]
        except json.JSONDecodeError as e:
            continue

def send_data(data_type, data):
    action = {'type': data_type, 'data': data}
    asjson = json.dumps(action)
    print(asjson, flush=True)

def log(line):
    log_obj = {"type": "log", "data": line, "time": str(datetime.utcnow())}
    print(json.dumps(log_obj), flush=True)

start_function_server()
