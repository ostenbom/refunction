import threading
import time

class PrintThread(threading.Thread):
    def __init__(self):
        threading.Thread.__init__(self)

    def run(self):
        f = open("/tmp/count.txt", "a")
        count = 0
        while True:
            f.write("at: " + str(count) + "\n" )
            f.flush()
            print("at:", count, flush=True)
            count += 1
            time.sleep(0.02)

thread = PrintThread()
thread.start()
print("active threads:", threading.active_count(), flush=True)
thread.join()

