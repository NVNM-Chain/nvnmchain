package swagger

import (
	_ "github.com/NVNM-Chain/nvnmchain/client/docs/statik" // Import NVNM Chain statik
	"github.com/rakyll/statik/fs"
)

// https://github.com/rakyll/statik/issues/56

// FS is the NVNM Chain swagger filesystem
var FS, _ = fs.New()
