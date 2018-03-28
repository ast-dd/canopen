package sdo

import (
	"bytes"
	"github.com/ast-dd/can"
	"github.com/ast-dd/canopen"
	"reflect"
	"testing"
	"syscall"
)

type uploadReadWriteCloser struct {
	writeFd int
}

func (rw *uploadReadWriteCloser) Read(b []byte) (n int, err error) {
	panic("Read() shouldn't be called anymore")
}

// write uploadFrame instead of given data
func (rw *uploadReadWriteCloser) Write(b []byte) (n int, err error) {
	if response, err := can.Marshal(uploadFrame); err == nil {
		err = syscall.Sendmsg(rw.writeFd, response, nil, nil, 0)
		n = len(b)
	}

	return
}

func (rw *uploadReadWriteCloser) Close() error { return nil }

var uploadFrame = can.Frame{
	ID:     0x580,
	Length: 0x8,
	Flags:  0x0,
	Res0:   0x0,
	Res1:   0x0,
	Data: [can.MaxFrameDataLength]uint8{
		0x42, // 0100 0010 (= expedited)
		0xBB, 0xAA,
		0xCC,
		0x1, 0x2, 0x3, 0x4},
}

func HandleUpload(b []byte) (n int, err error) {
	var buf bytes.Buffer
	if b, err = can.Marshal(uploadFrame); err == nil {
		n, err = buf.Write(b)
	}

	return
}

func TestExpeditedUpload(t *testing.T) {
	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		panic(err)
	}
	rwc := can.NewReadWriteCloser(&uploadReadWriteCloser{writeFd: pair[0]}, pair[1])
	bus := can.NewBus(rwc)

	go bus.ConnectAndPublish()
	defer bus.Disconnect()

	object := canopen.NewObjectIndex(0xAABB, 0xCC)
	// Read values for object
	upload := Upload{
		ObjectIndex:   object,
		RequestCobID:  0x600,
		ResponseCobID: uint16(uploadFrame.ID),
	}

	b, err := upload.Do(bus)
	if err != nil {
		t.Fatal(err)
	}

	if is, want := b, []byte{0x1, 0x2, 0x3, 0x4}; reflect.DeepEqual(is, want) != true {
		t.Fatalf("is=%v want=%v", is, want)
	}
}
