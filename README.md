#TFTPd
An RFC 1350 compliant TFTP server written in Go. 

## Build Instructions
Simply run `go run tftp.go`.

### Configuration
Some constants may be changed:
* `mtu`: Adjust to allow higher MTU
* `chunkSize`: Adjust to allow out-of-spec chunk sizes
* `tftpPort`: Sets the port to listen for new connections (ports below 1024 require root privileges)
* `minPort`: Sets the minimum connection port (ports below 1024 require root privileges)
* `maxPort`: Sets the maximum connection port
* `timeout`: Sets the maximum amount of time to wait for a response from a client

## Known Issues
* Files are transfered in-place. A fix could be to prepend a dot to the filename until the transfer is complete in order to hide the file from the user. 
* The server serves files out of the current directory. This could be configurable.

