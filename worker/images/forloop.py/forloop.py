import time

f = open("/tmp/count.txt", "a")
count = 0
while True:
    f.write("at: " + str(count) + "\n" )
    f.flush()
    print("at:", count, flush=True)
    count += 1
    time.sleep(0.02)

