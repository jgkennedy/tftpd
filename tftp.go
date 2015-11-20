package main

import (
    "fmt"
    "net"
    "os"
    "encoding/binary"
    "strings"
    "math/rand"
    "time"
    "strconv"
)

const (
    RRQ = 1
    WRQ = 2
    DATA = 3
    ACK = 4
    ERROR = 5
)
const mtu = 1500  // Adjust to allow higher MTU
const chunkSize = 512  // Adjust to allow out-of-spec chunk sizes
const minPort = 1024
const maxPort = 49152
const timeout = 2 * time.Minute

func checkError(err error) {
    if err != nil {
        fmt.Println("Error:" , err)
        os.Exit(1)
    }
}

func handleReadMsg(filename string, remoteAddr *net.UDPAddr) {
    // Set up connection
    listenPort := rand.Intn(maxPort-minPort) + minPort // Random int 1024-49151 (valid unprivileged ports)
    localAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+strconv.Itoa(listenPort))
    checkError(err)
    conn, err := net.DialUDP("udp", localAddr, remoteAddr)
    checkError(err)
    conn.SetDeadline(time.Now().Add(timeout))
    defer conn.Close()

    // Open requested file
    f, err := os.Open(filename)
    checkError(err)
    buff := make([]byte, chunkSize)
    var block uint16 = 1
    for n, err := f.Read(buff); n != 0; n, err = f.Read(buff) {
        // Convert opcode and block to big endian
        opcodeBig := make([]byte, 2)
        binary.BigEndian.PutUint16(opcodeBig, DATA)
        blockBig := make([]byte, 2)
        binary.BigEndian.PutUint16(blockBig, block)

        // Assemble message and send
        message := append(opcodeBig, blockBig...)
        message = append(message, buff[:n]...)
        _, err = conn.Write(message)
        checkError(err)
        fmt.Println("Sent", n, "bytes from", listenPort)

        // Receive ACK
        data := make([]byte, mtu)
        _, _, err = conn.ReadFromUDP(data)
        checkError(err)  // Possibly handle timeouts gracefully here

        // Verify block
        ackBlock := uint16(binary.BigEndian.Uint16(data[2:4]))
        fmt.Println("Recv ACK", ackBlock, "from", remoteAddr)
        if ackBlock != block {
            fmt.Println("Error: ACK mismatch")
            os.Exit(1)
        } else {
            block++
        }
    }
    f.Close()
}

func handleWriteMsg(filename string, remoteAddr *net.UDPAddr) {
    // Set up connection
    listenPort := rand.Intn(maxPort-minPort) + minPort // Random int 1024-49151 (valid unprivileged ports)
    localAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+strconv.Itoa(listenPort))
    checkError(err)
    conn, err := net.DialUDP("udp", localAddr, remoteAddr)
    checkError(err)
    conn.SetDeadline(time.Now().Add(timeout))
    defer conn.Close()

    // Open requested file
    f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0664)
    checkError(err)

    var opcodeBig []byte = make([]byte, 2)
    var blockBig []byte = make([]byte, 2)
    var block uint16 = 0
    endOfMessage := false
    for !endOfMessage {
        // Convert opcode and block to big endian
        binary.BigEndian.PutUint16(opcodeBig, ACK)
        binary.BigEndian.PutUint16(blockBig, block)

        // Assemble message and send
        message := append(opcodeBig, blockBig...)
        _, err = conn.Write(message)
        checkError(err)
        fmt.Println("Sent ACK", block, "to", remoteAddr)
        block++

        // Receive DATA
        data := make([]byte, mtu)
        n, _, err := conn.ReadFromUDP(data)
        checkError(err)
        if n-4 < 512 {
            endOfMessage = true
        }

        // Verify block
        ackBlock := uint16(binary.BigEndian.Uint16(data[2:4]))
        fmt.Println("Recv DATA", ackBlock, "from", listenPort)
        if ackBlock != block {
            fmt.Println("Error: ACK mismatch")
            os.Exit(1)
        }

        // Write received data to file
        _, err = f.Write(data[4:n])
        checkError(err)
    }
    // Send final ACK message
    binary.BigEndian.PutUint16(blockBig, block)
    message := append(opcodeBig, blockBig...)
    _, err = conn.Write(message)
    checkError(err)
    fmt.Println("Sent final ACK", block, "to", remoteAddr)
    f.Close()
}

func routePacket(data []byte, n int, addr *net.UDPAddr) {
    opCode := binary.BigEndian.Uint16(data[:2])
    switch opCode {
    case RRQ:
        // Extract payload section from message
        payload := string(data[2:])
        // Find the zero byte separating the two fields
        zeroIdx := strings.Index(payload, "\x00")
        handleReadMsg(payload[:zeroIdx], addr)
    case WRQ:
        // Extract payload section from message
        payload := string(data[2:])
        // Find the zero byte separating the two fields
        zeroIdx := strings.Index(payload, "\x00")
        handleWriteMsg(payload[:zeroIdx], addr)
    case DATA:
        fmt.Println("Unexpected DATA message")
        os.Exit(1)
    case ACK:
        fmt.Println("Unexpected ACK message")
        os.Exit(1)
    case ERROR:
        code := int16(binary.BigEndian.Uint16(data[2:4]))
        message := string(data[4:n-1])
        fmt.Println("ERROR!", code, message)
    default:
        fmt.Println("Undefined message type", opCode)
        os.Exit(1)
    }
}

func recvData(conn *net.UDPConn) {
    for {
        data := make([]byte, mtu)
        n, addr, err := conn.ReadFromUDP(data)
        fmt.Println("Recv packet of size", n, "from", addr)
        checkError(err)
        go routePacket(data, n, addr)
    }
}

func main() {
    bindAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:6969")
    checkError(err)
    conn, err := net.ListenUDP("udp", bindAddr)
    checkError(err)
    defer conn.Close()
    recvData(conn)
}
