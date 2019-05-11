import base64
import sys

with open(sys.argv[1], 'rb') as f:
    content = f.read()
    print(content)
    print()
    print(base64.b64encode(content))
