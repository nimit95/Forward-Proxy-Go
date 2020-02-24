package main

import (
	"fmt"
	"net"
	"io"
	"regexp"
)

func server() {
  // listen on a port
  ln, err := net.Listen("tcp", ":8118")
  if err != nil {
    fmt.Println(err)
    return
  }
  for {
    // accept a connection
    c, err := ln.Accept()
    if err != nil {
      fmt.Println(err)
      continue
    }
    // handle the connection
    go handleProxyConnection(c)
  }
}

func handleProxyConnection(downstreamConn net.Conn) {

	//Read First Packet to get host
	b := make([]byte, 4096)

	_, errorDownstream := downstreamConn.Read(b)
	if errorDownstream != nil  {
		fmt.Println("DOWNTREAM ERROR ", errorDownstream)
		downstreamConn.Close()
		return
	}

	upstreamHost := getHostFromHeader(b)

	upstreamConn, err := net.Dial("tcp", upstreamHost)

	if err != nil {
		fmt.Println("UPSTREAM ERROR ", err)
		downstreamConn.Close()
		return
	}

	upstreamChan, upstreamErr := chanFromConn(upstreamConn)
	downstreamChan, downstreamErr := chanFromConn(downstreamConn)
	
	upstreamConn.Write(b)
	for {
		select {
		case b1 := <-upstreamChan:
				if b1 == nil {return;}
				downstreamConn.Write(b1)
		case b2 := <-downstreamChan:
				if b2 == nil {return;}
				upstreamConn.Write(b2)
				
		case err1 := <-downstreamErr:
			if(err1 != nil) {
				if err1 == io.EOF {
					// downstreamConn.Write(io.EOF)
					// close(upstreamChan)
					// close(downstreamChan)
					upstreamConn.Close()
					downstreamConn.Close()
					// return;
				}
			}
		case err2 := <-upstreamErr:
			if(err2 != nil) {
				// fmt.Println("error upstream", err2)
				// close(upstreamChan)
				// close(downstreamChan)
				upstreamConn.Close()
				downstreamConn.Close()
			}
		}
	}
}

func getHostFromHeader(header []byte ) string {
	re := regexp.MustCompile(`Host: (.+)\r\n`)
	submatchall := re.FindStringSubmatch(string(header))

	//TODO check if hoat has port, if not append 80
	return submatchall[1] + ":80"
}

func chanFromConn(conn net.Conn) (chan []byte, chan error) {
	c := make(chan []byte)
	errorChan := make(chan error)
	go func() {
			b := make([]byte, 1024)
      
			for {
					n, err := conn.Read(b)
					if n > 0 {
							res := make([]byte, n)
							// Copy the buffer so it doesn't get changed while read by the recipient.
							copy(res, b[:n])
							c <- res
					}
					if err != nil {
							errorChan <- err
							conn.Close()
							close(errorChan)
							close(c)
							break
					}
			}
	}()

	return c, errorChan
}

func main() {
	server()
}