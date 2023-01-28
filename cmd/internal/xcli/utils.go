package xcli

import "github.com/urfave/cli/v2"

func JoinFlags(flags ...[]cli.Flag) (ret []cli.Flag) {
	total := 0
	for _, slice := range flags {
		total += len(slice)
	}

	ret = make([]cli.Flag, total)
	offset := 0
	for _, slice := range flags {
		copy(ret[offset:], slice)
		offset += len(slice)
	}
	return
}
