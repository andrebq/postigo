package ioutil

import "io"

func bgCopy(out io.Writer, in io.Reader, errCh chan error) {
	var err error
	defer func() {
		if err == nil {
			err = io.EOF
		}
		errCh <- err
	}()
	_, err = io.Copy(out, in)
}

func BackgroundCopy(out io.ReadWriter, in io.ReadWriter) <-chan error {
	ch := make(chan error, 2)
	go bgCopy(out, in, ch)
	go bgCopy(in, out, ch)
	return ch
}
