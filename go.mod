module github.com/Fenway-snx/synthetix-mcp

go 1.26.1

// cockroachdb/errors (transitive via ethereum/go-ethereum → pebble) pins
// an ancient umbrella google.golang.org/genproto that bundles googleapis/rpc/*,
// conflicting with the split google.golang.org/genproto/googleapis/rpc module.
// Exclude it so MVS picks a modern version where those packages have been
// extracted. No cockroachdb/errors release currently avoids this pin.
exclude google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1

require (
	github.com/ethereum/go-ethereum v1.17.2
	github.com/go-viper/mapstructure/v2 v2.5.0
	github.com/google/jsonschema-go v0.4.2
	github.com/modelcontextprotocol/go-sdk v1.4.1
	github.com/rs/zerolog v1.34.0
	github.com/shopspring/decimal v1.4.0
	github.com/spf13/cast v1.10.0
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	github.com/synthetixio/synthetix-go v0.1.0
	golang.org/x/sync v0.20.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/ProjectZKM/Ziren/crates/go-runtime/zkvm_runtime v0.0.0-20251001021608-1fe7b43fc4d6 // indirect
	github.com/bits-and-blooms/bitset v1.20.0 // indirect
	github.com/consensys/gnark-crypto v0.18.1 // indirect
	github.com/crate-crypto/go-eth-kzg v1.5.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/ethereum/c-kzg-4844/v2 v2.1.6 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/supranational/blst v0.3.16 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/oauth2 v0.34.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Local sibling checkout of the published SDK while
// github.com/Fenway-snx/synthetix-go-sdk stays private. Drop this and
// pin a real semver tag once the SDK repo is public (or once GOPRIVATE
// + git creds are set up on every consumer).
replace github.com/synthetixio/synthetix-go => /Users/bcelermajer/synthetix-go
