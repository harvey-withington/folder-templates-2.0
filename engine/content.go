package foldertemplate

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
)

// Content-processing rules for .ft$ files (spec §6.2 port fixes):
//   - size ceiling instead of unbounded reads,
//   - refuse binary-looking content (NUL in the first 8 KB) instead of corrupting,
//   - preserve a UTF-8 BOM if present,
//   - stream line-wise (tokens are single-line by construction — parameter
//     names are identifiers, so {{$name}} can never span a newline).

var (
	// ErrContentTooLarge: a .ft$ file exceeds Options.ContentSizeLimit.
	ErrContentTooLarge = errors.New("content-replacement file exceeds size limit")
	// ErrBinaryContent: a .ft$ file looks binary and cannot be token-processed.
	ErrBinaryContent = errors.New("content-replacement file looks binary (NUL bytes)")
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

const binarySniffLen = 8 << 10

// processContent streams src → dst applying token replacement per line.
func processContent(src io.Reader, dst io.Writer, bindings []binding) error {
	// Reader buffer must be >= the sniff length or Peek returns ErrBufferFull.
	r := bufio.NewReaderSize(src, binarySniffLen)
	w := bufio.NewWriter(dst)

	head, err := r.Peek(binarySniffLen)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if bytes.IndexByte(head, 0) >= 0 {
		return ErrBinaryContent
	}
	if bytes.HasPrefix(head, utf8BOM) {
		if _, err := r.Discard(len(utf8BOM)); err != nil {
			return err
		}
		if _, err := w.Write(utf8BOM); err != nil {
			return err
		}
	}

	for {
		line, err := r.ReadString('\n')
		if line != "" {
			if _, werr := w.WriteString(applyToContent(line, bindings)); werr != nil {
				return werr
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
	}
	return w.Flush()
}

// processContentFile applies processContent from srcPath to dstPath with the
// size ceiling enforced up front.
func processContentFile(srcPath, dstPath string, bindings []binding, limit int64) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	if info.Size() > limit {
		return fmt.Errorf("%s (%d bytes > %d): %w", srcPath, info.Size(), limit, ErrContentTooLarge)
	}
	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	if err := processContent(in, out, bindings); err != nil {
		out.Close()
		os.Remove(dstPath)
		return fmt.Errorf("%s: %w", srcPath, err)
	}
	return out.Close()
}

// copyFile copies byte-for-byte (binary-safe by construction).
func copyFile(srcPath, dstPath string) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dstPath)
		return err
	}
	return out.Close()
}
