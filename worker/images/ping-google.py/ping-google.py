import time
import os

print("starting google ping", flush=True)

while True:
    result = os.popen(' '.join(("ping", "-c", "1", "216.58.199.68"))).read()
    print("got response", result, flush=True)
    time.sleep(0.2)

print("exiting", flush=True)
