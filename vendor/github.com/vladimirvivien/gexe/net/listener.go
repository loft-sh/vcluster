package net

import (
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
)

func AddrUsable(address string) error {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return fmt.Errorf("net: address parsing: %s", err)
	}

	lsnr, err := net.Listen(addr.Network(), addr.String())

	if err != nil {
		sysErr, ok := err.(*os.SyscallError)
		if ok && errors.Is(sysErr.Err, syscall.EADDRINUSE) {
			return fmt.Errorf("net: addr in use: %s", sysErr.Err)
		}
		return fmt.Errorf("net: %s", err)
	}

	defer lsnr.Close()
	return nil
}
