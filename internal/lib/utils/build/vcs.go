package build

var (
	buildCommit string
)

// Obtains the calling process' build commit, if available; otherwise
// obtains the empty string.
//
// Note:
// This requires that the go build command includes
//
//	`-ldflags "-X github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/build.buildCommit=${BUILD_COMMIT}"`
//
// where `BUILD_COMMIT` is obtained at build time, from a command such as
//
//	`BUILD_COMMIT=$(git rev-parse HEAD)`
func BuildCommit() string {
	return buildCommit
}
