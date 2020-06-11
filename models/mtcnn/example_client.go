package mtcnn

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	conn, _ := net.Dial("tcp", "localhost:3333")
	for {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		fmt.Println(text)
		conn.Write([]byte(text))
		buffer := make([]byte, 1024)
		n, _ := conn.Read(buffer)
		fmt.Print("result: " + string(buffer[:n]))
	}
}
