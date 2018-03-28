package sdo

import (
	"github.com/ast-dd/can"
	"github.com/ast-dd/canopen"
	"log"
	"testing"
	"syscall"
)

type downloadReadWriteCloser struct {
	writeFd int
}

func (rw *downloadReadWriteCloser) Read(b []byte) (n int, err error) {
	panic("Read() shouldn't be called anymore")
}

// write expected response instead of given data
func (rw *downloadReadWriteCloser) Write(b []byte) (n int, err error) {
	var (
		sendFrm can.Frame
		respFrm can.Frame
	)
	if err := can.Unmarshal(b, &sendFrm); err != nil {
		return 0, err
	}

	switch sendFrm.Data[0] & TransferMaskCommandSpecifier {
	case ClientInitiateDownload:
		respFrm = downloadInitiateFrame
	case ClientSegmentDownload:
		respFrm = downloadSegmentFrame
	default:
		log.Fatal("Unknown command")
		break
	}
	if response, err := can.Marshal(respFrm); err == nil {
		err = syscall.Sendmsg(rw.writeFd, response, nil, nil, 0)
		n = len(b)
	}

	return
}

func (rw *downloadReadWriteCloser) Close() error { return nil }

var downloadInitiateFrame = can.Frame{
	ID:     0x580,
	Length: 0x8,
	Flags:  0x0,
	Res0:   0x0,
	Res1:   0x0,
	Data: [can.MaxFrameDataLength]uint8{
		ServerInitiateDownload,
		0xBB, 0xAA,
		0xCC,
		0x0, 0x0, 0x0, 0x0},
}

var downloadSegmentFrame = can.Frame{
	ID:     0x580,
	Length: 0x8,
	Flags:  0x0,
	Res0:   0x0,
	Res1:   0x0,
	Data: [can.MaxFrameDataLength]uint8{
		ServerSegmentDownload,
		0xBB, 0xAA,
		0xCC,
		0x0, 0x0, 0x0, 0x0},
}

func TestDownload(t *testing.T) {
	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		log.Fatal(err)
	}
	rwc := can.NewReadWriteCloser(&downloadReadWriteCloser{writeFd: pair[0]}, pair[1])
	bus := can.NewBus(rwc)

	go bus.ConnectAndPublish()
	defer bus.Disconnect()

	object := canopen.NewObjectIndex(0xAABB, 0xCC)
	download := Download{
		ObjectIndex:   object,
		RequestCobID:  0x600,
		ResponseCobID: uint16(downloadSegmentFrame.ID),
		// 0x2 + WRITE (String) + 0x91 (Datatype)
		Data: []byte{0x2, 0x57, 0x52, 0x49, 0x54, 0x45, 0x91},
	}

	err = download.Do(bus)
	if err != nil {
		t.Fatal(err)
	}
}
