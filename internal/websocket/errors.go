package websocket

import "errors"

const (
	// ErrorConnection - error when connecting
	ErrorConnection = iota + 100
	// ErrorRead - error when reading message
	ErrorRead
	// ErrorKeepalive - error while sending ping message
	ErrorKeepalive
	// ErrorCloseConnection - error closing connection
	ErrorCloseConnection

	// ErrorStartServer - error when starting ws server
	ErrorStartServer

	// ErrorNilChannel - error rwhen data channel is nil
	ErrorNilChannel
)

// Error - error structure
type Error struct {
	code    int
	err     error
	message string
}

// Error - to implement error interface
func (e Error) Error() string {
	return e.message
}

// NewError - Error constructor
func NewError(t int, err error) error {
	return Error{
		code:    t,
		err:     err,
		message: err.Error(),
	}
}

// NewReadError - NewError decorator
func NewReadError(err error) error {
	return NewError(ErrorRead, err)
}

// NewChannelNilError - NewError decorator
func NewChannelNilError() error {
	err := errors.New("Datachannel is nil")
	return NewError(ErrorNilChannel, err)
}

// NewKeepaliveError - NewError decorator
func NewKeepaliveError(err error) error {
	return NewError(ErrorKeepalive, err)
}

// NewConnectionError - NewError decorator
func NewConnectionError(err error) error {
	return NewError(ErrorConnection, err)
}

// NewCloseConnectionError - NewError decorator
func NewCloseConnectionError(err error) error {
	return NewError(ErrorCloseConnection, err)
}

// NewServerStartError - NewError decorator
func NewServerStartError(err error) error {
	return NewError(ErrorStartServer, err)
}
