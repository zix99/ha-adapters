package amcrest

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

func (s *AmcrestDevice) DownloadFile(path string) (io.ReadCloser, error) {
	// http://admin:password@ip/cgi-bin/RPC_Loadfile/mnt/sd/2021-10-04/001/dav/10/10.56.56-10.57.44[M][0@0][0].mp4
	return s.requestStream("/cgi-bin/RPC_Loadfile" + path)
}

// Called with `path` from a `NewFile` event
func (s *AmcrestDevice) DownloadFileTo(path, to string) error {
	stream, err := s.DownloadFile(path)
	if err != nil {
		return err
	}
	defer stream.Close()

	f, err := os.Create(to)
	if err != nil {
		return err
	}

	var total uint64

	buf := make([]byte, 1024*4)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			total += uint64(n)
			f.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	logrus.Debugf("Wrote %d bytes to %s", total, to)

	return nil
}
