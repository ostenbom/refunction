print('HERE WE ARE -----------------------------------------\n', flush=True)
import socket
#
BUFFER_SIZE = 20  # Normally 1024, but we want fast response
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
print(s, flush=True)
s.bind(('', 5000))
s.listen(1)

while True:
    conn, addr = s.accept()
    print('Connection address:', addr)
    data = conn.recv(BUFFER_SIZE)
    if not data: break
    print("received data:", data)
    conn.send(data)  # echo
    conn.close()
