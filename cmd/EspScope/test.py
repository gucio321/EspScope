#!/usr/bin/env python3
import struct,socket, time

for i in range(1,10):
    pkt = struct.pack(
        "<IIIH4H",
        0x53434F50,   # magic
        i,            # seq
        50000,        # sample rate
        4,            # sample count
        100,200,300,400
    )

    s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM)
    s.sendto(pkt,("127.0.0.1",5000))
    time.sleep(1)
